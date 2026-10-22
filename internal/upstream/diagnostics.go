package upstream

import (
	"sync"
	"time"
)

// Diagnostics contains runtime diagnostic telemetry for the DoH upstream resolver.
type Diagnostics struct {
	EndpointURL      string    `json:"endpointUrl"`
	Hostname         string    `json:"hostname"`
	CurrentIP        string    `json:"currentIp"`
	LastLatencyMs    int       `json:"lastLatencyMs"`
	LastSuccessTime  time.Time `json:"lastSuccessTime"`
	LastErrorTime    time.Time `json:"lastErrorTime"`
	LastErrorMessage string    `json:"lastErrorMessage,omitempty"`
	TotalQueries     uint64    `json:"totalQueries"`
	FailedQueries    uint64    `json:"failedQueries"`
	IsReady          bool      `json:"isReady"`
}

type diagnosticsTracker struct {
	mu          sync.RWMutex
	endpointURL string
	hostname    string
	dialer      *EndpointDialer

	lastLatencyMs    int
	lastSuccessTime  time.Time
	lastErrorTime    time.Time
	lastErrorMessage string
	totalQueries     uint64
	failedQueries    uint64
}

func newDiagnosticsTracker(endpointURL, hostname string, dialer *EndpointDialer) *diagnosticsTracker {
	return &diagnosticsTracker{
		endpointURL: endpointURL,
		hostname:    hostname,
		dialer:      dialer,
	}
}

func (dt *diagnosticsTracker) recordSuccess(latencyMs int) {
	dt.mu.Lock()
	defer dt.mu.Unlock()
	dt.totalQueries++
	dt.lastLatencyMs = latencyMs
	dt.lastSuccessTime = time.Now().UTC()
}

func (dt *diagnosticsTracker) recordFailure(err error, latencyMs int) {
	dt.mu.Lock()
	defer dt.mu.Unlock()
	dt.totalQueries++
	dt.failedQueries++
	dt.lastLatencyMs = latencyMs
	dt.lastErrorTime = time.Now().UTC()
	if err != nil {
		dt.lastErrorMessage = err.Error()
	}
}

func (dt *diagnosticsTracker) snapshot() Diagnostics {
	dt.mu.RLock()
	defer dt.mu.RUnlock()

	currentIP := ""
	if dt.dialer != nil {
		currentIP = dt.dialer.CurrentIP()
	}

	isReady := dt.totalQueries == 0 || dt.failedQueries < dt.totalQueries

	return Diagnostics{
		EndpointURL:      dt.endpointURL,
		Hostname:         dt.hostname,
		CurrentIP:        currentIP,
		LastLatencyMs:    dt.lastLatencyMs,
		LastSuccessTime:  dt.lastSuccessTime,
		LastErrorTime:    dt.lastErrorTime,
		LastErrorMessage: dt.lastErrorMessage,
		TotalQueries:     dt.totalQueries,
		FailedQueries:    dt.failedQueries,
		IsReady:          isReady,
	}
}
