package filtermanager

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"blockads-desktop/internal/filters"
	"blockads-desktop/internal/storage"
)

var (
	ErrAlreadyUpdating        = errors.New("subscription update already in progress")
	ErrHTMLResponse           = errors.New("server returned an HTML page instead of filter list")
	ErrEmptyFilterList        = errors.New("filter list contains zero valid rules")
	ErrExcessiveWarnings      = errors.New("filter list rejected due to excessive malformed entries")
	ErrUnsolicited304Recovery = errors.New("server returned 304 on unconditional recovery request")
)

// MatcherReloader exposes atomic matcher swapping.
type MatcherReloader interface {
	SetMatcher(m *filters.Matcher)
	Matcher() *filters.Matcher
}

// UpdateResult provides structured diagnostics of an update attempt.
type UpdateResult struct {
	SubscriptionID string        `json:"subscription_id"`
	StatusCode     int           `json:"status_code"`
	Status         string        `json:"status"` // "updated", "not_modified", "unchanged", "error"
	Checksum       string        `json:"checksum"`
	RuleCount      int           `json:"rule_count"`
	Warnings       int           `json:"warnings"`
	Duration       time.Duration `json:"duration"`
	BytesReceived  int64         `json:"bytes_received"`
	ErrorMessage   string        `json:"error_message,omitempty"`
}

// FilterUpdater coordinates the safe download, validation, persistence, and matcher reload of filter lists.
type FilterUpdater struct {
	store      *storage.Storage
	downloader *Downloader
	reloader   MatcherReloader

	mu       sync.Mutex
	inflight map[string]struct{}
}

// NewFilterUpdater creates a new FilterUpdater.
func NewFilterUpdater(store *storage.Storage, downloader *Downloader, reloader MatcherReloader) *FilterUpdater {
	return &FilterUpdater{
		store:      store,
		downloader: downloader,
		reloader:   reloader,
		inflight:   make(map[string]struct{}),
	}
}

// isHTMLContent inspects the leading non-whitespace bytes to detect HTML or error pages.
func isHTMLContent(body []byte) bool {
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) > 512 {
		trimmed = trimmed[:512]
	}
	lower := strings.ToLower(string(trimmed))
	return strings.HasPrefix(lower, "<!doctype html") ||
		strings.HasPrefix(lower, "<html") ||
		strings.HasPrefix(lower, "<head") ||
		strings.Contains(lower, "<body") ||
		strings.Contains(lower, "<title>404") ||
		strings.Contains(lower, "<title>500") ||
		strings.Contains(lower, "<title>error")
}

