package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"homebooks/internal/models"
)

// CreateJob creates a new job and returns its ID
func (db *DB) CreateJob(jobType string, payload any) (int64, error) {
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return 0, fmt.Errorf("marshal payload: %w", err)
	}

	result, err := db.Exec(`
		INSERT INTO jobs (job_type, payload)
		VALUES (?, ?)
	`, jobType, string(payloadJSON))
	if err != nil {
		return 0, fmt.Errorf("insert job: %w", err)
	}
	return result.LastInsertId()
}

// ClaimNextJob atomically claims the next pending job for processing
func (db *DB) ClaimNextJob() (*models.Job, error) {
	tx, err := db.Begin()
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Find next pending job
	var job models.Job
	var startedAt, completedAt sql.NullTime
	err = tx.QueryRow(`
		SELECT id, job_type, payload, status, progress, result, attempts, max_attempts, created_at, started_at, completed_at
		FROM jobs
		WHERE status = 'pending'
		ORDER BY created_at ASC
		LIMIT 1
	`).Scan(&job.ID, &job.JobType, &job.Payload, &job.Status, &job.Progress, &job.Result,
		&job.Attempts, &job.MaxAttempts, &job.CreatedAt, &startedAt, &completedAt)

	if err == sql.ErrNoRows {
		return nil, nil // No pending jobs
	}
	if err != nil {
		return nil, fmt.Errorf("query job: %w", err)
	}

	if startedAt.Valid {
		job.StartedAt = &startedAt.Time
	}
	if completedAt.Valid {
		job.CompletedAt = &completedAt.Time
	}

	// Claim the job
	now := time.Now()
	_, err = tx.Exec(`
		UPDATE jobs
		SET status = 'running', started_at = ?, attempts = attempts + 1
		WHERE id = ?
	`, now, job.ID)
	if err != nil {
		return nil, fmt.Errorf("claim job: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}

	job.Status = "running"
	job.StartedAt = &now
	job.Attempts++

	return &job, nil
}

// GetJob returns a job by ID
func (db *DB) GetJob(id int64) (*models.Job, error) {
	var job models.Job
	var startedAt, completedAt sql.NullTime
	err := db.QueryRow(`
		SELECT id, job_type, payload, status, progress, result, attempts, max_attempts, created_at, started_at, completed_at
		FROM jobs
		WHERE id = ?
	`, id).Scan(&job.ID, &job.JobType, &job.Payload, &job.Status, &job.Progress, &job.Result,
		&job.Attempts, &job.MaxAttempts, &job.CreatedAt, &startedAt, &completedAt)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("job not found")
	}
	if err != nil {
		return nil, fmt.Errorf("query job: %w", err)
	}

	if startedAt.Valid {
		job.StartedAt = &startedAt.Time
	}
	if completedAt.Valid {
		job.CompletedAt = &completedAt.Time
	}

	return &job, nil
}

// UpdateJobProgress updates the progress percentage of a running job
func (db *DB) UpdateJobProgress(id int64, progress int) error {
	_, err := db.Exec(`
		UPDATE jobs SET progress = ? WHERE id = ?
	`, progress, id)
	if err != nil {
		return fmt.Errorf("update progress: %w", err)
	}
	return nil
}

// CompleteJob marks a job as completed with an optional result
func (db *DB) CompleteJob(id int64, result string) error {
	_, err := db.Exec(`
		UPDATE jobs
		SET status = 'completed', progress = 100, result = ?, completed_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, result, id)
	if err != nil {
		return fmt.Errorf("complete job: %w", err)
	}
	return nil
}

// FailJob marks a job as failed with an error message
func (db *DB) FailJob(id int64, errMsg string) error {
	_, err := db.Exec(`
		UPDATE jobs
		SET status = 'failed', result = ?, completed_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, errMsg, id)
	if err != nil {
		return fmt.Errorf("fail job: %w", err)
	}
	return nil
}

// RetryJob resets a job to pending status for retry
func (db *DB) RetryJob(id int64) error {
	_, err := db.Exec(`
		UPDATE jobs
		SET status = 'pending', started_at = NULL
		WHERE id = ?
	`, id)
	if err != nil {
		return fmt.Errorf("retry job: %w", err)
	}
	return nil
}
