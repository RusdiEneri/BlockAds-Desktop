package storage_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"blockads-desktop/internal/storage"
)

func TestAsyncLogger_LifecycleAndBatchFlush(t *testing.T) {
	store, ctx := setupTestStorage(t)

	logger := storage.NewAsyncLogger(storage.AsyncLoggerConfig{
		Storage:       store,
		QueueCapacity: 500,
		BatchSize:     10,
		FlushInterval: 50 * time.Millisecond,
	})
	defer logger.Close()

	// Enqueue 25 events
	now := time.Now().UTC()
	for i := 0; i < 25; i++ {
		ok := logger.Enqueue(storage.QueryEvent{
			ID:                fmt.Sprintf("ev-%d", i),
			Timestamp:         now.Add(time.Duration(i) * time.Second),
			Domain:            fmt.Sprintf("domain%d.com", i),
			ClientIP:          "127.0.0.1",
			QueryType:         "A",
			Status:            "ALLOWED",
			UpstreamLatencyMs: 15,
			Protocol:          "UDP",
		})
		if !ok {
			t.Fatalf("failed to enqueue event %d", i)
		}
	}

	// Wait for background worker to flush
	time.Sleep(150 * time.Millisecond)

	if logger.TotalWritten() != 25 {
		t.Fatalf("expected 25 written events, got %d", logger.TotalWritten())
	}

	// Query events from DB
	entries, total, err := store.GetQueryEvents(ctx, storage.QueryFilter{Limit: 50})
	if err != nil {
		t.Fatalf("failed to query events: %v", err)
	}
	if total != 25 || len(entries) != 25 {
		t.Errorf("expected 25 entries in db, got total=%d, len=%d", total, len(entries))
	}
}

func TestAsyncLogger_GracefulShutdownDrainsQueue(t *testing.T) {
	store, ctx := setupTestStorage(t)

	logger := storage.NewAsyncLogger(storage.AsyncLoggerConfig{
		Storage:       store,
		QueueCapacity: 500,
		BatchSize:     100,              // Large batch size so it won't flush immediately by count
		FlushInterval: 10 * time.Second, // Long interval so it won't flush by timer
	})

	now := time.Now().UTC()
	for i := 0; i < 30; i++ {
		logger.Enqueue(storage.QueryEvent{
			ID:        fmt.Sprintf("drain-%d", i),
			Timestamp: now,
			Domain:    "drain-test.local",
			ClientIP:  "127.0.0.1",
			QueryType: "A",
			Status:    "ALLOWED",
		})
	}

	// Close immediately: must drain pending items before returning
	if err := logger.Close(); err != nil {
		t.Fatalf("failed to close logger: %v", err)
	}

	if logger.TotalWritten() != 30 {
		t.Errorf("expected 30 written events after shutdown drain, got %d", logger.TotalWritten())
	}

	_, total, err := store.GetQueryEvents(ctx, storage.QueryFilter{Limit: 50})
	if err != nil {
		t.Fatalf("failed to get query events: %v", err)
	}
	if total != 30 {
		t.Errorf("expected 30 persisted events, got %d", total)
	}
}

func TestAsyncLogger_QueueSaturationDropsWithoutBlocking(t *testing.T) {
	store, _ := setupTestStorage(t)

	// Create logger with small capacity 5 and long flush interval
	logger := storage.NewAsyncLogger(storage.AsyncLoggerConfig{
		Storage:       store,
		QueueCapacity: 5,
		BatchSize:     100,
		FlushInterval: 1 * time.Hour,
	})
	defer logger.Close()

	now := time.Now().UTC()
	// Rapidly enqueue 15 items
	accepted := 0
	dropped := 0
	for i := 0; i < 15; i++ {
		if logger.Enqueue(storage.QueryEvent{
			ID:        fmt.Sprintf("sat-%d", i),
			Timestamp: now,
			Domain:    "saturation.test",
		}) {
			accepted++
		} else {
			dropped++
		}
	}

	if dropped == 0 {
		t.Errorf("expected dropped events when queue capacity (5) was exceeded")
	}
	if logger.DroppedCount() != uint64(dropped) {
		t.Errorf("expected DroppedCount %d, got %d", dropped, logger.DroppedCount())
	}
	if accepted+dropped != 15 {
		t.Errorf("expected total 15, got accepted=%d, dropped=%d", accepted, dropped)
	}
}

