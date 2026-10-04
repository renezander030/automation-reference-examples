package pubsub

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/redis/go-redis/v9"
	"sync"
	"time"
)

// BufferedPublisher serializes publication. Redis Pub/Sub is at-most-once.
// A failed command may have reached Redis; retrying a retained batch can duplicate it.
// Timer errors are delivered on Errors; use Redis Streams for durable delivery.
type BufferedPublisher struct {
	client     *redis.Client
	channel    string
	bufferSize int
	flushDelay time.Duration
	mu         sync.Mutex
	buffer     []json.RawMessage
	timer      *time.Timer
	closed     bool
	Errors     chan error
}

func NewBufferedPublisher(client *redis.Client, channel string, size int, delay time.Duration) *BufferedPublisher {
	if client == nil || channel == "" || size <= 0 || delay <= 0 {
		panic("client, channel, positive capacity and delay required")
	}
	return &BufferedPublisher{client: client, channel: channel, bufferSize: size, flushDelay: delay, Errors: make(chan error, 1)}
}
func (p *BufferedPublisher) Publish(ctx context.Context, event interface{}) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return fmt.Errorf("publisher closed")
	}
	if len(p.buffer) >= p.bufferSize {
		return fmt.Errorf("buffer full: resolve failed flush first")
	}
	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}
	p.buffer = append(p.buffer, payload)
	if len(p.buffer) == p.bufferSize {
		return p.flush(ctx)
	}
	if p.timer == nil {
		p.timer = time.AfterFunc(p.flushDelay, func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := p.Flush(ctx); err != nil {
				select {
				case p.Errors <- err:
				default:
				}
			}
		})
	}
	return nil
}
func (p *BufferedPublisher) PublishImmediate(ctx context.Context, event interface{}) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return fmt.Errorf("publisher closed")
	}
	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}
	if err = p.flush(ctx); err != nil {
		return err
	}
	return p.client.Publish(ctx, p.channel, payload).Err()
}
func (p *BufferedPublisher) flush(ctx context.Context) error {
	if p.timer != nil {
		p.timer.Stop()
		p.timer = nil
	}
	if len(p.buffer) == 0 {
		return nil
	}
	payload, err := json.Marshal(p.buffer)
	if err != nil {
		return err
	}
	if err = p.client.Publish(ctx, p.channel, payload).Err(); err != nil {
		return err
	}
	p.buffer = nil
	return nil
}
func (p *BufferedPublisher) Flush(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.flush(ctx)
}

// Close prevents new events even if flush fails. Flush can be retried explicitly.
func (p *BufferedPublisher) Close(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.closed = true
	return p.flush(ctx)
}
