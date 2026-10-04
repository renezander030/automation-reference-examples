package quota

import (
	"context"
	"testing"
	"time"
)

func TestInvalidAndBoundary(t *testing.T) {
	q := NewQuotaManager(nil)
	if q.Register("x", 1, "bad") == nil {
		t.Fatal("cycle")
	}
	q.Register("x", 1, ResetDaily)
	if _, e := q.Reserve(context.Background(), "x", -1); e == nil {
		t.Fatal("negative")
	}
	if _, e := q.Reserve(context.Background(), "unknown", 1); e == nil {
		t.Fatal("unknown")
	}
	at := time.Date(2026, 1, 1, 1, 0, 0, 0, time.FixedZone("east", 2*3600))
	if got := redisKeyAt("x", ResetDaily, at); got != "quota:x:2025-12-31" {
		t.Fatal(got)
	}
	if got := redisKeyAt("x", ResetMonthly, at); got != "quota:x:2025-12" {
		t.Fatal(got)
	}
}
