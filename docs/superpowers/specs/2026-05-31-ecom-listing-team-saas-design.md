# E-commerce Listing Team SaaS — Design

**Date:** 2026-05-31
**Status:** Approved (pending user spec review)

## 1. Purpose

A SaaS that turns a **product name** into a complete, ready-to-publish marketplace listing
(Shopee/Lazada style): selling points, promo copy, layout direction, an English image prompt,
and a finished **product image with promotional text baked in**.

Five AI agents run as a pipeline, powered by OpenAI. The service is offered as a monthly
**subscription** with per-plan generation quotas, billed through LemonSqueezy.

Target user: Thai online sellers who lack good product photography.

## 2. The 6-agent pipeline

Sequential, each stage consumes the previous stage's typed output:

| Agent | Role | Output |
|---|---|---|
| **BENEFIT** | E-commerce strategist | 3 core selling points that solve a customer problem |
| **PROMO** | Sales copywriter | Headline + promotion text (e.g. "ซื้อ 1 แถม 1 ส่งฟรี") |
| **DESIGN** | Graphic designer | Layout + color tone direction |
| **PROMPT** | Prompt engineer | Detailed English image prompt incl. exact on-image text |
| **STUDIO** | Image generator | Final image via `gpt-image-1`, saved to disk |
| **QUALITY CHECK** | QA reviewer | Validates the listing (text legible, price/promo consistent, image present); marks `passed`/`needs_fix` |

Agents 1–4 + QC call OpenAI **chat completions** (text). STUDIO calls OpenAI **image generation**
(`gpt-image-1`). All LLM access goes behind an `llm.Client` interface so it can be mocked in tests.

The **Orchestrator** wires the stages with per-step timeout + bounded retry, logs each stage,
emits a **progress event** per stage (used by the live dashboard), and produces a single
`Listing` result struct. Each stage reports `0→100%` so the UI agent cards can animate like
the reference dashboard.

## 3. Output conventions

- **Image:** saved to `./output/<job_id>.png` (path configurable); API returns a served URL
  `/images/<job_id>.png`. Path also stored in DB.
- **Language:** copy (selling points, headline, promo) defaults to **Thai**; the image prompt
  is English (model requirement). Configurable per request via `lang` (`th` default, `en`).

## 4. Architecture

Tech stack:
- **Go 1.24**, standard-library `net/http` with Go 1.22+ pattern routing (no heavy framework)
- **OpenAI**: GPT for text, `gpt-image-1` for images
- **PostgreSQL**: users, subscriptions, jobs, usage
- **LemonSqueezy**: subscription billing (merchant of record), behind a `billing.Provider` interface
- **Auth**: JWT (sessions) + bcrypt password hashing
- **Frontend**: a **pixel-art retro-game dashboard** matching the "E-COMMERCE HUB v2.0"
  reference — dark navy background, neon cyan/green/orange accents, circuit-board borders,
  pixel fonts (`Press Start 2P` for headings, `VT323` for body), animated progress bars and a
  live clock. Server-rendered HTML + hand-written pixel CSS (no Tailwind), a little vanilla JS +
  **SSE** for live updates, and `Chart.js` for the sales line chart. See §6a for the layout.
- **Brand visuals**: pixel-art agent sprites (the 6 agents at their desks), pixel product
  thumbnails, the "E-COMMERCE HUB" logo, and circuit-board background tiles are **AI-generated**
  (image-generation skills, pixel-art style) and shipped as static assets under
  `internal/web/static/`.

Project layout:
```
cmd/server/main.go          # entrypoint, wiring, graceful shutdown
internal/
  agents/                   # benefit.go promo.go design.go prompt.go studio.go qualitycheck.go orchestrator.go
  llm/                      # llm.Client interface + openai.go + mock.go
  api/                      # router.go handlers_*.go middleware.go
  auth/                     # jwt.go password.go
  billing/                  # provider.go lemonsqueezy.go mock.go webhook.go
  subscription/             # plans.go quota.go
  store/                    # postgres.go users.go subscriptions.go jobs.go usage.go
  jobs/                     # in-process worker pool + queue
  config/                   # config.go (env)
  web/                      # templates + static (landing, dashboard)
migrations/                 # NNNN_*.sql
docker-compose.yml          # app + postgres
.env.example
Makefile
```

## 5. Request flow (async)

Image generation takes ~20–40s, so listing creation is **asynchronous**:

1. `POST /api/v1/listings {product_name, lang?, options?}` with `Authorization: Bearer <jwt>`
2. Auth middleware resolves the user
3. Quota middleware checks the user's plan quota for the current period (atomic, DB-backed)
4. A `job` row is inserted (`status=pending`); `job_id` returned immediately (HTTP 202)
5. A worker goroutine runs the orchestrator pipeline; status moves `pending → running → done|failed`
6. As each agent runs, the orchestrator emits **stage progress events** (`{job_id, agent, percent, task}`)
   onto an in-memory bus; the dashboard receives them over **SSE** (`/api/v1/events`) and animates
   the agent cards live (no polling needed)
7. On success: image saved, `Listing` JSON stored on the job row, usage counter incremented
8. `GET /api/v1/listings/{id}` returns the final result (SSE is the live channel; polling still works as fallback)

Worker pool is in-process (configurable size), backed by the DB job table so restarts can
requeue `pending`/`running` jobs.

## 6. API endpoints

