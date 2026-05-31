# 🛒 E-COMMERCE HUB — AI Listing Team

A complete Go SaaS that turns a **product name** into a ready-to-publish marketplace
listing (Shopee/Lazada style) using a **6-agent AI pipeline**, served behind a REST API
with JWT auth, subscription billing (LemonSqueezy), per-plan quotas, Postgres persistence,
and a live **pixel-art dashboard** driven by Server-Sent Events.

> Runs out of the box in **mock mode** with **zero external keys** — no OpenAI, no database,
> no payment provider required. Flip two env vars to go live.

Full design spec: [`docs/superpowers/specs/2026-05-31-ecom-listing-team-saas-design.md`](docs/superpowers/specs/2026-05-31-ecom-listing-team-saas-design.md)

## The 6 agents

| Agent | Role |
|-------|------|
| **BENEFIT** | Extracts 3 core selling points |
| **PROMO** | Writes headline + promotion copy |
| **DESIGN** | Decides layout + color tone |
| **PROMPT** | Builds the detailed English image prompt |
| **STUDIO** | Generates the product image (gpt-image-1) |
| **QUALITY CHECK** | Validates the finished listing |

The orchestrator runs them in sequence, emitting live progress for each stage to the
dashboard over SSE — so the agent cards animate exactly like the reference UI.

## Quick start (mock mode, no keys)

```bash
make run
# open http://localhost:8080            (landing)
# open http://localhost:8080/dashboard  (app)
```

A **ready-to-use demo login** is created automatically in mock mode:

```
email:    demo@ecom.dev
password: demo1234        (Pro plan, 100 listings/month)
```

Set `SEED_EMAIL` / `SEED_PASSWORD` / `SEED_PLAN` to customize it, or `SEED_EMAIL=off`
to disable. You can also just register a fresh account from the dashboard.

In mock mode the agents return canned Thai copy and STUDIO renders a placeholder image,
so you can exercise the entire flow — auth, generation, live progress, quotas, mock
upgrade — without spending a cent.

## Run with Docker (app + Postgres)

```bash
docker compose up --build
# http://localhost:8080
```

## Go live (real AI + billing)

Copy `.env.example` to `.env` and set:

```bash
AI_MODE=real
OPENAI_API_KEY=sk-...
BILLING_MODE=real
LEMONSQUEEZY_API_KEY=...
LEMONSQUEEZY_STORE_ID=...
LEMONSQUEEZY_WEBHOOK_SECRET=...
LS_VARIANT_PRO=...        # LemonSqueezy variant id for the Pro plan
LS_VARIANT_BUSINESS=...   # ... and Business
DATABASE_URL=postgres://ecom:ecom@localhost:5432/ecom?sslmode=disable
JWT_SECRET=<long-random-string>
```

Point your LemonSqueezy webhook at `POST /webhooks/lemonsqueezy`.

## Plans

| Plan | Price | Listings / month |
|------|-------|------------------|
| Free | ฿0 | 5 |
| Pro | ฿299 | 100 |
| Business | ฿999 | 500 |

Quota is enforced with an atomic reserve/refund counter (failed jobs don't consume quota).

## API

| Method | Path | Auth | Purpose |
|--------|------|------|---------|
| POST | `/api/v1/auth/register` | – | create account (starts on Free) |
| POST | `/api/v1/auth/login` | – | get JWT |
| GET | `/api/v1/me` | ✓ | account + subscription + quota |
| POST | `/api/v1/listings` | ✓ | create generation job → `202` + job |
| GET | `/api/v1/listings/{id}` | ✓ | job status + result |
| GET | `/api/v1/listings` | ✓ | list your jobs |
| GET | `/api/v1/dashboard` | ✓ | KPI aggregate for the dashboard |
| GET | `/api/v1/events?token=…` | ✓ | **SSE** live agent progress |
| POST | `/api/v1/billing/checkout` | ✓ | LemonSqueezy checkout URL |
| POST | `/api/v1/billing/portal` | ✓ | customer portal URL |
| POST | `/webhooks/lemonsqueezy` | signed | subscription lifecycle events |
| GET | `/images/{file}` | – | generated product image |

### Example

```bash
TOKEN=$(curl -s localhost:8080/api/v1/auth/register \
  -d '{"email":"me@shop.com","password":"secret123"}' | jq -r .token)

JOB=$(curl -s localhost:8080/api/v1/listings -H "Authorization: Bearer $TOKEN" \
  -d '{"product_name":"ครีมกันแดด","lang":"th"}' | jq -r .id)

curl -s localhost:8080/api/v1/listings/$JOB -H "Authorization: Bearer $TOKEN" | jq
```

## Architecture

```
cmd/server            entrypoint + graceful shutdown
internal/agents       6 agents + orchestrator (progress callbacks)
internal/llm          Client interface · OpenAI · deterministic Mock
internal/jobs         worker pool (runs pipeline, persists image, emits events)
internal/events       in-memory pub/sub → SSE
internal/store        Store interface · in-memory · Postgres (pgx)
internal/auth         bcrypt + JWT
internal/subscription plan catalog + atomic quota
internal/billing      Provider interface · LemonSqueezy · Mock
internal/api          router, middleware, handlers, SSE, webhook, pages
internal/web          embedded pixel-art templates + static assets
tools/genassets       regenerates the pixel agent sprites
```

Both AI and billing sit behind interfaces with mock implementations, so the whole stack
is testable and runnable offline.

## Development

```bash
make test    # full unit + httptest suite (no external services)
make vet
make build   # -> bin/server
go run ./tools/genassets   # regenerate agent sprites
```

## License

MIT
