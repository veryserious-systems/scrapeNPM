package discovery

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"

	"scrapeNPM/internal/models"
)

type JobQueueRepository struct {
	db *pgxpool.Pool
}

func NewJobQueueRepository(db *pgxpool.Pool) *JobQueueRepository {
	return &JobQueueRepository{db: db}
}

func (r *JobQueueRepository) EnqueueJob(ctx context.Context, job models.Job) (uuid.UUID, error) {
	payloadBytes, err := json.Marshal(job.Payload)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	var jobID uuid.UUID
	err = r.db.QueryRow(ctx, `
        INSERT INTO job_queue (
            job_type, status, priority, payload, max_attempts
        ) VALUES (
            $1, 'pending', $2, $3, $4
        ) RETURNING id
    `, job.Type, job.Priority, payloadBytes, job.MaxAttempts).Scan(&jobID)

	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to insert job: %w", err)
	}

	return jobID, nil
}

func (r *JobQueueRepository) GetScrapeProgress(ctx context.Context, id string) (string, int64, error) {
	var lastSequence string
	var totalProcessed int64

	err := r.db.QueryRow(ctx, `
        SELECT last_sequence, total_processed 
        FROM scrape_progress 
        WHERE id = $1
    `, id).Scan(&lastSequence, &totalProcessed)

	if err != nil {
		if err == pgx.ErrNoRows {
			return "0", 0, nil
		}
		return "", 0, fmt.Errorf("failed to get scrape progress: %w", err)
	}

	return lastSequence, totalProcessed, nil
}

func (r *JobQueueRepository) UpdateScrapeProgress(ctx context.Context, id string, lastSequence string, totalProcessed int64) error {
	_, err := r.db.Exec(ctx, `
        INSERT INTO scrape_progress (id, last_sequence, total_processed, last_updated)
        VALUES ($1, $2, $3, NOW())
        ON CONFLICT (id) DO UPDATE SET
            last_sequence = $2,
            total_processed = $3,
            last_updated = NOW()
    `, id, lastSequence, totalProcessed)

	if err != nil {
		return fmt.Errorf("failed to update scrape progress: %w", err)
	}

	return nil
}

func (r *JobQueueRepository) GetQueueStats(ctx context.Context) (map[string]int, error) {
	var pending, processing, completed, failed, total int

	err := r.db.QueryRow(ctx, `
        SELECT
            COUNT(*) FILTER (WHERE status = 'pending') AS pending,
            COUNT(*) FILTER (WHERE status = 'processing') AS processing,
            COUNT(*) FILTER (WHERE status = 'completed') AS completed,
            COUNT(*) FILTER (WHERE status = 'failed') AS failed,
            COUNT(*) AS total
        FROM job_queue
    `).Scan(&pending, &processing, &completed, &failed, &total)

	if err != nil {
		return nil, fmt.Errorf("failed to get queue stats: %w", err)
	}

	return map[string]int{
		"pending":    pending,
		"processing": processing,
		"completed":  completed,
		"failed":     failed,
		"total":      total,
	}, nil
}
