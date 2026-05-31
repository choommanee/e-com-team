package store

import (
	"context"
	"errors"
	"testing"
	"time"

	"ecomteam/internal/domain"
)

func TestMemoryUserLifecycle(t *testing.T) {
	s := NewMemory()
	ctx := context.Background()
	u := domain.User{ID: "u1", Email: "A@B.com", PasswordHash: "h", CreatedAt: time.Now()}

	if err := s.CreateUser(ctx, u); err != nil {
		t.Fatalf("create: %v", err)
	}
	// Duplicate email (case-insensitive) rejected.
	if err := s.CreateUser(ctx, domain.User{ID: "u2", Email: "a@b.com"}); !errors.Is(err, ErrDuplicate) {
		t.Fatalf("expected ErrDuplicate, got %v", err)
	}
	got, err := s.GetUserByEmail(ctx, "a@b.com")
	if err != nil || got.ID != "u1" {
		t.Fatalf("get by email: %v / %+v", err, got)
	}
	if _, err := s.GetUserByID(ctx, "missing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestMemoryJobsOrderingAndPending(t *testing.T) {
	s := NewMemory()
	ctx := context.Background()
	old := domain.Job{ID: "j1", UserID: "u1", Status: domain.JobDone, CreatedAt: time.Now().Add(-time.Hour)}
	recent := domain.Job{ID: "j2", UserID: "u1", Status: domain.JobPending, CreatedAt: time.Now()}
	_ = s.CreateJob(ctx, old)
	_ = s.CreateJob(ctx, recent)

	list, _ := s.ListJobsByUser(ctx, "u1", 10, 0)
	if len(list) != 2 || list[0].ID != "j2" {
		t.Fatalf("expected newest-first ordering, got %+v", list)
	}
	pending, _ := s.ListPendingJobs(ctx)
	if len(pending) != 1 || pending[0].ID != "j2" {
		t.Fatalf("expected 1 pending job j2, got %+v", pending)
	}
}

func TestMemoryUsageIncrement(t *testing.T) {
	s := NewMemory()
	ctx := context.Background()
	n, _ := s.IncrementUsage(ctx, "u1", "2026-05")
	if n != 1 {
		t.Fatalf("expected 1, got %d", n)
	}
	n, _ = s.IncrementUsage(ctx, "u1", "2026-05")
	if n != 2 {
		t.Fatalf("expected 2, got %d", n)
	}
	// Different period resets.
	got, _ := s.GetUsage(ctx, "u1", "2026-06")
	if got != 0 {
		t.Fatalf("expected 0 for new period, got %d", got)
	}
}
