package monitoring

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"
)

type Stats struct {
	UptimeSeconds  float64 `json:"uptimeSeconds"`
	PeakMemoryMB   float64 `json:"peakMemoryMB"`
	MemoryMB       float64 `json:"memoryMB"`
	MemoryPercent  float64 `json:"memoryPercent"`
	GoroutineCount int     `json:"goroutineCount"`
	GCCount        uint32  `json:"gcCount"`
	WithinLimits   bool    `json:"withinLimits"`
}

// PerfMonitor collects runtime metrics at a configurable interval.
type PerfMonitor struct {
	mu             sync.RWMutex
	startTime      time.Time
	maxMemoryMB    float64
	peakMemoryMB   float64
	memorySamples  []float64
	lastGoroutines int
	gcCount        uint32
}

// New creates a monitor with a memory limit in MB.
func New(maxMemoryMB float64) *PerfMonitor {
	if maxMemoryMB <= 0 {
		panic("memory limit must be positive")
	}
	return &PerfMonitor{
		startTime:     time.Now(),
		maxMemoryMB:   maxMemoryMB,
		memorySamples: make([]float64, 0, 100),
	}
}

// Start begins periodic metric collection. Blocks until ctx is cancelled.
func (pm *PerfMonitor) Start(ctx context.Context, interval time.Duration) error {
	if interval <= 0 {
		return fmt.Errorf("collection interval must be positive")
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			pm.collect()
		}
	}
}

func (pm *PerfMonitor) collect() {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	memMB := float64(m.Alloc) / 1024 / 1024
	pm.memorySamples = append(pm.memorySamples, memMB)
	if len(pm.memorySamples) > 100 {
		pm.memorySamples = pm.memorySamples[1:]
	}
	if memMB > pm.peakMemoryMB {
		pm.peakMemoryMB = memMB
	}

	pm.gcCount = m.NumGC
	pm.lastGoroutines = runtime.NumGoroutine()
}

// GetStats returns a snapshot of current performance metrics.
func (pm *PerfMonitor) GetStats() Stats {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	var currentMem float64
	if len(pm.memorySamples) > 0 {
		currentMem = pm.memorySamples[len(pm.memorySamples)-1]
	}

	return Stats{
		PeakMemoryMB:   pm.peakMemoryMB,
		UptimeSeconds:  time.Since(pm.startTime).Seconds(),
		MemoryMB:       currentMem,
		MemoryPercent:  (currentMem / pm.maxMemoryMB) * 100,
		GoroutineCount: pm.lastGoroutines,
		GCCount:        pm.gcCount,
		WithinLimits:   currentMem <= pm.maxMemoryMB,
	}
}

// String returns a one-line summary for logging.
func (pm *PerfMonitor) String() string {
	s := pm.GetStats()
	return fmt.Sprintf("uptime=%.0fs mem=%.1fMB goroutines=%d gc=%d",
		s.UptimeSeconds, s.MemoryMB, s.GoroutineCount, s.GCCount)
}
