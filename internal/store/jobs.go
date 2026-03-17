package store

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/cesargomez89/navidrums/internal/domain"
)

func (db *DB) CreateJob(job *domain.Job) error {
	query := `INSERT OR IGNORE INTO jobs (id, type, status, progress, source_id, parent_job_id, created_at, updated_at)
		VALUES (:id, :type, :status, :progress, :source_id, :parent_job_id, :created_at, :updated_at)`

	_, err := db.NamedExec(query, job)
	return err
}

func (db *DB) GetJob(id string) (*domain.Job, error) {
	query := `SELECT id, type, status, progress, source_id, parent_job_id, created_at, updated_at, error FROM jobs WHERE id = ?`

	job := &domain.Job{}
	err := db.Get(job, query, id)
	if err != nil {
		return nil, err
	}
	return job, nil
}

func (db *DB) UpdateJobStatus(id string, status domain.JobStatus, progress float64) error {
	query := `UPDATE jobs SET status = ?, progress = ?, updated_at = ? WHERE id = ?`
	_, err := db.Exec(query, status, progress, time.Now(), id)
	return err
}

func (db *DB) UpdateJobError(id string, errorMsg string) error {
	query := `UPDATE jobs SET status = ?, error = ?, updated_at = ? WHERE id = ?`
	_, err := db.Exec(query, domain.JobStatusFailed, errorMsg, time.Now(), id)
	return err
}

func (db *DB) ClearJobError(id string) error {
	query := `UPDATE jobs SET status = ?, progress = 0, error = NULL, updated_at = ? WHERE id = ?`
	_, err := db.Exec(query, domain.JobStatusQueued, time.Now(), id)
	return err
}

func (db *DB) ListJobs(limit int) ([]*domain.Job, error) {
	query := `SELECT id, type, status, progress, source_id, parent_job_id, created_at, updated_at, error FROM jobs ORDER BY created_at DESC LIMIT ?`

	var jobs []*domain.Job
	err := db.Select(&jobs, query, limit)
	return jobs, err
}

func (db *DB) ListActiveJobs(offset, limit int) ([]*domain.Job, error) {
	query := `SELECT id, type, status, progress, source_id, parent_job_id, created_at, updated_at FROM jobs WHERE status IN (?, ?) ORDER BY created_at ASC LIMIT ? OFFSET ?`

	var jobs []*domain.Job
	err := db.Select(&jobs, query, domain.JobStatusQueued, domain.JobStatusRunning, limit, offset)
	return jobs, err
}

func (db *DB) CountActiveJobs() (int, error) {
	query := `SELECT COUNT(*) FROM jobs WHERE status IN (?, ?)`
	var count int
	err := db.Get(&count, query, domain.JobStatusQueued, domain.JobStatusRunning)
	return count, err
}

func (db *DB) ListFinishedJobs(offset, limit int) ([]*domain.Job, error) {
	query := `SELECT id, type, status, progress, source_id, parent_job_id, created_at, updated_at, error FROM jobs WHERE status IN (?, ?, ?) ORDER BY updated_at DESC LIMIT ? OFFSET ?`

	var jobs []*domain.Job
	err := db.Select(&jobs, query, domain.JobStatusCompleted, domain.JobStatusFailed, domain.JobStatusCancelled, limit, offset)
	return jobs, err
}

func (db *DB) CountFinishedJobs() (int, error) {
	query := `SELECT COUNT(*) FROM jobs WHERE status IN (?, ?, ?)`
	var count int
	err := db.Get(&count, query, domain.JobStatusCompleted, domain.JobStatusFailed, domain.JobStatusCancelled)
	return count, err
}

func (db *DB) GetActiveJobBySourceID(sourceID string, jobType domain.JobType) (*domain.Job, error) {
	query := `SELECT id, type, status, progress, source_id, parent_job_id, created_at, updated_at 
		FROM jobs 
		WHERE source_id = ? AND type = ? AND status IN (?, ?)
		LIMIT 1`

	job := &domain.Job{}
	err := db.Get(job, query, sourceID, jobType, domain.JobStatusQueued, domain.JobStatusRunning)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return job, nil
}

