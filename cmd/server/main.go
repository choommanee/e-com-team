// Command server boots the E-commerce Listing Team SaaS.
package main

import (
	"context"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"ecomteam/internal/agents"
	"ecomteam/internal/api"
	"ecomteam/internal/auth"
	"ecomteam/internal/billing"
	"ecomteam/internal/config"
	"ecomteam/internal/events"
	"ecomteam/internal/jobs"
	"ecomteam/internal/llm"
	"ecomteam/internal/store"
	"ecomteam/internal/subscription"
	"ecomteam/internal/web"
)

func main() {
	cfg := config.Load()
	if err := cfg.Validate(); err != nil {
		log.Fatalf("config: %v", err)
	}

	rootCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// --- Store ---
	st := buildStore(rootCtx, cfg)
	defer st.Close()

	// --- LLM ---
	var client llm.Client
	if cfg.AIMode == "real" {
		client = llm.NewOpenAI(cfg.OpenAIAPIKey, cfg.OpenAIModel, cfg.OpenAIImage)
		log.Println("AI: OpenAI (real)")
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
