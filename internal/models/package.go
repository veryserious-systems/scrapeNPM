package models

import (
	"time"

	"github.com/google/uuid"
)

type Package struct {
	ID              uuid.UUID `json:"id" db:"id"`
	Name            string    `json:"name" db:"name"`
	Version         string    `json:"version" db:"version"`
	Description     string    `json:"description" db:"description"`
	Author          string    `json:"author" db:"author"`
	Homepage        string    `json:"homepage" db:"homepage"`
	Repository      string    `json:"repository" db:"repository"`
	License         string    `json:"license" db:"license"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time `json:"updated_at" db:"updated_at"`
	Downloads       int64     `json:"downloads" db:"downloads"`
	PopularityScore float64   `json:"popularity_score" db:"popularity_score"`
	LastUpdated     time.Time `json:"last_updated" db:"last_updated"`
}

type PackageScript struct {
	ID         uuid.UUID `json:"id" db:"id"`
	PackageID  uuid.UUID `json:"package_id" db:"package_id"`
	ScriptType string    `json:"script_type" db:"script_type"`
	Content    string    `json:"content" db:"content"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time `json:"updated_at" db:"updated_at"`
}

type Job struct {
	ID               uuid.UUID              `json:"id" db:"id"`
	Type             string                 `json:"type" db:"job_type"`
	Status           string                 `json:"status" db:"status"`
	Priority         int                    `json:"priority" db:"priority"`
	Payload          map[string]interface{} `json:"payload" db:"payload"`
	CreatedAt        time.Time              `json:"created_at" db:"created_at"`
	StartedAt        *time.Time             `json:"started_at,omitempty" db:"started_at"`
	CompletedAt      *time.Time             `json:"completed_at,omitempty" db:"completed_at"`
	Attempts         int                    `json:"attempts" db:"attempts"`
	MaxAttempts      int                    `json:"max_attempts" db:"max_attempts"`
	ErrorMessage     string                 `json:"error_message,omitempty" db:"error_message"`
	WorkerID         string                 `json:"worker_id,omitempty" db:"worker_id"`
	NextAttemptAfter time.Time              `json:"next_attempt_after" db:"next_attempt_after"`
}
