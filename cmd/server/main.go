// Command server boots the E-commerce Listing Team SaaS.
package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"ecomteam/internal/affiliate"
	"ecomteam/internal/agents"
	"ecomteam/internal/api"
	"ecomteam/internal/auth"
	"ecomteam/internal/billing"
	"ecomteam/internal/config"
	"ecomteam/internal/domain"
	"ecomteam/internal/events"
	"ecomteam/internal/jobs"
	"ecomteam/internal/llm"
	"ecomteam/internal/store"
	"ecomteam/internal/subscription"
	"ecomteam/internal/web"
)

func main() {
	config.LoadDotenv(".env")
	cfg := config.Load()
	if err := cfg.Validate(); err != nil {
		log.Fatalf("config: %v", err)
	}

	rootCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// --- Store ---
	st := buildStore(rootCtx, cfg)
	defer st.Close()

	// --- Seed demo login ---
	seedUser(rootCtx, cfg, st)

	// --- LLM ---
	var client llm.Client
	if cfg.AIMode == "real" {
		// Real text + image, with a placeholder-image fallback so a missing
		// image entitlement (e.g. org verification) doesn't break the pipeline.
		var primary llm.Client
		switch cfg.AIProvider {
		case "gemini":
			primary = llm.NewGemini(cfg.GeminiAPIKey, cfg.GeminiModel, cfg.GeminiImage)
			log.Printf("AI: Gemini real (%s + %s, placeholder fallback on image error)", cfg.GeminiModel, cfg.GeminiImage)
		default:
			primary = llm.NewOpenAI(cfg.OpenAIAPIKey, cfg.OpenAIModel, cfg.OpenAIImage)
			log.Printf("AI: OpenAI real (%s + %s, placeholder fallback on image error)", cfg.OpenAIModel, cfg.OpenAIImage)
		}
		client = llm.WithImageFallback(primary, llm.NewMock())
	} else {
		client = llm.NewMock()
		log.Println("AI: mock")
	}

	// --- Billing ---
	var biller billing.Provider
	if cfg.BillingMode == "real" {
		biller = billing.NewLemonSqueezy(cfg.LemonSqueezyAPIKey, cfg.LemonSqueezyStoreID, cfg.LemonSqueezyWebhookSecret)
		log.Println("Billing: LemonSqueezy (real)")
	} else {
		biller = billing.NewMock(cfg.PublicBaseURL)
		log.Println("Billing: mock")
	}

	// --- Core services ---
	bus := events.New()
	catalog := subscription.NewCatalog(cfg.LSVariantPro, cfg.LSVariantBusiness)
	quota := subscription.NewQuota(st, catalog)
	orch := agents.NewOrchestrator(client)
	pool := jobs.New(orch, st, bus, quota, cfg.OutputDir, cfg.PublicBaseURL, cfg.WorkerCount)

	poolCtx, poolCancel := context.WithCancel(context.Background())
	go pool.Start(poolCtx)

	// --- Templates ---
	tpl, err := web.Templates()
	if err != nil {
		log.Fatalf("templates: %v", err)
	}

	srv := api.New(api.Deps{
		Config:    cfg,
		Store:     st,
		Tokens:    auth.NewTokenManager(cfg.JWTSecret, 7*24*time.Hour),
		Bus:       bus,
		Pool:      pool,
		Quota:     quota,
		Catalog:   catalog,
		Billing:   biller,
		Affiliate: affiliate.New(client),
		Templates: tpl,
	})

	httpServer := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           srv.Handler(web.StaticHandler()),
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		log.Printf("listening on http://localhost:%s  (open %s/dashboard)", cfg.Port, cfg.PublicBaseURL)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("http: %v", err)
		}
	}()

	<-rootCtx.Done()
	log.Println("shutting down...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = httpServer.Shutdown(shutdownCtx)
	poolCancel()
}

// buildStore returns a Postgres store when DATABASE_URL is set, otherwise an
// in-memory store so the app runs with zero external dependencies.
func buildStore(ctx context.Context, cfg config.Config) store.Store {
	if cfg.DatabaseURL == "" {
		log.Println("Store: in-memory (no DATABASE_URL)")
		return store.NewMemory()
	}
	pg, err := store.NewPostgres(ctx, cfg.DatabaseURL, store.Schema())
	if err != nil {
		log.Printf("Store: Postgres unavailable (%v); falling back to in-memory", err)
		return store.NewMemory()
	}
	log.Println("Store: Postgres")
	return pg
}

// seedUser creates the configured demo login on startup (idempotent) so there
// is always an account ready to sign in with.
func seedUser(ctx context.Context, cfg config.Config, st store.Store) {
	if cfg.SeedEmail == "" {
		return
	}
	if _, err := st.GetUserByEmail(ctx, cfg.SeedEmail); err == nil {
		log.Printf("Seed user already exists: %s", cfg.SeedEmail)
		return
	} else if !errors.Is(err, store.ErrNotFound) {
		log.Printf("Seed user check failed: %v", err)
		return
	}

	hash, err := auth.HashPassword(cfg.SeedPassword)
	if err != nil {
		log.Printf("Seed user: hash failed: %v", err)
		return
	}
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	u := domain.User{ID: hex.EncodeToString(b), Email: cfg.SeedEmail, PasswordHash: hash, CreatedAt: time.Now()}
	if err := st.CreateUser(ctx, u); err != nil {
		log.Printf("Seed user: create failed: %v", err)
		return
	}
	plan := domain.PlanID(cfg.SeedPlan)
	switch plan {
	case domain.PlanFree, domain.PlanPro, domain.PlanBusiness:
	default:
		plan = domain.PlanFree
	}
	_ = st.UpsertSubscription(ctx, domain.Subscription{
		UserID: u.ID, Plan: plan, Status: "active",
		PeriodStart: time.Now(), PeriodEnd: time.Now().AddDate(0, 1, 0),
	})
	log.Printf("✅ Seeded demo login → email: %s  password: %s  (plan: %s)", cfg.SeedEmail, cfg.SeedPassword, plan)
}
