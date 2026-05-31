package jobs

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"ecomteam/internal/agents"
	"ecomteam/internal/domain"
	"ecomteam/internal/events"
	"ecomteam/internal/llm"
	"ecomteam/internal/store"
	"ecomteam/internal/subscription"
)

func newTestPool(t *testing.T) (*Pool, *store.Memory, *events.Bus, string) {
	t.Helper()
	s := store.NewMemory()
	bus := events.New()
	q := subscription.NewQuota(s, subscription.NewCatalog("", ""))
	orch := agents.NewOrchestrator(llm.NewMock())
	dir := t.TempDir()
	p := New(orch, s, bus, q, dir, "http://localhost:8080", 2)
	return p, s, bus, dir
}

func TestProcessJobSuccess(t *testing.T) {
	p, s, bus, dir := newTestPool(t)
	ctx := context.Background()

	_ = s.CreateUser(ctx, domain.User{ID: "u1", Email: "a@b.com"})
	job := domain.Job{ID: "j1", UserID: "u1", ProductName: "ครีมกันแดด", Lang: "th",
		Status: domain.JobPending, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	_ = s.CreateJob(ctx, job)

	ch, unsub := bus.Subscribe("u1")
	defer unsub()

	p.process(ctx, "j1")

	got, _ := s.GetJob(ctx, "j1")
	if got.Status != domain.JobDone {
		t.Fatalf("expected done, got %s (err=%s)", got.Status, got.Error)
	}
	if got.Result == nil || got.Result.ImageURL == "" {
		t.Fatalf("expected listing with image url, got %+v", got.Result)
	}
	if _, err := os.Stat(filepath.Join(dir, "j1.png")); err != nil {
		t.Fatalf("expected image file written: %v", err)
	}

	// A job_done event must be delivered.
	if !drainFor(ch, domain.EventJobDone, 2*time.Second) {
		t.Fatal("expected a job_done event")
	}
}

func TestProcessJobFailureRefundsQuota(t *testing.T) {
	p, s, _, _ := newTestPool(t)
	ctx := context.Background()
	_ = s.CreateUser(ctx, domain.User{ID: "u1", Email: "a@b.com"})

	// Reserve one unit as the API would, then force failure via a broken orchestrator.
	now := time.Now()
	_, _, _ = p.quota.Reserve(ctx, "u1", domain.PlanFree, now)
	p.orch = agents.NewOrchestrator(failingClient{})

	job := domain.Job{ID: "j2", UserID: "u1", ProductName: "x", Lang: "th",
		Status: domain.JobPending, CreatedAt: now, UpdatedAt: now}
	_ = s.CreateJob(ctx, job)

	p.process(ctx, "j2")

	got, _ := s.GetJob(ctx, "j2")
	if got.Status != domain.JobFailed {
		t.Fatalf("expected failed, got %s", got.Status)
	}
	if used, _ := s.GetUsage(ctx, "u1", subscription.PeriodKey(now)); used != 0 {
		t.Fatalf("expected quota refunded to 0, got %d", used)
	}
}

func drainFor(ch <-chan domain.Event, want domain.EventType, d time.Duration) bool {
	deadline := time.After(d)
	for {
		select {
		case e := <-ch:
			if e.Type == want {
				return true
			}
		case <-deadline:
			return false
		}
	}
}

// failingClient makes the first agent (Chat) fail so the orchestrator errors.
type failingClient struct{}

func (failingClient) Chat(context.Context, string, string) (string, error) {
	return "", errBoom
}
func (failingClient) Image(context.Context, string) ([]byte, error) { return nil, errBoom }

var errBoom = &boomErr{}

type boomErr struct{}

func (*boomErr) Error() string { return "boom" }

var _ llm.Client = failingClient{}
