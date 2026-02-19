package store

import (
	"database/sql"
	"time"

	"github.com/cesargomez89/navidrums/internal/domain"
)

func (db *DB) CreateJob(job *domain.Job) error {
	query := `INSERT OR IGNORE INTO jobs (id, type, status, progress, source_id, created_at, updated_at)
		VALUES (:id, :type, :status, :progress, :source_id, :created_at, :updated_at)`

	_, err := db.NamedExec(query, job)
	return err
}

func (db *DB) GetJob(id string) (*domain.Job, error) {
	query := `SELECT id, type, status, progress, source_id, created_at, updated_at, error FROM jobs WHERE id = ?`

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
	query := `SELECT id, type, status, progress, source_id, created_at, updated_at, error FROM jobs ORDER BY created_at DESC LIMIT ?`

	var jobs []*domain.Job
	err := db.Select(&jobs, query, limit)
	return jobs, err
}

func (db *DB) ListActiveJobs() ([]*domain.Job, error) {
	query := `SELECT id, type, status, progress, source_id, created_at, updated_at FROM jobs WHERE status IN ('queued', 'running') ORDER BY created_at ASC`

	var jobs []*domain.Job
	err := db.Select(&jobs, query)
	return jobs, err
}

func (db *DB) ListFinishedJobs(limit int) ([]*domain.Job, error) {
	query := `SELECT id, type, status, progress, source_id, created_at, updated_at, error FROM jobs WHERE status IN ('completed', 'failed', 'cancelled') ORDER BY updated_at DESC LIMIT ?`

	var jobs []*domain.Job
	err := db.Select(&jobs, query, limit)
	return jobs, err
}

func (db *DB) GetActiveJobBySourceID(sourceID string, jobType domain.JobType) (*domain.Job, error) {
	query := `SELECT id, type, status, progress, source_id, created_at, updated_at 
		FROM jobs 
		WHERE source_id = ? AND type = ? AND status IN ('queued', 'running')
		LIMIT 1`

	job := &domain.Job{}
	err := db.Get(job, query, sourceID, jobType)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return job, nil
}

func (db *DB) IsTrackActive(providerID string) (bool, error) {
	query := `SELECT COUNT(*) FROM jobs WHERE source_id = ? AND type = 'track' AND status IN ('queued', 'running')`
	var count int
	err := db.Get(&count, query, providerID)
	return count > 0, err
}

func (db *DB) ResetStuckJobs() error {
	query := `UPDATE jobs SET status = ?, updated_at = ? WHERE status = 'running'`
	_, err := db.Exec(query, domain.JobStatusQueued, time.Now())
	return err
}

func (db *DB) ClearFinishedJobs() error {
	query := `DELETE FROM jobs WHERE status IN ('completed', 'failed', 'cancelled')`
	_, err := db.Exec(query)
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
		SUM(CASE WHEN status = 'completed' THEN 1 ELSE 0 END) as completed,
		SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END) as failed,
		SUM(CASE WHEN status = 'cancelled' THEN 1 ELSE 0 END) as cancelled
	FROM jobs 
	WHERE status IN ('completed', 'failed', 'cancelled')`

	stats := &JobStats{}
	err := db.Get(stats, query)
	return stats, err
}
