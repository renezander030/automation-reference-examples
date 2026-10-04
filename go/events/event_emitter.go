package events

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

type EventHandler func(ctx context.Context, data interface{}) error

// EventEmitter provides a thread-safe pub/sub event bus.
// Emit waits for at most eight concurrent handlers and reports errors/panics.
// Handlers must cooperate with context cancellation; delivery is in-process only.
type EventEmitter struct {
	mu       sync.RWMutex
	handlers map[string][]EventHandler
}

func NewEventEmitter() *EventEmitter {
	return &EventEmitter{handlers: make(map[string][]EventHandler)}
}

// On registers a handler for the given event name.
func (e *EventEmitter) On(name string, handler EventHandler) {
	if handler == nil {
		panic("nil handler")
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	e.handlers[name] = append(e.handlers[name], handler)
}

// Emit joins bounded handler results. It is not a durable event queue.
func (e *EventEmitter) Emit(ctx context.Context, name string, data interface{}) error {
	e.mu.RLock()
	snapshot := make([]EventHandler, len(e.handlers[name]))
	copy(snapshot, e.handlers[name])
	e.mu.RUnlock()

	results := make(chan error, len(snapshot))
	sem := make(chan struct{}, 8)
	var wg sync.WaitGroup
	for _, h := range snapshot {
		select {
		case sem <- struct{}{}:
		case <-ctx.Done():
			results <- ctx.Err()
			continue
		}
		wg.Add(1)
		go func(fn EventHandler) {
			defer wg.Done()
			defer func() { <-sem }()
			defer func() {
				if v := recover(); v != nil {
					results <- fmt.Errorf("handler panic")
				}
			}()
			results <- fn(ctx, data)
		}(h)
	}
	wg.Wait()
	close(results)
	var errs []error
	for err := range results {
		if err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// RemoveAll clears handlers for an event.
func (e *EventEmitter) RemoveAll(name string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	delete(e.handlers, name)
}

// Count returns the number of registered handlers for an event.
func (e *EventEmitter) Count(name string) int {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return len(e.handlers[name])
}