func (db *DB) IsTrackActive(providerID string) (bool, error) {
	query := `SELECT COUNT(*) FROM jobs WHERE source_id = ? AND type = ? AND status IN (?, ?)`
	var count int
	err := db.Get(&count, query, providerID, domain.JobTypeTrack, domain.JobStatusQueued, domain.JobStatusRunning)
	return count > 0, err
}

func (db *DB) ResetStuckJobs() error {
	query := `UPDATE jobs SET status = ?, updated_at = ? WHERE status = ?`
	_, err := db.Exec(query, domain.JobStatusQueued, time.Now(), domain.JobStatusRunning)
	return err
}

func (db *DB) ClearFinishedJobs() error {
	query := `DELETE FROM jobs WHERE status IN (?, ?, ?)`
	_, err := db.Exec(query, domain.JobStatusCompleted, domain.JobStatusFailed, domain.JobStatusCancelled)
	return err
}

type JobStats struct {
	Total     int `db:"total"`
	Completed int `db:"completed"`
	Failed    int `db:"failed"`
	Cancelled int `db:"cancelled"`
}

func (db *DB) GetJobStats() (*JobStats, error) {
	query := `SELECT 
		COUNT(*) as total,
		SUM(CASE WHEN status = ? THEN 1 ELSE 0 END) as completed,
		SUM(CASE WHEN status = ? THEN 1 ELSE 0 END) as failed,
		SUM(CASE WHEN status = ? THEN 1 ELSE 0 END) as cancelled
	FROM jobs 
	WHERE status IN (?, ?, ?)`

	stats := &JobStats{}
	err := db.Get(stats, query,
		domain.JobStatusCompleted, domain.JobStatusFailed, domain.JobStatusCancelled,
		domain.JobStatusCompleted, domain.JobStatusFailed, domain.JobStatusCancelled)
	return stats, err
}

func (db *DB) CountJobsForParent(parentID string) (total int, pending int, err error) {
	row := db.QueryRow(`SELECT COUNT(*) FROM jobs WHERE parent_job_id = ?`, parentID)
	if err := row.Scan(&total); err != nil {
		return 0, 0, err
	}

	row = db.QueryRow(`
		SELECT COUNT(*) FROM jobs 
		WHERE parent_job_id = ? AND status IN (?, ?)`,
		parentID, domain.JobStatusQueued, domain.JobStatusRunning)
	if err := row.Scan(&pending); err != nil {
		return 0, 0, err
	}
	return total, pending, nil
}

func (db *DB) UpdateJobProgress(id string, progress float64) error {
	_, err := db.Exec(`UPDATE jobs SET progress = ?, updated_at = ? WHERE id = ?`,
		progress, time.Now(), id)
	return err
}

func (db *DB) CreateJobBatch(jobs []*domain.Job) error {
	tx, err := db.root.Beginx()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck // rollback is best-effort

	query := `INSERT OR IGNORE INTO jobs (id, type, status, progress, source_id, parent_job_id, created_at, updated_at)
		VALUES (:id, :type, :status, :progress, :source_id, :parent_job_id, :created_at, :updated_at)`

	for _, job := range jobs {
		if job.CreatedAt.IsZero() {
			job.CreatedAt = time.Now()
		}
		if job.UpdatedAt.IsZero() {
			job.UpdatedAt = time.Now()
		}

		if _, err := tx.NamedExec(query, job); err != nil {
			return fmt.Errorf("failed to create job %s: %w", job.ID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func (db *DB) CancelJobsByParentID(parentID string) error {
	_, err := db.Exec(`
		UPDATE jobs 
		SET status = ?, updated_at = ? 
		WHERE parent_job_id = ? AND status IN (?, ?)`,
		domain.JobStatusCancelled, time.Now(), parentID, domain.JobStatusQueued, domain.JobStatusRunning)
	return err
}
