// Package agents implements the 6-stage listing-generation pipeline.
//
// Each agent consumes the accumulated StageData, calls the LLM (or generates an
// image), reports progress via a Progress callback, and writes its result back
// into StageData. The Orchestrator wires them in order.
package agents

import (
	"context"

	"ecomteam/internal/domain"
	"ecomteam/internal/llm"
)

// Agent display names — also used as the SSE "agent" field and the mock tag.
const (
	NameBenefit = "BENEFIT"
	NamePromo   = "PROMO"
	NameDesign  = "DESIGN"
	NamePrompt  = "PROMPT"
	NameStudio  = "STUDIO"
	NameQC      = "QUALITY CHECK"
)

// Progress is called by agents to report incremental progress (0..100).
// task is a short human-readable label shown on the dashboard card.
type Progress func(agent string, percent int, task string)

// StageData is the mutable bag of results threaded through the pipeline.
type StageData struct {
	ProductName string
	Lang        string // "th" or "en"
	Listing     domain.Listing
	// ImagePNG holds the raw bytes produced by STUDIO; the orchestrator's caller
	// (the worker) is responsible for persisting it and setting Listing.ImageURL.
	ImagePNG []byte
}

// Agent is one stage of the pipeline.
type Agent interface {
	Name() string
	Run(ctx context.Context, d *StageData, client llm.Client, p Progress) error
}

// langInstruction returns a directive telling the model which language to write in.
func langInstruction(lang string) string {
	if lang == "en" {
		return "Write all customer-facing text in natural English."
	}
	return "Write all customer-facing text in natural Thai (ภาษาไทย)."
}
