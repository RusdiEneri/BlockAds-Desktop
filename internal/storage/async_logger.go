package storage

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

// Default async logger parameters.
const (
	DefaultQueueCapacity = 4096
	DefaultBatchSize     = 100
	DefaultFlushInterval = 200 * time.Millisecond
)

// AsyncLoggerConfig configures the non-blocking query event logging worker.
type AsyncLoggerConfig struct {
	Storage       *Storage
	QueueCapacity int
	BatchSize     int
	FlushInterval time.Duration
}

// AsyncLogger queues DNS query events in memory and writes them in transactional batches
// to SQLite asynchronously, completely off the DNS request/response path.
type AsyncLogger struct {
	storage       *Storage
	queue         chan QueryEvent
	batchSize     int
	flushInterval time.Duration

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// Atomic telemetry and diagnostic counters
	totalEnqueued atomic.Uint64
	totalWritten  atomic.Uint64
	droppedCount  atomic.Uint64
	writeFailures atomic.Uint64
}

// NewAsyncLogger starts the background logging worker goroutine.
func NewAsyncLogger(cfg AsyncLoggerConfig) *AsyncLogger {
	cap := cfg.QueueCapacity
	if cap <= 0 {
		cap = DefaultQueueCapacity
	}

	batch := cfg.BatchSize
	if batch <= 0 {
		batch = DefaultBatchSize
	}

	interval := cfg.FlushInterval
	if interval <= 0 {
		interval = DefaultFlushInterval
	}

	ctx, cancel := context.WithCancel(context.Background())

	l := &AsyncLogger{
		storage:       cfg.Storage,
		queue:         make(chan QueryEvent, cap),
		batchSize:     batch,
		flushInterval: interval,
		ctx:           ctx,
		cancel:        cancel,
	}

	l.wg.Add(1)
	go l.run()

	return l
}

// Enqueue submits a query event to the logger without blocking or disk I/O.
// Returns true if the event was queued, or false if the queue was full (dropped).
func (l *AsyncLogger) Enqueue(event QueryEvent) bool {
	select {
	case l.queue <- event:
		l.totalEnqueued.Add(1)
		return true
	default:
		l.droppedCount.Add(1)
		return false
	}
}

// run is the background worker that flushes batches on timer or batch size threshold.
func (l *AsyncLogger) run() {
	defer l.wg.Done()

	ticker := time.NewTicker(l.flushInterval)
	defer ticker.Stop()

	batch := make([]QueryEvent, 0, l.batchSize)

	flush := func() {
		if len(batch) == 0 {
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		err := l.storage.InsertQueryEventsBatch(ctx, batch)
		cancel()

		if err != nil {
			l.writeFailures.Add(1)
		} else {
			l.totalWritten.Add(uint64(len(batch)))
		}

		batch = batch[:0]
	}

	for {
		select {
		case <-l.ctx.Done():
			// Drain any remaining events in the channel before exiting
			for {
				select {
				case ev := <-l.queue:
					batch = append(batch, ev)
					if len(batch) >= l.batchSize {
						flush()
					}
				default:
					flush()
					return
				}
			}

		case ev := <-l.queue:
			batch = append(batch, ev)
			if len(batch) >= l.batchSize {
				flush()
			}

		case <-ticker.C:
			flush()
		}
	}
}

// Close gracefully flushes all remaining queued events and terminates the worker.
func (l *AsyncLogger) Close() error {
	l.cancel()
	l.wg.Wait()
	return nil
}

// QueueDepth returns the current number of events waiting in the queue.
func (l *AsyncLogger) QueueDepth() int {
	return len(l.queue)
}

// QueueCapacity returns the maximum buffer capacity.
func (l *AsyncLogger) QueueCapacity() int {
	return cap(l.queue)
}

// DroppedCount returns the total number of events dropped due to queue saturation.
func (l *AsyncLogger) DroppedCount() uint64 {
	return l.droppedCount.Load()
}

// TotalEnqueued returns the total number of successfully queued events.
func (l *AsyncLogger) TotalEnqueued() uint64 {
	return l.totalEnqueued.Load()
}

// TotalWritten returns the total number of events successfully persisted to SQLite.
func (l *AsyncLogger) TotalWritten() uint64 {
	return l.totalWritten.Load()
}

// WriteFailures returns the count of batch write errors encountered.
func (l *AsyncLogger) WriteFailures() uint64 {
	return l.writeFailures.Load()
}
