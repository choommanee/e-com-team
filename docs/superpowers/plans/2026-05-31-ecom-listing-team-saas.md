# E-commerce Listing Team SaaS — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a complete, runnable Go SaaS that turns a product name into a full marketplace listing (copy + promo + AI image) via a 6-agent pipeline, served behind a REST API with JWT auth, LemonSqueezy subscriptions + quotas, Postgres persistence, a live SSE-driven pixel-art dashboard, and a mock mode that runs with zero external keys.

**Architecture:** Hexagonal-ish. `llm.Client` and `billing.Provider` are interfaces with real (OpenAI / LemonSqueezy) and mock implementations. An in-process worker pool runs the orchestrator pipeline; stage progress is pushed to a pub/sub event bus consumed over SSE. Postgres via `pgx`. Standard-library `net/http` with Go 1.22 routing. Server-rendered pixel-art HTML templates + Chart.js + EventSource.

**Tech Stack:** Go 1.24, `jackc/pgx/v5`, `golang-jwt/jwt/v5`, `golang.org/x/crypto/bcrypt`, `go-chi`-free std router, Chart.js (CDN), Press Start 2P / VT323 fonts (CDN), Docker Compose (app + postgres).

---

## File structure

```
go.mod
.env.example
Makefile
docker-compose.yml
Dockerfile
cmd/server/main.go                  # wiring + graceful shutdown
internal/
  config/config.go                  # env → Config struct, AI_MODE/BILLING_MODE
  domain/domain.go                   # core types: Listing, Job, User, Subscription, Plan, events
  llm/llm.go                         # Client interface (Chat, Image)
  llm/openai.go                      # OpenAI impl (chat + gpt-image-1)
  llm/mock.go                        # deterministic mock
  agents/agent.go                    # Agent interface, StageResult, progress callback
  agents/benefit.go
  agents/promo.go
  agents/design.go
  agents/prompt.go
  agents/studio.go                   # writes image file
  agents/qualitycheck.go
  agents/orchestrator.go             # runs 6 stages, emits progress, builds Listing
  events/bus.go                      # pub/sub bus for progress + stats events
  jobs/worker.go                     # worker pool, picks pending jobs, runs orchestrator
  store/store.go                     # Store interface
  store/postgres.go                  # pgx pool + migrations runner
  store/users.go store/subscriptions.go store/jobs.go store/usage.go
  auth/password.go                   # bcrypt hash/verify
  auth/jwt.go                        # issue/parse JWT
  subscription/plans.go              # plan catalog + limits + LS variant map
  subscription/quota.go              # check-and-increment usage
  billing/billing.go                 # Provider interface
  billing/lemonsqueezy.go            # checkout/portal/webhook verify
  billing/mock.go
  api/server.go                      # router, mux, static, templates
  api/middleware.go                  # auth, quota, recover, request-id
  api/handlers_auth.go
  api/handlers_listings.go
  api/handlers_billing.go
  api/handlers_dashboard.go          # /api/v1/dashboard aggregate
  api/handlers_events.go             # SSE
  api/handlers_webhook.go
  api/render.go                      # JSON + template helpers
  web/templates/*.html               # landing.html, dashboard.html, layout
  web/static/css/pixel.css
  web/static/js/dashboard.js
  web/static/img/*                   # AI-generated pixel assets
migrations/0001_init.sql
```

## Build order (tasks)

Each task: write code, build/test, commit. TDD where logic is non-trivial (agents, quota, jwt, orchestrator, webhook signature). Glue (handlers, wiring) verified by `go build` + `go vet` + targeted httptest.

- **Task 1 — Module bootstrap:** `go.mod`, `config`, `domain` types, `Makefile`, `.env.example`. Build green.
- **Task 2 — LLM layer:** `llm.Client` interface + mock + OpenAI impl. Unit test mock determinism.
- **Task 3 — Agents + orchestrator:** 6 agents, `Agent` interface, orchestrator with progress callback. TDD orchestrator against mock LLM (asserts 6 stages, progress 0→100, Listing fields populated). Studio writes a real PNG (mock returns a 1×1 base64 in mock mode).
- **Task 4 — Events bus:** pub/sub with subscribe/publish/unsubscribe. TDD fan-out + unsubscribe.
- **Task 5 — Auth:** bcrypt password + JWT issue/parse. TDD round-trip + tamper/expiry.
- **Task 6 — Store:** Store interface + pgx Postgres impl + migrations. Repo methods for users/subs/jobs/usage. (Integration test behind `//go:build integration` tag; default build just compiles.)
- **Task 7 — Subscription + quota:** plan catalog, quota check-and-increment. TDD limits + period rollover with a fake store.
- **Task 8 — Billing:** Provider interface + mock + LemonSqueezy (checkout/portal/webhook HMAC verify). TDD webhook signature verify + mock checkout.
- **Task 9 — Jobs worker:** worker pool pulls pending jobs, runs orchestrator, persists result, increments usage, publishes events. TDD single job lifecycle with fakes.
- **Task 10 — API:** server/router, middleware (auth/quota/recover), handlers (auth, listings, dashboard, events SSE, billing, webhook), render helpers. httptest for register→login→create listing→poll.
- **Task 11 — Pixel frontend:** layout/landing/dashboard templates, pixel.css, dashboard.js (EventSource + Chart.js). 
- **Task 12 — AI-generated pixel assets:** logo, 6 agent sprites, hero, product placeholders → `web/static/img/`.
- **Task 13 — Packaging:** Dockerfile, docker-compose (app+postgres), README with run instructions (real + mock mode). Final `go build ./... && go vet ./... && go test ./...` green.

## Verification gates (per task)

- `go build ./...` succeeds
- `go vet ./...` clean
- `go test ./...` passes (integration-tagged tests excluded unless DB present)
- Final: `make run` boots in mock mode with no keys; dashboard loads; creating a listing animates agents and yields an image.

## Key interfaces (locked)

```go
// llm
type Client interface {
    Chat(ctx context.Context, system, user string) (string, error)
    Image(ctx context.Context, prompt string) (pngBytes []byte, err error)
}

// agents
type Progress func(agent string, percent int, task string)
type Agent interface {
    Name() string
    Run(ctx context.Context, in StageInput, p Progress) (StageOutput, error)
}

// events
type Event struct { Type, JobID, Agent, Task string; Percent int; Payload any }
type Bus interface {
    Publish(Event)
    Subscribe(userID string) (<-chan Event, func())
}

// billing
type Provider interface {
    Checkout(ctx context.Context, userID, email, plan string) (url string, err error)
    PortalURL(ctx context.Context, subID string) (string, error)
    VerifyWebhook(body []byte, sig string) (bool)
    ParseEvent(body []byte) (WebhookEvent, error)
}
```

These names are final and referenced across tasks.
