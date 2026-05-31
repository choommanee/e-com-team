package agents

import (
	"context"
	"fmt"
	"time"

	"ecomteam/internal/domain"
	"ecomteam/internal/llm"
)

// Orchestrator runs the 6 agents in order, reporting progress for each.
type Orchestrator struct {
	client      llm.Client
	stages      []Agent
	stageTimeout time.Duration
	maxRetries  int
}

// NewOrchestrator builds the standard 6-stage pipeline.
func NewOrchestrator(client llm.Client) *Orchestrator {
	return &Orchestrator{
		client: client,
		stages: []Agent{
			Benefit{}, Promo{}, Design{}, Prompt{}, Studio{}, QualityCheck{},
		},
		stageTimeout: 60 * time.Second,
		maxRetries:   2,
	}
}

// Stages returns the agent display names in execution order.
func (o *Orchestrator) Stages() []string {
	names := make([]string, len(o.stages))
	for i, s := range o.stages {
		names[i] = s.Name()
	}
	return names
}

// Run executes the full pipeline. The returned StageData carries the finished
// Listing and the generated image bytes. Progress is reported via p.
func (o *Orchestrator) Run(ctx context.Context, productName, lang string, p Progress) (*StageData, error) {
	if p == nil {
		p = func(string, int, string) {}
	}
	d := &StageData{
		ProductName: productName,
		Lang:        lang,
		Listing:     domain.Listing{ProductName: productName},
	}
	for _, stage := range o.stages {
		p(stage.Name(), 5, "Starting")
		if err := o.runStage(ctx, stage, d, p); err != nil {
			return d, fmt.Errorf("stage %s failed: %w", stage.Name(), err)
		}
	}
	return d, nil
}

// runStage runs one agent with a per-stage timeout and bounded retry.
func (o *Orchestrator) runStage(ctx context.Context, stage Agent, d *StageData, p Progress) error {
	var lastErr error
	for attempt := 0; attempt <= o.maxRetries; attempt++ {
		stageCtx, cancel := context.WithTimeout(ctx, o.stageTimeout)
		err := stage.Run(stageCtx, d, o.client, p)
		cancel()
		if err == nil {
			return nil
		}
		lastErr = err
		// Don't retry if the parent context is done.
		if ctx.Err() != nil {
			return ctx.Err()
		}
	}
	return lastErr
}
