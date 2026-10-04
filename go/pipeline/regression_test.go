package pipeline

import (
	"context"
	"testing"
	"time"
)

func TestCancellationAndFreeze(t *testing.T) {
	r := NewRunner()
	r.RegisterHandler("x", func(ctx context.Context, s Step, d map[string]interface{}) error { <-ctx.Done(); return ctx.Err() })
	p := Pipeline{Name: "p", Timeout: time.Millisecond, Steps: []Step{{Name: "x", Type: "x"}}}
	if err := r.Run(context.Background(), p, nil); err == nil {
		t.Fatal("timeout")
	}
	if err := r.RegisterHandler("y", func(context.Context, Step, map[string]interface{}) error { return nil }); err == nil {
		t.Fatal("registry mutation")
	}
}
func TestCallerCancelled(t *testing.T) {
	ctx, c := context.WithCancel(context.Background())
	c()
	if err := NewRunner().Run(ctx, Pipeline{Steps: []Step{{Type: "x"}}}, nil); err == nil {
		t.Fatal("caller cancellation")
	}
}
