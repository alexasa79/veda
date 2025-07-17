package perf

import (
	"sort"
	"sync"
	"time"
	"veda/internal/log"
)

var LL *log.Logger = log.NewLogger("PERF")

// TimeTracker manages performance measurements
type TimeTracker struct {
	mu       sync.Mutex
	metrics  map[string]*Metric
	enabled  bool
	reporter *Reporter
}

// Metric holds timing statistics for a named operation
type Metric struct {
	name      string
	count     int64
	totalTime time.Duration
	minTime   time.Duration
	maxTime   time.Duration
	lastTime  time.Duration
}

// New creates a new TimeTracker
func New(enabled bool, reportInterval time.Duration) *TimeTracker {
	tt := &TimeTracker{
		metrics: make(map[string]*Metric),
		enabled: enabled,
	}

	if enabled && reportInterval > 0 {
		tt.reporter = &Reporter{
			tracker:  tt,
			interval: reportInterval,
			logger:   LL,
			stopCh:   make(chan struct{}),
		}
		tt.reporter.Start()
	}

	return tt
}

// Start returns the current time for measurement
func (tt *TimeTracker) Start() time.Time {
	if !tt.enabled {
		return time.Time{}
	}
	return time.Now()
}

// ReportSince measures time since 'start', records it under 'name',
// and returns current time for next measurement
func (tt *TimeTracker) ReportSince(name string, start time.Time) time.Time {
	if !tt.enabled || start.IsZero() {
		return tt.Start()
	}

	elapsed := time.Since(start)
	tt.record(name, elapsed)

	// Return current time for chaining measurements
	return time.Now()
}

// record updates the metrics for a named operation
func (tt *TimeTracker) record(name string, elapsed time.Duration) {
	tt.mu.Lock()
	defer tt.mu.Unlock()

	metric, exists := tt.metrics[name]
	if !exists {
		metric = &Metric{
			name:    name,
			minTime: elapsed,
			maxTime: elapsed,
		}
		tt.metrics[name] = metric
	}

	metric.count++
	metric.totalTime += elapsed
	metric.lastTime = elapsed

	if elapsed < metric.minTime {
		metric.minTime = elapsed
	}
	if elapsed > metric.maxTime {
		metric.maxTime = elapsed
	}
}

// Stop stops the reporter if running
func (tt *TimeTracker) Stop() {
	if tt.reporter != nil {
		close(tt.reporter.stopCh)
	}
}

// Reporter handles periodic reporting of metrics
type Reporter struct {
	tracker  *TimeTracker
	interval time.Duration
	logger   *log.Logger
	stopCh   chan struct{}
}

// Start begins the periodic reporting
func (r *Reporter) Start() {
	go func() {
		ticker := time.NewTicker(r.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				r.report()
			case <-r.stopCh:
				return
			}
		}
	}()
}

// report outputs the current metrics and resets them
func (r *Reporter) report() {
	r.tracker.mu.Lock()
	defer r.tracker.mu.Unlock()

	if len(r.tracker.metrics) == 0 {
		return
	}

	r.logger.Info("Performance Report (last %.1fs):", r.interval.Seconds())

	// Convert map to slice for sorting
	metrics := make([]*Metric, 0, len(r.tracker.metrics))
	for _, m := range r.tracker.metrics {
		metrics = append(metrics, m)
	}

	// Sort by total time (most expensive first)
	sort.Slice(metrics, func(i, j int) bool {
		return metrics[i].totalTime > metrics[j].totalTime
	})

	// Report each metric with microsecond precision
	for _, metric := range metrics {
		avg := metric.totalTime / time.Duration(metric.count)
		r.logger.Info("  %-30s avg=%10.6fms min=%10.6fms max=%10.6fms count=%6d total=%11.6fms",
			metric.name,
			float64(avg)/float64(time.Millisecond),
			float64(metric.minTime)/float64(time.Millisecond),
			float64(metric.maxTime)/float64(time.Millisecond),
			metric.count,
			float64(metric.totalTime)/float64(time.Millisecond))
	}

	// Reset metrics after reporting
	r.tracker.metrics = make(map[string]*Metric)
}

// Enable enables or disables the tracker
func (tt *TimeTracker) Enable(enabled bool) {
	tt.mu.Lock()
	defer tt.mu.Unlock()
	tt.enabled = enabled
}

// IsEnabled returns whether the tracker is enabled
func (tt *TimeTracker) IsEnabled() bool {
	tt.mu.Lock()
	defer tt.mu.Unlock()
	return tt.enabled
}