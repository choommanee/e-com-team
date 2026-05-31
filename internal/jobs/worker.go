// Package jobs runs listing-generation jobs on a pool of workers, persisting
// results and pushing live progress to the event bus.
package jobs

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"time"

	"ecomteam/internal/agents"
	"ecomteam/internal/domain"
	"ecomteam/internal/events"
	"ecomteam/internal/store"
	"ecomteam/internal/subscription"
)

// Pool processes jobs concurrently.
type Pool struct {
	orch          *agents.Orchestrator
	store         store.Store
	bus           *events.Bus
	quota         *subscription.Quota
	outputDir     string
	publicBaseURL string
	workers       int
	queue         chan string
	now           func() time.Time
}

// New builds a worker pool.
func New(orch *agents.Orchestrator, s store.Store, bus *events.Bus, q *subscription.Quota, outputDir, publicBaseURL string, workers int) *Pool {
	if workers < 1 {
		workers = 1
	}
	return &Pool{
		orch:          orch,
		store:         s,
		bus:           bus,
		quota:         q,
		outputDir:     outputDir,
		publicBaseURL: publicBaseURL,
		workers:       workers,
		queue:         make(chan string, 256),
		now:           time.Now,
	}
}

// Enqueue submits a job id for processing. Non-blocking unless the queue is full.
func (p *Pool) Enqueue(jobID string) {
	p.queue <- jobID
}

// Start launches the workers and requeues any jobs left pending/running from a
// previous run. It blocks until ctx is cancelled, then drains.
func (p *Pool) Start(ctx context.Context) {
	if err := os.MkdirAll(p.outputDir, 0o755); err != nil {
		log.Printf("jobs: cannot create output dir: %v", err)
	}
	// Requeue interrupted jobs.
	if pending, err := p.store.ListPendingJobs(ctx); err == nil {
		for _, j := range pending {
			select {
			case p.queue <- j.ID:
			default:
			}
		}
	}

	done := make(chan struct{})
	for i := 0; i < p.workers; i++ {
		go func() {
			for {
				select {
				case <-ctx.Done():
					done <- struct{}{}
					return
				case id := <-p.queue:
					p.process(ctx, id)
				}
			}
		}()
	}
	<-ctx.Done()
	for i := 0; i < p.workers; i++ {
		<-done
	}
}

// process runs a single job end-to-end.
func (p *Pool) process(ctx context.Context, jobID string) {
	job, err := p.store.GetJob(ctx, jobID)
	if err != nil {
		log.Printf("jobs: load %s: %v", jobID, err)
		return
	}

	job.Status = domain.JobRunning
	job.UpdatedAt = p.now()
	_ = p.store.UpdateJob(ctx, job)

	progress := func(agent string, percent int, task string) {
		p.bus.Publish(domain.Event{
			Type: domain.EventProgress, UserID: job.UserID, JobID: job.ID,
			Agent: agent, Percent: percent, Task: task,
		})
	}

	data, runErr := p.orch.Run(ctx, job.ProductName, job.Lang, progress)
	if runErr != nil {
		p.fail(ctx, job, runErr.Error())
		return
	}

	// Persist the generated image.
	imgPath := filepath.Join(p.outputDir, job.ID+".png")
	if err := os.WriteFile(imgPath, data.ImagePNG, 0o644); err != nil {
		p.fail(ctx, job, "could not save image: "+err.Error())
		return
	}
	listing := data.Listing
	listing.ImageURL = p.publicBaseURL + "/images/" + job.ID + ".png"

	job.Status = domain.JobDone
	job.Result = &listing
	job.ImagePath = imgPath
	job.Error = ""
	job.UpdatedAt = p.now()
	if err := p.store.UpdateJob(ctx, job); err != nil {
		log.Printf("jobs: persist %s: %v", job.ID, err)
	}

	p.bus.Publish(domain.Event{
		Type: domain.EventJobDone, UserID: job.UserID, JobID: job.ID, Payload: listing,
	})
}

// fail marks a job failed, refunds the reserved quota, and notifies the user.
func (p *Pool) fail(ctx context.Context, job domain.Job, msg string) {
	job.Status = domain.JobFailed
	job.Error = msg
	job.UpdatedAt = p.now()
	_ = p.store.UpdateJob(ctx, job)
	// Per spec, failed jobs do not consume quota.
	p.quota.Refund(ctx, job.UserID, job.CreatedAt)
	p.bus.Publish(domain.Event{
		Type: domain.EventJobFail, UserID: job.UserID, JobID: job.ID, Task: msg,
	})
	log.Printf("jobs: job %s failed: %s", job.ID, msg)
}
