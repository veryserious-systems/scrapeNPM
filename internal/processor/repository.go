package processor

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"

	"scrapeNPM/internal/models"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) StorePackage(ctx context.Context, pkg models.Package) (uuid.UUID, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	var packageID uuid.UUID

	err = tx.QueryRow(ctx, `
        SELECT id FROM packages WHERE name = $1
    `, pkg.Name).Scan(&packageID)

	if err != nil {
		if err != pgx.ErrNoRows {
			return uuid.Nil, fmt.Errorf("failed to check if package exists: %w", err)
		}

		err = tx.QueryRow(ctx, `
            INSERT INTO packages (
                name, version, description, author, homepage, repository,
                license, created_at, updated_at, downloads, popularity_score, last_updated
            ) VALUES (
                $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, NOW()
            ) RETURNING id
        `, pkg.Name, pkg.Version, pkg.Description, pkg.Author, pkg.Homepage, pkg.Repository,
			pkg.License, pkg.CreatedAt, pkg.UpdatedAt, pkg.Downloads, pkg.PopularityScore).Scan(&packageID)

		if err != nil {
			return uuid.Nil, fmt.Errorf("failed to insert package: %w", err)
		}
	} else {
		_, err = tx.Exec(ctx, `
            UPDATE packages SET
                version = $2,
                description = $3,
                author = $4,
                homepage = $5,
                repository = $6,
                license = $7,
                created_at = $8,
                updated_at = $9,
                downloads = $10,
                popularity_score = $11,
                last_updated = NOW()
            WHERE id = $1
        `, packageID, pkg.Version, pkg.Description, pkg.Author, pkg.Homepage, pkg.Repository,
			pkg.License, pkg.CreatedAt, pkg.UpdatedAt, pkg.Downloads, pkg.PopularityScore)

		if err != nil {
			return uuid.Nil, fmt.Errorf("failed to update package: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return uuid.Nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return packageID, nil
}

func (r *Repository) StoreScript(ctx context.Context, script models.PackageScript) error {
	_, err := r.db.Exec(ctx, `
        INSERT INTO package_scripts (
            package_id, script_type, content, created_at, updated_at
        ) VALUES (
            $1, $2, $3, NOW(), NOW()
        ) ON CONFLICT (package_id, script_type) DO UPDATE SET
            content = $3,
            updated_at = NOW()
    `, script.PackageID, script.ScriptType, script.Content)

	if err != nil {
		return fmt.Errorf("failed to store script: %w", err)
	}

	return nil
}

func (r *Repository) ClaimJob(ctx context.Context, workerID string) (*models.Job, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	var job models.Job
	var payloadJSON []byte
	var errorMessage sql.NullString
	var startedAt sql.NullTime

	err = tx.QueryRow(ctx, `
        UPDATE job_queue 
        SET 
            status = 'processing', 
            started_at = NOW(), 
            worker_id = $1,
            attempts = attempts + 1
        WHERE id = (
            SELECT id 
            FROM job_queue 
            WHERE 
                status = 'pending' 
                AND next_attempt_after <= NOW() 
            ORDER BY priority, created_at 
            LIMIT 1
            FOR UPDATE SKIP LOCKED
        )
        RETURNING id, job_type, status, priority, payload, created_at, 
                  started_at, attempts, max_attempts, error_message, next_attempt_after
    `, workerID).Scan(
		&job.ID, &job.Type, &job.Status, &job.Priority, &payloadJSON, &job.CreatedAt,
		&startedAt, &job.Attempts, &job.MaxAttempts, &errorMessage, &job.NextAttemptAfter,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to claim job: %w", err)
	}

	if startedAt.Valid {
		job.StartedAt = &startedAt.Time
	}

	if errorMessage.Valid {
		job.ErrorMessage = errorMessage.String
	} else {
		job.ErrorMessage = ""
	}

	if err := json.Unmarshal(payloadJSON, &job.Payload); err != nil {
		return nil, fmt.Errorf("failed to unmarshal job payload: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return &job, nil
}

func (r *Repository) CompleteJob(ctx context.Context, jobID uuid.UUID) error {
	now := time.Now()
	_, err := r.db.Exec(ctx, `
        UPDATE job_queue 
        SET 
            status = 'completed', 
            completed_at = $2
        WHERE id = $1
    `, jobID, now)

	if err != nil {
		return fmt.Errorf("failed to complete job: %w", err)
	}

	return nil
}

func (r *Repository) FailJob(ctx context.Context, jobID uuid.UUID, errorMsg string) error {
	_, err := r.db.Exec(ctx, `
        UPDATE job_queue 
        SET 
            status = CASE WHEN attempts >= max_attempts THEN 'failed' ELSE 'pending' END,
            error_message = $2,
            next_attempt_after = CASE WHEN attempts >= max_attempts 
                                THEN NULL 
                                ELSE NOW() + (POWER(2, attempts) * INTERVAL '1 minute') 
                                END
        WHERE id = $1
    `, jobID, errorMsg)

	if err != nil {
		return fmt.Errorf("failed to update failed job: %w", err)
	}

	return nil
}
