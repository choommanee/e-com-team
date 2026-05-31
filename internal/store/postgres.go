package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"ecomteam/internal/domain"
)

// Postgres is a pgx-backed Store.
type Postgres struct {
	pool *pgxpool.Pool
}

// NewPostgres connects to the database and runs the embedded migration.
func NewPostgres(ctx context.Context, dsn, migrationSQL string) (*Postgres, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("connect: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping: %w", err)
	}
	if migrationSQL != "" {
		if _, err := pool.Exec(ctx, migrationSQL); err != nil {
			pool.Close()
			return nil, fmt.Errorf("migrate: %w", err)
		}
	}
	return &Postgres{pool: pool}, nil
}

func (p *Postgres) Close() { p.pool.Close() }

func (p *Postgres) CreateUser(ctx context.Context, u domain.User) error {
	_, err := p.pool.Exec(ctx,
		`INSERT INTO users (id, email, password_hash, created_at) VALUES ($1,$2,$3,$4)`,
		u.ID, u.Email, u.PasswordHash, u.CreatedAt)
	if err != nil && isUniqueViolation(err) {
		return ErrDuplicate
	}
	return err
}

func (p *Postgres) GetUserByEmail(ctx context.Context, email string) (domain.User, error) {
	return p.scanUser(ctx, `SELECT id,email,password_hash,created_at FROM users WHERE lower(email)=lower($1)`, email)
}

func (p *Postgres) GetUserByID(ctx context.Context, id string) (domain.User, error) {
	return p.scanUser(ctx, `SELECT id,email,password_hash,created_at FROM users WHERE id=$1`, id)
}

func (p *Postgres) scanUser(ctx context.Context, q string, arg string) (domain.User, error) {
	var u domain.User
	err := p.pool.QueryRow(ctx, q, arg).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.User{}, ErrNotFound
	}
	return u, err
}

func (p *Postgres) GetSubscription(ctx context.Context, userID string) (domain.Subscription, error) {
	var s domain.Subscription
	err := p.pool.QueryRow(ctx,
		`SELECT user_id,plan,status,ls_subscription,period_start,period_end FROM subscriptions WHERE user_id=$1`, userID).
		Scan(&s.UserID, &s.Plan, &s.Status, &s.LSSubscription, &s.PeriodStart, &s.PeriodEnd)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Subscription{}, ErrNotFound
	}
	return s, err
}

func (p *Postgres) UpsertSubscription(ctx context.Context, s domain.Subscription) error {
	_, err := p.pool.Exec(ctx,
		`INSERT INTO subscriptions (user_id,plan,status,ls_subscription,period_start,period_end)
		 VALUES ($1,$2,$3,$4,$5,$6)
		 ON CONFLICT (user_id) DO UPDATE SET
		   plan=EXCLUDED.plan, status=EXCLUDED.status, ls_subscription=EXCLUDED.ls_subscription,
		   period_start=EXCLUDED.period_start, period_end=EXCLUDED.period_end`,
		s.UserID, s.Plan, s.Status, s.LSSubscription, s.PeriodStart, s.PeriodEnd)
	return err
}

