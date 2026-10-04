package events

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"
)

func TestBoundAndErrors(t *testing.T) {
	e := NewEventEmitter()
	var active, peak atomic.Int32
	for i := 0; i < 32; i++ {
		e.On("x", func(context.Context, interface{}) error {
			n := active.Add(1)
			for {
				p := peak.Load()
				if n <= p || peak.CompareAndSwap(p, n) {
					break
				}
			}
			time.Sleep(time.Millisecond)
			active.Add(-1)
			return nil
		})
	}
	e.On("x", func(context.Context, interface{}) error { return fmt.Errorf("failed") })
	e.On("x", func(context.Context, interface{}) error { panic("bad") })
	if err := e.Emit(context.Background(), "x", nil); err == nil {
		t.Fatal("missing errors")
	}
	if peak.Load() > 8 {
		t.Fatal("unbounded", peak.Load())
	}
}