| Method | Path | Auth | Purpose |
|---|---|---|---|
| POST | `/api/v1/auth/register` | public | create account |
| POST | `/api/v1/auth/login` | public | get JWT |
| GET | `/api/v1/me` | user | account + subscription + usage |
| POST | `/api/v1/listings` | user | create generation job (202 + job_id) |
| GET | `/api/v1/listings/{id}` | user | job status + result |
| GET | `/api/v1/listings` | user | list user's jobs (paginated) |
| GET | `/api/v1/events` | user | **SSE** stream of live agent progress + stats |
| GET | `/api/v1/dashboard` | user | dashboard aggregate (KPIs, order summary, sales series) |
| POST | `/api/v1/billing/checkout` | user | LemonSqueezy checkout URL for a plan |
| POST | `/api/v1/billing/portal` | user | customer portal URL |
| POST | `/webhooks/lemonsqueezy` | signed | subscription lifecycle events |
| GET | `/images/{file}` | public | serve generated image |
| GET | `/` `/dashboard` | web | landing + dashboard pages |

## 6a. Pixel dashboard layout (matches "E-COMMERCE HUB v2.0")

Single authenticated screen, grid layout, pixel theme:

- **Header bar** — "E-COMMERCE HUB v2.0" wordmark + live `TIME` clock (right).
- **Left column — TEAM STATUS** — search box, `[ALL] [WORKING] [IDLE]` filter, and the 6 agents
  as sprite rows with `WORKING/IDLE` label + animated green progress bar (driven by SSE).
- **Center top — PRODUCT LISTING** — grid of generated listings as pixel cards (discount badge,
  thumbnail, name, price, stock), `+ ADD PRODUCT` button (opens "new listing" form), pagination.
- **Center bottom — SHOP DASHBOARD** — SALES OVERVIEW line chart (7 days, Chart.js) + ORDER
  SUMMARY (total / pending / shipped / cancelled / avg order value) + TOTAL SALES / VISITORS /
  CONVERSION tiles with trend arrows.
- **Right column — AGENT WORK CARDS** — one card per agent (BENEFIT, PROMO, DESIGN, PROMPT,
  STUDIO, QUALITY CHECK) showing the current task text + live progress % for the active job.
- **Bottom KPI bar** — OVERALL SALES, TOTAL ORDERS, UNITS SOLD, ACTIVE PRODUCTS, SHOP RATING,
  RESPONSE RATE, each with a trend percentage.

Stats (orders, sales, ratings, etc.) are computed from the user's real job/listing data; for a
brand-new account they start at zero and grow as listings are created. The agent cards and team
status reflect **real pipeline progress** of in-flight jobs via SSE — this is what makes it
"work like the image", not a static mockup.

## 7. Subscription & quota

Plans (configurable in `subscription/plans.go`):

| Plan | Price | Listings / month |
|---|---|---|
| Free | ฿0 | 5 |
| Pro | ฿299 | 100 |
| Business | ฿999 | 500 |

- Each plan maps to a LemonSqueezy variant ID (from env/config).
- Usage tracked per user per billing period in `usage` table; quota middleware does an atomic
  check-and-increment, returns HTTP 402/429 when exceeded.
- Webhook events (`subscription_created`, `_updated`, `_cancelled`, `_expired`) update the
  user's plan + period in the `subscriptions` table.
- Webhook signature verified with the LemonSqueezy signing secret.

## 8. Data model (Postgres)

- `users` — id, email (unique), password_hash, created_at
- `subscriptions` — user_id, plan, status, ls_subscription_id, period_start, period_end
- `jobs` — id, user_id, product_name, lang, status, result_json, image_path, error, timestamps
- `usage` — user_id, period_start, count (unique on user_id+period_start)

## 9. Error handling

- LLM errors: bounded retry (e.g. 2) with backoff per stage; on final failure the job is marked
  `failed` with a sanitized error message; usage is **not** counted for failed jobs.
- Quota exceeded: `402 Payment Required` with upgrade hint.
- Auth failures: `401`. Validation: `400` with field detail.
- Webhook: invalid signature → `401`; unknown event → `200` ignored (idempotent by event id).
- Graceful shutdown drains in-flight workers.

## 10. Testing

- **agents/orchestrator**: unit tests with a `mock` `llm.Client` (deterministic, no API spend)
- **billing**: `mock` `billing.Provider`; webhook handler tested with crafted signed payloads
- **subscription/quota**: table-driven tests for limit boundaries + period rollover
- **api**: `httptest` handler tests with an in-memory/sqlite or test-Postgres store
- **store**: repo tests against a Postgres test container (or skipped via build tag if unavailable)

## 11. Out of scope (first round)

- Real-time push (use polling)
- Multi-image variations per listing (single image first)
- Team/multi-seat accounts
- Admin panel beyond the basic dashboard
- Non-OpenAI image backends

## 12. Configuration (.env)

`OPENAI_API_KEY`, `DATABASE_URL`, `JWT_SECRET`, `LEMONSQUEEZY_API_KEY`,
`LEMONSQUEEZY_STORE_ID`, `LEMONSQUEEZY_WEBHOOK_SECRET`, `LS_VARIANT_PRO`, `LS_VARIANT_BUSINESS`,
`OUTPUT_DIR`, `PUBLIC_BASE_URL`, `PORT`, `WORKER_COUNT`.
A `mock` mode (`AI_MODE=mock`, `BILLING_MODE=mock`) lets the whole stack run with no external keys.
