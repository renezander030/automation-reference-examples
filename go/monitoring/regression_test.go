package monitoring

import (
	"context"
	"testing"
)

func TestFixedLimit(t *testing.T) {
	p := New(.00001)
	p.collect()
	s := p.GetStats()
	if s.WithinLimits || p.maxMemoryMB != .00001 || s.PeakMemoryMB <= p.maxMemoryMB {
		t.Fatal(s)
	}
	if err := p.Start(context.Background(), 0); err == nil {
		t.Fatal("zero interval")
	}
}
