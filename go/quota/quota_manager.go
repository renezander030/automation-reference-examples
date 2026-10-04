package quota

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

type ResetCycle string

const (
	ResetDaily   ResetCycle = "daily"
	ResetMonthly ResetCycle = "monthly"
)

type QuotaConfig struct {
	Provider string
	Limit    int64
	Cycle    ResetCycle
}

// QuotaManager tracks API usage quotas via Redis with atomic Lua scripts.
type QuotaManager struct {
	mu     sync.RWMutex
	rdb    *redis.Client
	quotas map[string]QuotaConfig
}

func NewQuotaManager(rdb *redis.Client) *QuotaManager {
	return &QuotaManager{rdb: rdb, quotas: make(map[string]QuotaConfig)}
}

func (q *QuotaManager) Register(provider string, limit int64, cycle ResetCycle) error {
	if provider == "" || limit <= 0 || limit > 1<<53-1 || (cycle != ResetDaily && cycle != ResetMonthly) {
		return fmt.Errorf("invalid quota configuration")
	}
	q.mu.Lock()
	defer q.mu.Unlock()
	q.quotas[provider] = QuotaConfig{Provider: provider, Limit: limit, Cycle: cycle}
	return nil
}

// Reserve atomically checks and increments usage. Returns true if within quota.
func (q *QuotaManager) Reserve(ctx context.Context, provider string, cost int64) (bool, error) {
	if cost < 0 || cost > 1<<53-1 {
		return false, fmt.Errorf("cost must be a nonnegative safe integer")
	}
	q.mu.RLock()
	cfg, ok := q.quotas[provider]
	q.mu.RUnlock()
	if !ok {
		return false, fmt.Errorf("no quota configured for provider %s", provider)
	}

	key := q.redisKey(provider, cfg.Cycle)

	// Atomic check-and-increment with auto-expiring keys
	script := redis.NewScript(`
		local current = tonumber(redis.call('GET', KEYS[1]) or "0")
		local limit = tonumber(ARGV[1])
		local cost = tonumber(ARGV[2])
		local ttl = tonumber(ARGV[3])
		if current + cost > limit then
			return -1
		end
		local val = redis.call('INCRBY', KEYS[1], cost)
		if val == cost then
			redis.call('EXPIRE', KEYS[1], ttl)
		end
		return val
	`)

	ttl := q.ttlSeconds(cfg.Cycle)
	result, err := script.Run(ctx, q.rdb, []string{key}, cfg.Limit, cost, ttl).Int64()
	if err != nil {
		return false, fmt.Errorf("quota reserve for %s: %w", provider, err)
	}
	return result >= 0, nil
}

// ThrottledCall wraps an API call with a quota check.
func (q *QuotaManager) ThrottledCall(ctx context.Context, provider string, cost int64, fn func() error) error {
	allowed, err := q.Reserve(ctx, provider, cost)
	if err != nil {
		return err
	}
	if !allowed {
		return fmt.Errorf("api quota exhausted for %s", provider)
	}
	return fn()
}

func (q *QuotaManager) redisKey(provider string, cycle ResetCycle) string {
	return redisKeyAt(provider, cycle, time.Now())
}

func redisKeyAt(provider string, cycle ResetCycle, at time.Time) string {
	now := at.UTC()
	if cycle == ResetMonthly {
		return fmt.Sprintf("quota:%s:%s", provider, now.Format("2006-01"))
	}
	return fmt.Sprintf("quota:%s:%s", provider, now.Format("2006-01-02"))
}

func (q *QuotaManager) ttlSeconds(cycle ResetCycle) int {
	if cycle == ResetMonthly {
		return 35 * 24 * 3600 // ~35 days
	}
	return 2 * 24 * 3600 // ~2 days buffer
}
