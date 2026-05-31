// Package config loads runtime configuration from environment variables.
package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config holds all runtime configuration for the server.
type Config struct {
	Port          string
	PublicBaseURL string
	OutputDir     string
	WorkerCount   int

	// Modes: "real" or "mock". Mock lets the whole stack run with no external keys.
	AIMode      string
	BillingMode string

	JWTSecret string

	// Seed user created on startup so there is always a ready-to-use login.
	// In mock mode this defaults to a demo account; override via SEED_EMAIL/
	// SEED_PASSWORD, or set SEED_EMAIL=off to disable.
	SeedEmail    string
	SeedPassword string
	SeedPlan     string // free | pro | business

	// AIProvider selects the real backend when AIMode=real: "openai" or "gemini".
	AIProvider string

	OpenAIAPIKey string
	OpenAIModel  string // chat model, e.g. "gpt-4o-mini"
	OpenAIImage  string // image model, e.g. "gpt-image-1"

	GeminiAPIKey string
	GeminiModel  string // chat model, e.g. "gemini-2.0-flash"
	GeminiImage  string // image model, e.g. "gemini-2.0-flash-preview-image-generation"

	DatabaseURL string

	LemonSqueezyAPIKey        string
	LemonSqueezyStoreID       string
	LemonSqueezyWebhookSecret string
	LSVariantPro              string
	LSVariantBusiness         string

	// Shopee Affiliate Open API: "mock" or "real".
	ShopeeAffMode   string
	ShopeeAffAppID  string
	ShopeeAffSecret string
	ShopeeAffRegion string // th, vn, id, ... (selects open-api host)
}

// Load reads configuration from the environment, applying sensible defaults.
func Load() Config {
	c := Config{
		Port:          env("PORT", "8080"),
		PublicBaseURL: env("PUBLIC_BASE_URL", "http://localhost:8080"),
		OutputDir:     env("OUTPUT_DIR", "./output"),
		WorkerCount:   envInt("WORKER_COUNT", 3),

		AIMode:      env("AI_MODE", "mock"),
		BillingMode: env("BILLING_MODE", "mock"),

		JWTSecret: env("JWT_SECRET", "dev-insecure-secret-change-me"),

		AIProvider: env("AI_PROVIDER", "openai"),

		OpenAIAPIKey: os.Getenv("OPENAI_API_KEY"),
		OpenAIModel:  env("OPENAI_MODEL", "gpt-4o-mini"),
		OpenAIImage:  env("OPENAI_IMAGE_MODEL", "gpt-image-1"),

		GeminiAPIKey: os.Getenv("GEMINI_API_KEY"),
		GeminiModel:  env("GEMINI_MODEL", "gemini-2.5-flash"),
		GeminiImage:  env("GEMINI_IMAGE_MODEL", "gemini-2.5-flash-image"),

		DatabaseURL: env("DATABASE_URL", ""),

		LemonSqueezyAPIKey:        os.Getenv("LEMONSQUEEZY_API_KEY"),
		LemonSqueezyStoreID:       os.Getenv("LEMONSQUEEZY_STORE_ID"),
		LemonSqueezyWebhookSecret: os.Getenv("LEMONSQUEEZY_WEBHOOK_SECRET"),
		LSVariantPro:              os.Getenv("LS_VARIANT_PRO"),
		LSVariantBusiness:         os.Getenv("LS_VARIANT_BUSINESS"),

		ShopeeAffMode:   env("SHOPEE_AFF_MODE", "mock"),
		ShopeeAffAppID:  os.Getenv("SHOPEE_AFF_APP_ID"),
		ShopeeAffSecret: os.Getenv("SHOPEE_AFF_SECRET"),
		ShopeeAffRegion: env("SHOPEE_AFF_REGION", "th"),

		SeedEmail:    os.Getenv("SEED_EMAIL"),
		SeedPassword: os.Getenv("SEED_PASSWORD"),
		SeedPlan:     env("SEED_PLAN", "pro"),
	}

	// In mock mode, provide a ready-to-use demo login by default so you can sign
	// in immediately without registering. Set SEED_EMAIL=off to disable.
	if c.SeedEmail == "" && c.AIMode == "mock" {
		c.SeedEmail = "demo@ecom.dev"
		c.SeedPassword = "demo1234"
	}
	if c.SeedEmail == "off" {
		c.SeedEmail = ""
	}
	if c.SeedPassword == "" {
		c.SeedPassword = "demo1234"
	}
	return c
}

// Validate checks that required values are present for the selected modes.
func (c Config) Validate() error {
	if c.AIMode == "real" {
		switch c.AIProvider {
		case "gemini":
			if c.GeminiAPIKey == "" {
				return fmt.Errorf("AI_MODE=real with AI_PROVIDER=gemini requires GEMINI_API_KEY")
			}
		default: // openai
			if c.OpenAIAPIKey == "" {
				return fmt.Errorf("AI_MODE=real requires OPENAI_API_KEY")
			}
		}
	}
	if c.BillingMode == "real" {
		if c.LemonSqueezyAPIKey == "" || c.LemonSqueezyWebhookSecret == "" {
			return fmt.Errorf("BILLING_MODE=real requires LEMONSQUEEZY_API_KEY and LEMONSQUEEZY_WEBHOOK_SECRET")
		}
	}
	if c.ShopeeAffMode == "real" && (c.ShopeeAffAppID == "" || c.ShopeeAffSecret == "") {
		return fmt.Errorf("SHOPEE_AFF_MODE=real requires SHOPEE_AFF_APP_ID and SHOPEE_AFF_SECRET")
	}
	if c.JWTSecret == "" {
		return fmt.Errorf("JWT_SECRET is required")
	}
	return nil
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}
