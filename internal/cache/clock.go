package cache

import (
	"sync"
	"time"
)

// Clock provides an interface for querying the current time, enabling deterministic testing.
type Clock interface {
	Now() time.Time
}

// RealClock queries the system UTC time.
type RealClock struct{}

// Now returns the current UTC time.
func (RealClock) Now() time.Time {
	return time.Now().UTC()
}

// MockClock is a thread-safe injectable clock for tests.
type MockClock struct {
	mu  sync.Mutex
	now time.Time
}

// NewMockClock creates a MockClock initialized to initial time.
func NewMockClock(initial time.Time) *MockClock {
	if initial.IsZero() {
		initial = time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	}
	return &MockClock{now: initial}
}

// Now returns the mock current time.
func (m *MockClock) Now() time.Time {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.now
}

// Advance moves the mock clock forward by d.
func (m *MockClock) Advance(d time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.now = m.now.Add(d)
}

// Set explicitly sets the mock clock time.
func (m *MockClock) Set(t time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.now = t
}