func TestStorage_QueryEvents_PaginationAndPruning(t *testing.T) {
	store, ctx := setupTestStorage(t)

	now := time.Now().UTC()
	var batch []storage.QueryEvent
	for i := 0; i < 20; i++ {
		status := "ALLOWED"
		if i%2 == 0 {
			status = "BLOCKED"
		}
		batch = append(batch, storage.QueryEvent{
			ID:        fmt.Sprintf("prune-%d", i),
			Timestamp: now.Add(-time.Duration(20-i) * time.Hour),
			Domain:    fmt.Sprintf("host%d.com", i),
			ClientIP:  "127.0.0.1",
			QueryType: "A",
			Status:    status,
		})
	}

	if err := store.InsertQueryEventsBatch(ctx, batch); err != nil {
		t.Fatalf("failed to insert batch: %v", err)
	}

	// 1. Pagination: Limit 5, Offset 0
	entries1, total, err := store.GetQueryEvents(ctx, storage.QueryFilter{Limit: 5, Offset: 0})
	if err != nil {
		t.Fatalf("failed to get page 1: %v", err)
	}
	if total != 20 || len(entries1) != 5 {
		t.Fatalf("expected total 20 and page length 5, got total=%d, len=%d", total, len(entries1))
	}

	// 2. Filter by status: BLOCKED
	blockedEntries, totalBlocked, err := store.GetQueryEvents(ctx, storage.QueryFilter{Status: "BLOCKED", Limit: 50})
	if err != nil {
		t.Fatalf("failed to filter by status: %v", err)
	}
	if totalBlocked != 10 || len(blockedEntries) != 10 {
		t.Errorf("expected 10 BLOCKED entries, got total=%d, len=%d", totalBlocked, len(blockedEntries))
	}

	// 3. Prune events older than 10.5 hours (events i=0..9 are 20h..11h old; events i=10..19 are 10h..1h old)
	deleted, err := store.PruneQueryEvents(ctx, 10*time.Hour+30*time.Minute)
	if err != nil {
		t.Fatalf("failed to prune events: %v", err)
	}
	if deleted != 10 {
		t.Errorf("expected 10 pruned events, got %d", deleted)
	}

	// Verify remaining count
	_, remaining, _ := store.GetQueryEvents(ctx, storage.QueryFilter{Limit: 50})
	if remaining != 10 {
		t.Errorf("expected 10 remaining events, got %d", remaining)
	}
}

func BenchmarkAsyncLogger_Enqueue(b *testing.B) {
	tempDir := b.TempDir()
	store, err := storage.Open(tempDir + "/bench.db")
	if err != nil {
		b.Fatalf("failed to open storage: %v", err)
	}
	defer store.Close()
	_ = store.Migrate(context.Background())

	logger := storage.NewAsyncLogger(storage.AsyncLoggerConfig{
		Storage:       store,
		QueueCapacity: 50000,
		BatchSize:     500,
		FlushInterval: 100 * time.Millisecond,
	})
	defer logger.Close()

	ev := storage.QueryEvent{
		ID:                "bench-1",
		Timestamp:         time.Now(),
		Domain:            "benchmark.google.com",
		ClientIP:          "127.0.0.1",
		QueryType:         "A",
		Status:            "ALLOWED",
		UpstreamLatencyMs: 12,
		FilterRule:        "Passthrough",
		Reason:            "Clean domain",
		Protocol:          "UDP",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Enqueue(ev)
	}
}

func BenchmarkStorage_InsertBatch(b *testing.B) {
	tempDir := b.TempDir()
	store, err := storage.Open(tempDir + "/bench_insert.db")
	if err != nil {
		b.Fatalf("failed to open storage: %v", err)
	}
	defer store.Close()
	_ = store.Migrate(context.Background())

	batchSize := 100
	batch := make([]storage.QueryEvent, batchSize)
	now := time.Now()
	for i := 0; i < batchSize; i++ {
		batch[i] = storage.QueryEvent{
			ID:                fmt.Sprintf("bench-b-%d", i),
			Timestamp:         now,
			Domain:            "adservice.google.com",
			ClientIP:          "127.0.0.1",
			QueryType:         "A",
			Status:            "BLOCKED",
			UpstreamLatencyMs: 1,
			FilterRule:        "StevenBlack",
			Reason:            "Ad tracker",
			Protocol:          "UDP",
		}
	}

	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Update IDs so they don't collide
		for j := 0; j < batchSize; j++ {
			batch[j].ID = fmt.Sprintf("bench-%d-%d", i, j)
		}
		if err := store.InsertQueryEventsBatch(ctx, batch); err != nil {
			b.Fatalf("insert batch failed: %v", err)
		}
	}
}
