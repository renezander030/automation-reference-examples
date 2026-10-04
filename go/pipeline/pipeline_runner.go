package pipeline

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// Step defines a single pipeline step from YAML config.
type Step struct {
	Name   string            `yaml:"name"`
	Type   string            `yaml:"type"`   // "deterministic", "ai", "approval"
	Action string            `yaml:"action"` // for deterministic steps
	Skill  string            `yaml:"skill"`  // for AI steps
	Vars   map[string]string `yaml:"vars"`
}

// Pipeline defines a named sequence of steps.
type Pipeline struct {
	Name    string        `yaml:"name"`
	Timeout time.Duration `yaml:"timeout"`
	Steps   []Step        `yaml:"steps"`
}

// StepHandler processes a single step. Receives and returns a shared data map.
type StepHandler func(ctx context.Context, step Step, data map[string]interface{}) error

// Runner executes pipelines with registered step handlers.
type Runner struct {
	mu       sync.Mutex
	frozen   bool
	handlers map[string]StepHandler
}

func NewRunner() *Runner {
	return &Runner{handlers: make(map[string]StepHandler)}
}

// RegisterHandler maps a step type to its handler.
func (r *Runner) RegisterHandler(stepType string, h StepHandler) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.frozen {
		return fmt.Errorf("registry is frozen")
	}
	if stepType == "" || h == nil {
		return fmt.Errorf("step type and handler required")
	}
	r.handlers[stepType] = h
	return nil
}

// Run executes with cooperative cancellation: handlers must honor ctx.
func (r *Runner) Run(parent context.Context, p Pipeline, data map[string]interface{}) error {
	r.mu.Lock()
	r.frozen = true
	r.mu.Unlock()
	if p.Timeout < 0 {
		return fmt.Errorf("timeout must not be negative")
	}
	timeout := p.Timeout
	if timeout == 0 {
		timeout = 5 * time.Minute
	}
	ctx, cancel := context.WithTimeout(parent, timeout)
	defer cancel()

	log.Printf("[pipeline:%s] starting (%d steps, timeout %s)", p.Name, len(p.Steps), timeout)

	for _, step := range p.Steps {
		select {
		case <-ctx.Done():
			return fmt.Errorf("pipeline %s timed out at step %s", p.Name, step.Name)
		default:
		}

		handler, ok := r.handlers[step.Type]
		if !ok {
			return fmt.Errorf("no handler for step type %q in step %s", step.Type, step.Name)
		}

		log.Printf("[pipeline:%s][step:%s] type=%s", p.Name, step.Name, step.Type)
		if err := handler(ctx, step, data); err != nil {
			return fmt.Errorf("step %s failed: %w", step.Name, err)
		}
	}

	if err := ctx.Err(); err != nil {
		return err
	}
	log.Printf("[pipeline:%s] completed", p.Name)
	return nil
}
