package filtermanager

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"blockads-desktop/internal/storage"
)

var (
	ErrSchedulerAlreadyRunning = errors.New("scheduler is already running")
	ErrSchedulerNotRunning     = errors.New("scheduler is not running")
)

const (
	DefaultInterval       = 24 * time.Hour
	DefaultMaxConcurrency = 2
	DefaultJitterFraction = 0.1
)

// SchedulerConfig defines configuration parameters for the background filter list update scheduler.
type SchedulerConfig struct {
	Interval       time.Duration
	MaxConcurrency int
	JitterFraction float64
}

// Scheduler runs periodic background refreshes with bounded concurrency, jitter, and graceful cancellation.
type Scheduler struct {
	updater     *FilterUpdater
	store       *storage.Storage
	interval    time.Duration
	concurrency int
	jitter      float64

	mu       sync.Mutex
	running  bool
	cancel   context.CancelFunc
	stopDone chan struct{}
}

// NewScheduler creates a configured Scheduler. The scheduler is NOT started automatically.
func NewScheduler(updater *FilterUpdater, store *storage.Storage, cfg SchedulerConfig) *Scheduler {
	interval := cfg.Interval
	if interval <= 0 {
		interval = DefaultInterval
	}

	concurrency := cfg.MaxConcurrency
	if concurrency <= 0 {
		concurrency = DefaultMaxConcurrency
	}

	jitter := cfg.JitterFraction
	if jitter <= 0 || jitter >= 1.0 {
		jitter = DefaultJitterFraction
	}

	return &Scheduler{
		updater:     updater,
		store:       store,
		interval:    interval,
		concurrency: concurrency,
		jitter:      jitter,
	}
}

// Start launches the background scheduler loop. It is safe to call only when stopped.
func (s *Scheduler) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return ErrSchedulerAlreadyRunning
	}

	subCtx, cancel := context.WithCancel(ctx)
	s.cancel = cancel
	s.stopDone = make(chan struct{})
	s.running = true

	go s.runLoop(subCtx)
	return nil
}

// Stop gracefully shuts down the background scheduler loop.
func (s *Scheduler) Stop() error {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return ErrSchedulerNotRunning
	}
	cancel := s.cancel
	done := s.stopDone
	s.running = false
	s.mu.Unlock()

	cancel()

	select {
	case <-done:
		return nil
	case <-time.After(5 * time.Second):
		return errors.New("timed out waiting for filter scheduler shutdown")
	}
}

// IsRunning reports whether the background scheduler is active.
func (s *Scheduler) IsRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running
}

// jitteredInterval calculates the next run duration with pseudo-random jitter.
func (s *Scheduler) jitteredInterval() time.Duration {
	base := float64(s.interval)
	delta := base * s.jitter
	// Random float between -delta and +delta
	offset := (rand.Float64()*2 - 1) * delta
	dur := time.Duration(base + offset)
	if dur <= 0 {
		dur = time.Minute
	}
	return dur
}

// runLoop is the internal background goroutine that triggers periodic updates.
func (s *Scheduler) runLoop(ctx context.Context) {
	defer close(s.stopDone)

	for {
		interval := s.jitteredInterval()
		timer := time.NewTimer(interval)

		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
			// Execute scheduled refresh across enabled subscriptions
			_, _ = s.RefreshAll(ctx)
		}
	}
}

// RefreshAll refreshes all enabled filter subscriptions with bounded concurrency.
func (s *Scheduler) RefreshAll(ctx context.Context) ([]UpdateResult, error) {
	subs, err := s.store.GetFilterSubscriptions(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list filter subscriptions: %w", err)
	}

	var enabledSubs []storage.FilterSubscription
	for _, sub := range subs {
		if sub.Enabled {
			enabledSubs = append(enabledSubs, sub)
		}
	}

	if len(enabledSubs) == 0 {
		return nil, nil
	}

	sem := make(chan struct{}, s.concurrency)
	results := make([]UpdateResult, len(enabledSubs))
	var wg sync.WaitGroup

	for i, sub := range enabledSubs {
		wg.Add(1)
		go func(idx int, subID string) {
			defer wg.Done()

			select {
			case sem <- struct{}{}:
			case <-ctx.Done():
				results[idx] = UpdateResult{
					SubscriptionID: subID,
					Status:         "error",
					ErrorMessage:   ctx.Err().Error(),
				}
				return
			}
			defer func() { <-sem }()

			// Perform update with bounded retry on transient failure
			res, err := s.refreshWithTransientRetry(ctx, subID)
			if res != nil {
				results[idx] = *res
			} else if err != nil {
				results[idx] = UpdateResult{
					SubscriptionID: subID,
					Status:         "error",
					ErrorMessage:   err.Error(),
				}
			}
		}(i, sub.ID)
	}

	wg.Wait()
	return results, nil
}

// isPermanentError checks whether an update error is non-retryable.
func isPermanentError(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, ErrHTMLResponse) ||
		errors.Is(err, ErrEmptyFilterList) ||
		errors.Is(err, ErrExcessiveWarnings) ||
		errors.Is(err, ErrSSRFBlocked) ||
		errors.Is(err, ErrInsecureScheme) ||
		errors.Is(err, ErrRedirectDowngrade) ||
		errors.Is(err, ErrTooManyRedirects) ||
		errors.Is(err, ErrAlreadyUpdating)
}

// refreshWithTransientRetry attempts a refresh, retrying once after a short backoff for transient network issues.
func (s *Scheduler) refreshWithTransientRetry(ctx context.Context, subID string) (*UpdateResult, error) {
	res, err := s.updater.RefreshSubscription(ctx, subID)
	if err == nil {
		return res, nil
	}

	// Do not retry permanent validation or security errors
	if isPermanentError(err) {
		return res, err
	}

	// Wait briefly before 1 bounded retry
	select {
	case <-ctx.Done():
		return res, ctx.Err()
	case <-time.After(200 * time.Millisecond):
	}

	return s.updater.RefreshSubscription(ctx, subID)
}