// RefreshSubscription updates a single subscription, supporting HTTP 304 conditional updates and atomic rollback on failure.
func (u *FilterUpdater) RefreshSubscription(ctx context.Context, subID string) (*UpdateResult, error) {
	// 1. In-flight check to prevent overlapping updates for the same subscription
	u.mu.Lock()
	if _, active := u.inflight[subID]; active {
		u.mu.Unlock()
		return nil, ErrAlreadyUpdating
	}
	u.inflight[subID] = struct{}{}
	u.mu.Unlock()

	defer func() {
		u.mu.Lock()
		delete(u.inflight, subID)
		u.mu.Unlock()
	}()

	// 2. Load subscription record
	sub, err := u.store.GetFilterSubscription(ctx, subID)
	if err != nil {
		return nil, fmt.Errorf("failed to load subscription %s: %w", subID, err)
	}

	// 3. Prepare download request with conditional validators if a valid snapshot exists
	req := DownloadRequest{
		URL: sub.SourceURL,
	}

	existingContent, storedChecksum, snapErr := u.store.GetFilterSnapshot(ctx, subID)
	hasValidSnapshot := (snapErr == nil && storedChecksum != "" && storedChecksum == sub.ContentChecksum)
	if hasValidSnapshot {
		req.ETag = sub.ETag
		req.LastModified = sub.LastModified
	}

	start := time.Now()
	resp, dlErr := u.downloader.Download(ctx, req)
	duration := time.Since(start)

	if dlErr != nil {
		_ = u.store.RecordFilterFetchError(ctx, subID, dlErr.Error())
		return &UpdateResult{
			SubscriptionID: subID,
			Status:         "error",
			Duration:       duration,
			ErrorMessage:   dlErr.Error(),
		}, dlErr
	}

	// 4. Handle HTTP 304 Not Modified
	if resp.StatusCode == http.StatusNotModified || resp.NotModified {
		if !hasValidSnapshot {
			// Missing or corrupt snapshot: retry once without conditional headers
			req.ETag = ""
			req.LastModified = ""
			recResp, recErr := u.downloader.Download(ctx, req)
			if recErr != nil {
				errMsg := fmt.Sprintf("unconditional recovery failed after 304: %v", recErr)
				_ = u.store.RecordFilterFetchError(ctx, subID, errMsg)
				return &UpdateResult{
					SubscriptionID: subID,
					StatusCode:     http.StatusNotModified,
					Status:         "error",
					Duration:       time.Since(start),
					ErrorMessage:   errMsg,
				}, recErr
			}
			if recResp.StatusCode == http.StatusNotModified || recResp.NotModified {
				_ = u.store.RecordFilterFetchError(ctx, subID, ErrUnsolicited304Recovery.Error())
				return &UpdateResult{
					SubscriptionID: subID,
					StatusCode:     http.StatusNotModified,
					Status:         "error",
					Duration:       time.Since(start),
					ErrorMessage:   ErrUnsolicited304Recovery.Error(),
				}, ErrUnsolicited304Recovery
			}
			resp = recResp
		} else {
			// Valid snapshot verified: retain content, checksum, and active rules.
			// Update validators per RFC 9110.
			etag := resp.ETag
			if etag == "" {
				etag = sub.ETag
			}
			lastMod := resp.LastModified
			if lastMod == "" {
				lastMod = sub.LastModified
			}

			notModMeta := storage.SnapshotMeta{
				ETag:         etag,
				LastModified: lastMod,
			}

			if err := u.store.RecordFilterFetchNotModified(ctx, subID, notModMeta); err != nil {
				return nil, fmt.Errorf("failed to record 304 metadata: %w", err)
			}

			return &UpdateResult{
				SubscriptionID: subID,
				StatusCode:     http.StatusNotModified,
				Status:         "not_modified",
				Checksum:       sub.ContentChecksum,
				RuleCount:      sub.RuleCount,
				Duration:       duration,
				BytesReceived:  resp.ContentLength,
			}, nil
		}
	}

	// 5. Handle HTTP 200 OK
	if resp.StatusCode != http.StatusOK {
		errMsg := fmt.Sprintf("HTTP %d %s", resp.StatusCode, http.StatusText(resp.StatusCode))
		_ = u.store.RecordFilterFetchError(ctx, subID, errMsg)
		return &UpdateResult{
			SubscriptionID: subID,
			StatusCode:     resp.StatusCode,
			Status:         "error",
			Duration:       duration,
			BytesReceived:  int64(len(resp.Body)),
			ErrorMessage:   errMsg,
		}, errors.New(errMsg)
	}

	// Reject HTML error pages
	if isHTMLContent(resp.Body) {
		_ = u.store.RecordFilterFetchError(ctx, subID, ErrHTMLResponse.Error())
		return &UpdateResult{
			SubscriptionID: subID,
			StatusCode:     http.StatusOK,
			Status:         "error",
			Duration:       duration,
			BytesReceived:  int64(len(resp.Body)),
			ErrorMessage:   ErrHTMLResponse.Error(),
		}, ErrHTMLResponse
	}

	// Compute checksum
	hasher := sha256.New()
	hasher.Write(resp.Body)
	checksum := hex.EncodeToString(hasher.Sum(nil))

	// Parse content using existing supported formats
	parseFormat := filters.FormatAuto
	if strings.EqualFold(sub.Format, "hosts") {
		parseFormat = filters.FormatHosts
	} else if strings.EqualFold(sub.Format, "plain") || strings.EqualFold(sub.Format, "domains") {
		parseFormat = filters.FormatPlain
	} else if strings.EqualFold(sub.Format, "adblock") {
		parseFormat = filters.FormatAdblock
	}

	parseRes, parseErr := filters.ParseFilterList(bytes.NewReader(resp.Body), parseFormat)
	if parseErr != nil {
		_ = u.store.RecordFilterFetchError(ctx, subID, fmt.Sprintf("parse failed: %v", parseErr))
		return &UpdateResult{
			SubscriptionID: subID,
			StatusCode:     http.StatusOK,
			Status:         "error",
			Duration:       duration,
			BytesReceived:  int64(len(resp.Body)),
			ErrorMessage:   parseErr.Error(),
		}, parseErr
	}

	// Quality checks: non-empty rules and reasonable error ratio
	if len(parseRes.Rules) == 0 {
		_ = u.store.RecordFilterFetchError(ctx, subID, ErrEmptyFilterList.Error())
		return &UpdateResult{
			SubscriptionID: subID,
			StatusCode:     http.StatusOK,
			Status:         "error",
			Duration:       duration,
			BytesReceived:  int64(len(resp.Body)),
			ErrorMessage:   ErrEmptyFilterList.Error(),
		}, ErrEmptyFilterList
	}

	if parseRes.Stats.WarningCount > 100 && parseRes.Stats.AcceptedCount < parseRes.Stats.WarningCount/4 {
		_ = u.store.RecordFilterFetchError(ctx, subID, ErrExcessiveWarnings.Error())
		return &UpdateResult{
			SubscriptionID: subID,
			StatusCode:     http.StatusOK,
			Status:         "error",
			Duration:       duration,
			BytesReceived:  int64(len(resp.Body)),
			ErrorMessage:   ErrExcessiveWarnings.Error(),
		}, ErrExcessiveWarnings
	}

	// Check if content checksum is unchanged
	if hasValidSnapshot && existingContent == string(resp.Body) && sub.ContentChecksum == checksum {
		// Content unchanged, update validators without rebuilding matcher
		meta := storage.SnapshotMeta{
			ETag:         resp.ETag,
			LastModified: resp.LastModified,
		}
		if meta.ETag == "" {
			meta.ETag = sub.ETag
		}
		if meta.LastModified == "" {
			meta.LastModified = sub.LastModified
		}
		if err := u.store.RecordFilterFetchNotModified(ctx, subID, meta); err != nil {
			return nil, fmt.Errorf("failed to record unchanged metadata: %w", err)
		}
		return &UpdateResult{
			SubscriptionID: subID,
			StatusCode:     http.StatusOK,
			Status:         "unchanged",
			Checksum:       checksum,
			RuleCount:      parseRes.Stats.AcceptedCount,
			Warnings:       len(parseRes.Warnings),
			Duration:       duration,
			BytesReceived:  int64(len(resp.Body)),
		}, nil
	}

	// 6. Save new snapshot and metadata atomically in SQLite
	meta := storage.SnapshotMeta{
		ETag:         resp.ETag,
		LastModified: resp.LastModified,
	}

	if _, err := u.store.SaveFilterSnapshot(ctx, subID, string(resp.Body), parseRes.Stats.AcceptedCount, meta); err != nil {
		_ = u.store.RecordFilterFetchError(ctx, subID, fmt.Sprintf("failed to save snapshot: %v", err))
		return &UpdateResult{
			SubscriptionID: subID,
			StatusCode:     http.StatusOK,
			Status:         "error",
			Duration:       duration,
			BytesReceived:  int64(len(resp.Body)),
			ErrorMessage:   err.Error(),
		}, err
	}

	// 7. Reconcile active state and hot-reload candidate matcher
	activeState, reconcileErr := storage.ReconcileActiveState(ctx, u.store)
	if reconcileErr != nil {
		_ = u.store.RecordFilterFetchError(ctx, subID, fmt.Sprintf("reconcile failed: %v", reconcileErr))
		return &UpdateResult{
			SubscriptionID: subID,
			StatusCode:     http.StatusOK,
			Status:         "error",
			Duration:       duration,
			BytesReceived:  int64(len(resp.Body)),
			ErrorMessage:   reconcileErr.Error(),
		}, reconcileErr
	}

	if u.reloader != nil && activeState.Matcher != nil {
		u.reloader.SetMatcher(activeState.Matcher)
	}

	return &UpdateResult{
		SubscriptionID: subID,
		StatusCode:     http.StatusOK,
		Status:         "updated",
		Checksum:       checksum,
		RuleCount:      parseRes.Stats.AcceptedCount,
		Warnings:       len(parseRes.Warnings),
		Duration:       duration,
		BytesReceived:  int64(len(resp.Body)),
	}, nil
}

// ReconcileMatcher forces a rebuild and reload of the active matcher from storage.
func (u *FilterUpdater) ReconcileMatcher(ctx context.Context) error {
	activeState, err := storage.ReconcileActiveState(ctx, u.store)
	if err != nil {
		return fmt.Errorf("reconcile active state failed: %w", err)
	}
	if u.reloader != nil && activeState.Matcher != nil {
		u.reloader.SetMatcher(activeState.Matcher)
	}
	return nil
}