func (p *Postgres) CreateJob(ctx context.Context, j domain.Job) error {
	_, err := p.pool.Exec(ctx,
		`INSERT INTO jobs (id,user_id,product_name,lang,status,image_path,error,created_at,updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		j.ID, j.UserID, j.ProductName, j.Lang, j.Status, j.ImagePath, j.Error, j.CreatedAt, j.UpdatedAt)
	return err
}

func (p *Postgres) UpdateJob(ctx context.Context, j domain.Job) error {
	var resultJSON []byte
	if j.Result != nil {
		b, err := json.Marshal(j.Result)
		if err != nil {
			return err
		}
		resultJSON = b
	}
	tag, err := p.pool.Exec(ctx,
		`UPDATE jobs SET status=$2, result_json=$3, image_path=$4, error=$5, updated_at=$6 WHERE id=$1`,
		j.ID, j.Status, resultJSON, j.ImagePath, j.Error, j.UpdatedAt)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (p *Postgres) GetJob(ctx context.Context, id string) (domain.Job, error) {
	return p.scanJob(ctx, `SELECT id,user_id,product_name,lang,status,result_json,image_path,error,created_at,updated_at FROM jobs WHERE id=$1`, id)
}

func (p *Postgres) scanJob(ctx context.Context, q, arg string) (domain.Job, error) {
	var j domain.Job
	var resultJSON []byte
	err := p.pool.QueryRow(ctx, q, arg).Scan(
		&j.ID, &j.UserID, &j.ProductName, &j.Lang, &j.Status, &resultJSON, &j.ImagePath, &j.Error, &j.CreatedAt, &j.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Job{}, ErrNotFound
	}
	if err != nil {
		return domain.Job{}, err
	}
	if len(resultJSON) > 0 {
		var l domain.Listing
		if err := json.Unmarshal(resultJSON, &l); err == nil {
			j.Result = &l
		}
	}
	return j, nil
}

func (p *Postgres) ListJobsByUser(ctx context.Context, userID string, limit, offset int) ([]domain.Job, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := p.pool.Query(ctx,
		`SELECT id,user_id,product_name,lang,status,result_json,image_path,error,created_at,updated_at
		 FROM jobs WHERE user_id=$1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		userID, limit, offset)
	if err != nil {
		return nil, err
	}
	return collectJobs(rows)
}

func (p *Postgres) ListPendingJobs(ctx context.Context) ([]domain.Job, error) {
	rows, err := p.pool.Query(ctx,
		`SELECT id,user_id,product_name,lang,status,result_json,image_path,error,created_at,updated_at
		 FROM jobs WHERE status IN ('pending','running') ORDER BY created_at ASC`)
	if err != nil {
		return nil, err
	}
	return collectJobs(rows)
}

func collectJobs(rows pgx.Rows) ([]domain.Job, error) {
	defer rows.Close()
	var out []domain.Job
	for rows.Next() {
		var j domain.Job
		var resultJSON []byte
		if err := rows.Scan(&j.ID, &j.UserID, &j.ProductName, &j.Lang, &j.Status,
			&resultJSON, &j.ImagePath, &j.Error, &j.CreatedAt, &j.UpdatedAt); err != nil {
			return nil, err
		}
		if len(resultJSON) > 0 {
			var l domain.Listing
			if err := json.Unmarshal(resultJSON, &l); err == nil {
				j.Result = &l
			}
		}
		out = append(out, j)
	}
	return out, rows.Err()
}

func (p *Postgres) GetUsage(ctx context.Context, userID, periodStart string) (int, error) {
	var count int
	err := p.pool.QueryRow(ctx,
		`SELECT count FROM usage WHERE user_id=$1 AND period_start=$2`, userID, periodStart).Scan(&count)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, nil
	}
	return count, err
}

func (p *Postgres) IncrementUsage(ctx context.Context, userID, periodStart string) (int, error) {
	var count int
	err := p.pool.QueryRow(ctx,
		`INSERT INTO usage (user_id, period_start, count) VALUES ($1,$2,1)
		 ON CONFLICT (user_id, period_start) DO UPDATE SET count = usage.count + 1
		 RETURNING count`,
		userID, periodStart).Scan(&count)
	return count, err
}

func (p *Postgres) DecrementUsage(ctx context.Context, userID, periodStart string) error {
	_, err := p.pool.Exec(ctx,
		`UPDATE usage SET count = GREATEST(count - 1, 0) WHERE user_id=$1 AND period_start=$2`,
		userID, periodStart)
	return err
}

// isUniqueViolation reports whether err is a Postgres unique-constraint error.
func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	// pgx surfaces *pgconn.PgError with Code 23505 for unique violations; match
	// on the substring to avoid importing pgconn just for the code.
	msg := err.Error()
	return strings.Contains(msg, "23505") || strings.Contains(msg, "duplicate key")
}
