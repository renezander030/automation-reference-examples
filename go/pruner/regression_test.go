package pruner

import (
	"context"
	"testing"
)

func TestInvalid(t *testing.T) {
	if New(nil).WithRetention(-1).WithBatchSize(0).Run(context.Background()) == nil {
		t.Fatal("invalid config")
	}
}
