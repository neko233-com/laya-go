// Package metrics records in-process decision telemetry for the ops dashboard.
package metrics

import (
	"sync"
	"time"
)

// Event is one recorded decide call.
type Event struct {
	At             time.Time `json:"at"`
	Model          string    `json:"model"`
	Top            string    `json:"top"`
	Probability    float64   `json:"probability"`
	LatencyUS      int64     `json:"latency_us"`
	CandidateCount int       `json:"candidate_count"`
	Compat         string    `json:"compat,omitempty"`
	Path           string    `json:"path,omitempty"`
}

// Snapshot is a metrics export for the admin API.
type Snapshot struct {
	TotalDecides   int64            `json:"total_decides"`
	AvgLatencyUS   float64          `json:"avg_latency_us"`
	ByModel        map[string]int64 `json:"by_model"`
	ByPattern      map[string]int64 `json:"by_pattern"`
	Recent         []Event          `json:"recent"`
	RecentCapacity int              `json:"recent_capacity"`
	StartedAt      time.Time        `json:"started_at"`
	UptimeSec      float64          `json:"uptime_sec"`
}

// Registry collects decision metrics.
type Registry struct {
	mu         sync.Mutex
	started    time.Time
	total      int64
	latencySum int64
	byModel    map[string]int64
	byPattern  map[string]int64
	recent     []Event
	maxRecent  int
}

// New creates a metrics registry.
func New(maxRecent int) *Registry {
	if maxRecent <= 0 {
		maxRecent = 100
	}
	return &Registry{
		started:   time.Now(),
		byModel:   make(map[string]int64),
		byPattern: make(map[string]int64),
		maxRecent: maxRecent,
		recent:    make([]Event, 0, maxRecent),
	}
}

// Record appends one decision event.
func (r *Registry) Record(ev Event) {
	if r == nil {
		return
	}
	if ev.At.IsZero() {
		ev.At = time.Now()
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.total++
	r.latencySum += ev.LatencyUS
	if ev.Model != "" {
		r.byModel[ev.Model]++
	}
	if ev.Top != "" {
		r.byPattern[ev.Top]++
	}
	r.recent = append(r.recent, ev)
	if len(r.recent) > r.maxRecent {
		r.recent = r.recent[len(r.recent)-r.maxRecent:]
	}
}

// Snapshot returns a copy of current metrics.
func (r *Registry) Snapshot() Snapshot {
	r.mu.Lock()
	defer r.mu.Unlock()
	byModel := make(map[string]int64, len(r.byModel))
	for k, v := range r.byModel {
		byModel[k] = v
	}
	byPattern := make(map[string]int64, len(r.byPattern))
	for k, v := range r.byPattern {
		byPattern[k] = v
	}
	recent := make([]Event, len(r.recent))
	copy(recent, r.recent)
	// newest first for UI
	for i, j := 0, len(recent)-1; i < j; i, j = i+1, j-1 {
		recent[i], recent[j] = recent[j], recent[i]
	}
	avg := float64(0)
	if r.total > 0 {
		avg = float64(r.latencySum) / float64(r.total)
	}
	return Snapshot{
		TotalDecides:   r.total,
		AvgLatencyUS:   avg,
		ByModel:        byModel,
		ByPattern:      byPattern,
		Recent:         recent,
		RecentCapacity: r.maxRecent,
		StartedAt:      r.started,
		UptimeSec:      time.Since(r.started).Seconds(),
	}
}

// Total returns decide count.
func (r *Registry) Total() int64 {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.total
}
