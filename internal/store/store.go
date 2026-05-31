// Package store defines persistence for users, subscriptions, jobs and usage,
// with an in-memory implementation (zero-dependency) and a Postgres one.
package store

import (
	"context"
	"errors"

	"ecomteam/internal/domain"
)

// ErrNotFound is returned when a requested row does not exist.
var ErrNotFound = errors.New("not found")

// ErrDuplicate is returned when a unique constraint (e.g. email) is violated.
var ErrDuplicate = errors.New("duplicate")

// Store is the persistence interface used by the rest of the app.
type Store interface {
	// Users
	CreateUser(ctx context.Context, u domain.User) error
	GetUserByEmail(ctx context.Context, email string) (domain.User, error)
	GetUserByID(ctx context.Context, id string) (domain.User, error)

	// Subscriptions
	GetSubscription(ctx context.Context, userID string) (domain.Subscription, error)
	UpsertSubscription(ctx context.Context, s domain.Subscription) error

	// Jobs
	CreateJob(ctx context.Context, j domain.Job) error
	GetJob(ctx context.Context, id string) (domain.Job, error)
	UpdateJob(ctx context.Context, j domain.Job) error
	ListJobsByUser(ctx context.Context, userID string, limit, offset int) ([]domain.Job, error)
	ListPendingJobs(ctx context.Context) ([]domain.Job, error)

	// Usage (per user, per billing period start)
	GetUsage(ctx context.Context, userID string, periodStart string) (int, error)
	IncrementUsage(ctx context.Context, userID string, periodStart string) (int, error)
	// DecrementUsage refunds one unit (never below zero). Used to roll back a
	// reservation when a request is rejected or a job ultimately fails.
	DecrementUsage(ctx context.Context, userID string, periodStart string) error

	// Close releases resources (no-op for in-memory).
	Close()
}
