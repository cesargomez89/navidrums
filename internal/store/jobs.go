package store

import (
	"database/sql"
	"time"

	"github.com/cesargomez89/navidrums/internal/domain"
)

func (db *DB) CreateJob(job *domain.Job) error {
	query := `INSERT OR IGNORE INTO jobs (id, type, status, title, artist, progress, source_id, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := db.Exec(query, job.ID, job.Type, job.Status, job.Title, job.Artist, job.Progress, job.SourceID, job.CreatedAt, job.UpdatedAt)
	return err
}

func (db *DB) GetJob(id string) (*domain.Job, error) {
	query := `SELECT id, type, status, title, artist, progress, source_id, created_at, updated_at, error FROM jobs WHERE id = ?`
	row := db.QueryRow(query, id)

	job := &domain.Job{}
	var errMsg sql.NullString
	err := row.Scan(&job.ID, &job.Type, &job.Status, &job.Title, &job.Artist, &job.Progress, &job.SourceID, &job.CreatedAt, &job.UpdatedAt, &errMsg)
	if err != nil {
		return nil, err
	}
	if errMsg.Valid {
		job.Error = errMsg.String
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

func (db *DB) UpdateJobMetadata(id string, title string, artist string) error {
	query := `UPDATE jobs SET title = ?, artist = ?, updated_at = ? WHERE id = ?`
	_, err := db.Exec(query, title, artist, time.Now(), id)
	return err
}

func (db *DB) ListJobs(limit int) ([]*domain.Job, error) {
	query := `SELECT id, type, status, title, artist, progress, source_id, created_at, updated_at, error FROM jobs ORDER BY created_at DESC LIMIT ?`
	rows, err := db.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []*domain.Job
	for rows.Next() {
		job := &domain.Job{}
		var errMsg sql.NullString
		err := rows.Scan(&job.ID, &job.Type, &job.Status, &job.Title, &job.Artist, &job.Progress, &job.SourceID, &job.CreatedAt, &job.UpdatedAt, &errMsg)
		if err != nil {
			return nil, err
		}
		if errMsg.Valid {
			job.Error = errMsg.String
		}
		jobs = append(jobs, job)
	}
	return jobs, nil
}

func (db *DB) ListActiveJobs() ([]*domain.Job, error) {
	query := `SELECT id, type, status, title, artist, progress, source_id, created_at, updated_at FROM jobs WHERE status IN ('queued', 'resolving_tracks', 'downloading') ORDER BY created_at ASC`
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []*domain.Job
	for rows.Next() {
		job := &domain.Job{}
		err := rows.Scan(&job.ID, &job.Type, &job.Status, &job.Title, &job.Artist, &job.Progress, &job.SourceID, &job.CreatedAt, &job.UpdatedAt)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, job)
	}
	return jobs, nil
}

func (db *DB) ListFinishedJobs(limit int) ([]*domain.Job, error) {
	query := `SELECT id, type, status, title, artist, progress, source_id, created_at, updated_at, error FROM jobs WHERE status IN ('completed', 'failed', 'cancelled') ORDER BY updated_at DESC LIMIT ?`
	rows, err := db.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []*domain.Job
	for rows.Next() {
		job := &domain.Job{}
		var errMsg sql.NullString
		err := rows.Scan(&job.ID, &job.Type, &job.Status, &job.Title, &job.Artist, &job.Progress, &job.SourceID, &job.CreatedAt, &job.UpdatedAt, &errMsg)
		if err != nil {
			return nil, err
		}
		if errMsg.Valid {
			job.Error = errMsg.String
		}
		jobs = append(jobs, job)
	}
	return jobs, nil
}

func (db *DB) GetActiveJobBySourceID(sourceID string, jobType domain.JobType) (*domain.Job, error) {
	query := `SELECT id, type, status, title, artist, progress, source_id, created_at, updated_at 
		FROM jobs 
		WHERE source_id = ? AND type = ? AND status IN ('queued', 'resolving_tracks', 'downloading')
		LIMIT 1`
	row := db.QueryRow(query, sourceID, jobType)

	job := &domain.Job{}
	err := row.Scan(&job.ID, &job.Type, &job.Status, &job.Title, &job.Artist, &job.Progress, &job.SourceID, &job.CreatedAt, &job.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return job, err
}

func (db *DB) IsTrackActive(trackID string) (bool, error) {
	query := `SELECT COUNT(*) FROM jobs WHERE source_id = ? AND type = 'track' AND status IN ('queued', 'resolving_tracks', 'downloading')`
	var count int
	err := db.QueryRow(query, trackID).Scan(&count)
	return count > 0, err
}

func (db *DB) ResetStuckJobs() error {
	query := `UPDATE jobs SET status = ?, updated_at = ? WHERE status IN ('resolving_tracks', 'downloading')`
	_, err := db.Exec(query, domain.JobStatusQueued, time.Now())
	return err
}

func (db *DB) ClearFinishedJobs() error {
	query := `DELETE FROM jobs WHERE status IN ('completed', 'failed', 'cancelled')`
	_, err := db.Exec(query)
	return err
}

type JobStats struct {
	Total     int
	Completed int
	Failed    int
	Cancelled int
}

func (db *DB) GetJobStats() (*JobStats, error) {
	stats := &JobStats{}

	query := `SELECT 
		COUNT(*) as total,
		SUM(CASE WHEN status = 'completed' THEN 1 ELSE 0 END) as completed,
		SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END) as failed,
		SUM(CASE WHEN status = 'cancelled' THEN 1 ELSE 0 END) as cancelled
	FROM jobs 
	WHERE status IN ('completed', 'failed', 'cancelled')`

	var total sql.NullInt64
	var completed, failed, cancelled sql.NullInt64

	err := db.QueryRow(query).Scan(&total, &completed, &failed, &cancelled)
	if err != nil {
		return nil, err
	}

	if total.Valid {
		stats.Total = int(total.Int64)
	}
	if completed.Valid {
		stats.Completed = int(completed.Int64)
	}
	if failed.Valid {
		stats.Failed = int(failed.Int64)
	}
	if cancelled.Valid {
		stats.Cancelled = int(cancelled.Int64)
	}

	return stats, nil
}
