package filtermanager_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"blockads-desktop/internal/filtermanager"
	"blockads-desktop/internal/filters"
	"blockads-desktop/internal/storage"
)

func TestScheduler_Lifecycle(t *testing.T) {
	store := setupTestStore(t)
	dl := filtermanager.NewDownloader(filtermanager.DownloaderConfig{AllowPrivateNetworks: true})
	reloader := &mockReloader{matcher: filters.NewMatcher()}
	updater := filtermanager.NewFilterUpdater(store, dl, reloader)

	scheduler := filtermanager.NewScheduler(updater, store, filtermanager.SchedulerConfig{
		Interval:       1 * time.Hour,
		MaxConcurrency: 2,
	})

	if scheduler.IsRunning() {
		t.Errorf("scheduler should not be running before Start()")
	}

	// 1. Start
	ctx := context.Background()
	if err := scheduler.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	if !scheduler.IsRunning() {
		t.Errorf("scheduler should be running after Start()")
	}

	// 2. Duplicate start should error
	if err := scheduler.Start(ctx); err != filtermanager.ErrSchedulerAlreadyRunning {
		t.Errorf("expected ErrSchedulerAlreadyRunning, got %v", err)
	}

	// 3. Stop
	if err := scheduler.Stop(); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}

	if scheduler.IsRunning() {
		t.Errorf("scheduler should not be running after Stop()")
	}

	// 4. Duplicate stop should error
	if err := scheduler.Stop(); err != filtermanager.ErrSchedulerNotRunning {
		t.Errorf("expected ErrSchedulerNotRunning, got %v", err)
	}
}

func TestScheduler_BoundedConcurrency(t *testing.T) {
	ctx := context.Background()
	store := setupTestStore(t)
	dl := filtermanager.NewDownloader(filtermanager.DownloaderConfig{AllowPrivateNetworks: true})
	reloader := &mockReloader{matcher: filters.NewMatcher()}
	updater := filtermanager.NewFilterUpdater(store, dl, reloader)

	const maxConcurrency = 2
	const subCount = 5

	var activeCount int64
	var peakActive int64

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		current := atomic.AddInt64(&activeCount, 1)
		for {
			peak := atomic.LoadInt64(&peakActive)
			if current <= peak || atomic.CompareAndSwapInt64(&peakActive, peak, current) {
				break
			}
		}

		time.Sleep(50 * time.Millisecond)
		atomic.AddInt64(&activeCount, -1)

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("0.0.0.0 domain.com\n"))
	}))
	defer server.Close()

	for i := 0; i < subCount; i++ {
		_ = store.SaveFilterSubscription(ctx, storage.FilterSubscription{
			ID:        fmt.Sprintf("sub-%d", i),
			Name:      fmt.Sprintf("Sub-%d", i),
			SourceURL: fmt.Sprintf("%s/list-%d", server.URL, i),
			Enabled:   true,
			Format:    "hosts",
		})
	}

	scheduler := filtermanager.NewScheduler(updater, store, filtermanager.SchedulerConfig{
		Interval:       1 * time.Hour,
		MaxConcurrency: maxConcurrency,
	})

	results, err := scheduler.RefreshAll(ctx)
	if err != nil {
		t.Fatalf("RefreshAll failed: %v", err)
	}

	if len(results) != subCount {
		t.Errorf("expected %d results, got %d", subCount, len(results))
	}

	peak := atomic.LoadInt64(&peakActive)
	if peak > maxConcurrency {
		t.Errorf("peak concurrency %d exceeded configured maximum %d", peak, maxConcurrency)
	}
}

func TestScheduler_TransientRetryPolicy(t *testing.T) {
	ctx := context.Background()
	store := setupTestStore(t)
	dl := filtermanager.NewDownloader(filtermanager.DownloaderConfig{AllowPrivateNetworks: true})
	reloader := &mockReloader{matcher: filters.NewMatcher()}
	updater := filtermanager.NewFilterUpdater(store, dl, reloader)

	// Sub 1: Transient 503 on first try, then 200 OK on retry
	var sub1Calls int32
	server1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls := atomic.AddInt32(&sub1Calls, 1)
		if calls == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("0.0.0.0 transient-fixed.com\n"))
	}))
	defer server1.Close()

	// Sub 2: Permanent HTML response, should NOT retry
	var sub2Calls int32
	server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&sub2Calls, 1)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("<!DOCTYPE html><html><body>Error</body></html>"))
	}))
	defer server2.Close()

	_ = store.SaveFilterSubscription(ctx, storage.FilterSubscription{
		ID:        "sub-transient",
		Name:      "Transient Sub",
		SourceURL: server1.URL,
		Enabled:   true,
		Format:    "hosts",
	})
	_ = store.SaveFilterSubscription(ctx, storage.FilterSubscription{
		ID:        "sub-permanent",
		Name:      "Permanent Sub",
		SourceURL: server2.URL,
		Enabled:   true,
		Format:    "hosts",
	})

	scheduler := filtermanager.NewScheduler(updater, store, filtermanager.SchedulerConfig{
		Interval:       1 * time.Hour,
		MaxConcurrency: 2,
	})

	results, err := scheduler.RefreshAll(ctx)
	if err != nil {
		t.Fatalf("RefreshAll failed: %v", err)
	}

	// Verify sub1 retried and succeeded
	if atomic.LoadInt32(&sub1Calls) != 2 {
		t.Errorf("expected sub1 to be called twice (initial + retry), got %d", sub1Calls)
	}

	// Verify sub2 was called only once (permanent error, no retry)
	if atomic.LoadInt32(&sub2Calls) != 1 {
		t.Errorf("expected sub2 to be called exactly once (no retry on HTML), got %d", sub2Calls)
	}

	foundTransientSuccess := false
	foundPermanentError := false
	for _, res := range results {
		if res.SubscriptionID == "sub-transient" && res.Status == "updated" {
			foundTransientSuccess = true
		}
		if res.SubscriptionID == "sub-permanent" && res.Status == "error" {
			foundPermanentError = true
		}
	}

	if !foundTransientSuccess {
		t.Errorf("expected sub-transient to succeed after retry")
	}
	if !foundPermanentError {
		t.Errorf("expected sub-permanent to fail with error status")
	}
}

func TestScheduler_PeriodicTrigger(t *testing.T) {
	ctx := context.Background()
	store := setupTestStore(t)
	dl := filtermanager.NewDownloader(filtermanager.DownloaderConfig{AllowPrivateNetworks: true})
	reloader := &mockReloader{matcher: filters.NewMatcher()}
	updater := filtermanager.NewFilterUpdater(store, dl, reloader)

	var requestCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requestCount, 1)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("0.0.0.0 periodic.org\n"))
	}))
	defer server.Close()

	_ = store.SaveFilterSubscription(ctx, storage.FilterSubscription{
		ID:        "sub-periodic",
		Name:      "Periodic Sub",
		SourceURL: server.URL,
		Enabled:   true,
		Format:    "hosts",
	})

	// Configure very fast interval for testing
	scheduler := filtermanager.NewScheduler(updater, store, filtermanager.SchedulerConfig{
		Interval:       25 * time.Millisecond,
		MaxConcurrency: 1,
		JitterFraction: 0.05,
	})

	if err := scheduler.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Wait for multiple ticks
	time.Sleep(120 * time.Millisecond)

	if err := scheduler.Stop(); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}

	calls := atomic.LoadInt32(&requestCount)
	if calls < 2 {
		t.Errorf("expected at least 2 periodic refreshes, got %d", calls)
	}
}
