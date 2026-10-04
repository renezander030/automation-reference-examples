package pruner

import (
	"context"
	"fmt"
	"time"
)

// Store defines the data operations the pruner needs.
type Store interface {
	PruneRecords(ctx context.Context, olderThan time.Time, limit int) (int64, error)
	FindOrphans(ctx context.Context, limit int) ([]int, error)
	DeleteByIDs(ctx context.Context, ids []int) error
}

// Metrics receives telemetry from the pruner.
type Metrics interface {
	Counter(name string, value float64, labels map[string]string)
}

// Pruner uses fluent setters. Configure before calling Run; no concurrent mutation.
type Pruner struct {
	store     Store
	retention time.Duration
	batchSize int
	metrics   Metrics
}

func New(store Store) *Pruner {
	return &Pruner{
		store:     store,
		retention: 180 * 24 * time.Hour, // 6 months default
		batchSize: 1000,
	}
}

// WithRetention sets how long records are kept before pruning.
func (p *Pruner) WithRetention(d time.Duration) *Pruner {
	p.retention = d
	return p
}

// WithBatchSize sets the maximum records per pruning pass.
func (p *Pruner) WithBatchSize(n int) *Pruner {
	p.batchSize = n
	return p
}

// WithMetrics attaches a metrics sink.
func (p *Pruner) WithMetrics(m Metrics) *Pruner {
	p.metrics = m
	return p
}

// Run executes one pruning pass: delete old records, then remove orphans.
func (p *Pruner) Run(ctx context.Context) error {
	if p.store == nil || p.retention <= 0 || p.batchSize <= 0 {
		return fmt.Errorf("store, positive retention and batch size required")
	}
	cutoff := time.Now().Add(-p.retention)

	deleted, err := p.store.PruneRecords(ctx, cutoff, p.batchSize)
	if err != nil {
		return err
	}
	if p.metrics != nil {
		p.metrics.Counter("pruner_records_deleted", float64(deleted), nil)
	}

	orphans, err := p.store.FindOrphans(ctx, p.batchSize)
	if err != nil {
		return err
	}
	if len(orphans) > 0 {
		if err := p.store.DeleteByIDs(ctx, orphans); err != nil {
			return err
		}
		if p.metrics != nil {
			p.metrics.Counter("pruner_orphans_deleted", float64(len(orphans)), nil)
		}
	}

	return nil
}
