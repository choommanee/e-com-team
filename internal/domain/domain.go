// Package domain holds the core types shared across the application.
package domain

import "time"

// User is a registered account.
type User struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
}

// PlanID identifies a subscription tier.
type PlanID string

const (
	PlanFree     PlanID = "free"
	PlanPro      PlanID = "pro"
	PlanBusiness PlanID = "business"
)

// Subscription is a user's current plan state.
type Subscription struct {
	UserID         string    `json:"user_id"`
	Plan           PlanID    `json:"plan"`
	Status         string    `json:"status"` // active, cancelled, expired
	LSSubscription string    `json:"-"`      // LemonSqueezy subscription id
	PeriodStart    time.Time `json:"period_start"`
	PeriodEnd      time.Time `json:"period_end"`
}

// JobStatus is the lifecycle state of a generation job.
type JobStatus string

const (
	JobPending JobStatus = "pending"
	JobRunning JobStatus = "running"
	JobDone    JobStatus = "done"
	JobFailed  JobStatus = "failed"
)

// Job is a single listing-generation request.
type Job struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	ProductName string    `json:"product_name"`
	Lang        string    `json:"lang"`
	Status      JobStatus `json:"status"`
	Result      *Listing  `json:"result,omitempty"`
	ImagePath   string    `json:"image_path,omitempty"`
	Error       string    `json:"error,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Listing is the finished output of the 6-agent pipeline.
type Listing struct {
	ProductName   string   `json:"product_name"`
	SellingPoints []string `json:"selling_points"` // from BENEFIT
	Headline      string   `json:"headline"`       // from PROMO
	Promotion     string   `json:"promotion"`      // from PROMO
	Layout        string   `json:"layout"`         // from DESIGN
	ColorTone     string   `json:"color_tone"`     // from DESIGN
	ImagePrompt   string   `json:"image_prompt"`   // from PROMPT (English)
	ImageURL      string   `json:"image_url"`      // from STUDIO
	QCStatus      string   `json:"qc_status"`      // from QUALITY CHECK: passed|needs_fix
	QCNotes       string   `json:"qc_notes"`
}

// Affiliate is a user enrolled in the affiliate/referral program.
type Affiliate struct {
	UserID    string    `json:"user_id"`
	Code      string    `json:"code"`   // unique referral code
	Status    string    `json:"status"` // approved | pending
	Bio       string    `json:"bio"`    // AI-generated profile
	Niche     string    `json:"niche"`
	Pitch     string    `json:"pitch"`
	Clicks    int       `json:"clicks"`
	Signups   int       `json:"signups"`
	EarningsTHB int     `json:"earnings_thb"`
	CreatedAt time.Time `json:"created_at"`
}

// Referral records that a user signed up through an affiliate's code.
type Referral struct {
	ReferredUserID string    `json:"referred_user_id"`
	AffiliateCode  string    `json:"affiliate_code"`
	Converted      bool      `json:"converted"` // true once the referred user upgrades
	CreatedAt      time.Time `json:"created_at"`
}

// EventType enumerates realtime event kinds pushed to dashboards.
type EventType string

const (
	EventProgress EventType = "progress" // an agent advanced
	EventJobDone  EventType = "job_done"
	EventJobFail  EventType = "job_failed"
	EventStats    EventType = "stats" // dashboard KPI refresh
)

// Event is a realtime message delivered over SSE.
type Event struct {
	Type    EventType `json:"type"`
	UserID  string    `json:"-"`
	JobID   string    `json:"job_id,omitempty"`
	Agent   string    `json:"agent,omitempty"`
	Task    string    `json:"task,omitempty"`
	Percent int       `json:"percent,omitempty"`
	Payload any       `json:"payload,omitempty"`
}
