package repository

import (
	"database/sql"
	"time"

	"github.com/cesargomez89/navidrums/internal/models"
)

func (db *DB) CreateJob(job *models.Job) error {
	query := `INSERT INTO jobs (id, type, status, title, artist, progress, source_id, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := db.Exec(query, job.ID, job.Type, job.Status, job.Title, job.Artist, job.Progress, job.SourceID, job.CreatedAt, job.UpdatedAt)
	return err
}

func (db *DB) GetJob(id string) (*models.Job, error) {
	query := `SELECT id, type, status, title, artist, progress, source_id, created_at, updated_at, error FROM jobs WHERE id = ?`
	row := db.QueryRow(query, id)

	job := &models.Job{}
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

func (db *DB) UpdateJobStatus(id string, status models.JobStatus, progress float64) error {
	query := `UPDATE jobs SET status = ?, progress = ?, updated_at = ? WHERE id = ?`
	_, err := db.Exec(query, status, progress, time.Now(), id)
	return err
}

func (db *DB) UpdateJobError(id string, errorMsg string) error {
	query := `UPDATE jobs SET status = ?, error = ?, updated_at = ? WHERE id = ?`
	_, err := db.Exec(query, models.JobStatusFailed, errorMsg, time.Now(), id)
	return err
}

func (db *DB) ListJobs(limit int) ([]*models.Job, error) {
	query := `SELECT id, type, status, title, artist, progress, source_id, created_at, updated_at, error FROM jobs ORDER BY created_at DESC LIMIT ?`
	rows, err := db.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []*models.Job
	for rows.Next() {
		job := &models.Job{}
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

func (db *DB) ListActiveJobs() ([]*models.Job, error) {
	query := `SELECT id, type, status, title, artist, progress, source_id, created_at, updated_at FROM jobs WHERE status IN ('queued', 'resolving_tracks', 'downloading') ORDER BY created_at ASC`
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []*models.Job
	for rows.Next() {
		job := &models.Job{}
		err := rows.Scan(&job.ID, &job.Type, &job.Status, &job.Title, &job.Artist, &job.Progress, &job.SourceID, &job.CreatedAt, &job.UpdatedAt)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, job)
	}
	return jobs, nil
}

func (db *DB) ListFinishedJobs(limit int) ([]*models.Job, error) {
	query := `SELECT id, type, status, title, artist, progress, source_id, created_at, updated_at, error FROM jobs WHERE status IN ('completed', 'failed', 'cancelled') ORDER BY updated_at DESC LIMIT ?`
	rows, err := db.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []*models.Job
	for rows.Next() {
		job := &models.Job{}
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

// Job Items
func (db *DB) CreateJobItem(item *models.JobItem) error {
	query := `INSERT INTO job_items (job_id, track_id, status, progress, title) VALUES (?, ?, ?, ?, ?)`
	_, err := db.Exec(query, item.JobID, item.TrackID, item.Status, item.Progress, item.Title)
	return err
}

func (db *DB) GetJobItems(jobID string) ([]*models.JobItem, error) {
	query := `SELECT id, job_id, track_id, status, progress, title, file_path FROM job_items WHERE job_id = ?`
	rows, err := db.Query(query, jobID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*models.JobItem
	for rows.Next() {
		item := &models.JobItem{}
		var filePath sql.NullString
		err := rows.Scan(&item.ID, &item.JobID, &item.TrackID, &item.Status, &item.Progress, &item.Title, &filePath)
		if err != nil {
			return nil, err
		}
		if filePath.Valid {
			item.FilePath = filePath.String
		}
		items = append(items, item)
	}
	return items, nil
}

func (db *DB) UpdateJobItemStatus(id int64, status models.JobItemStatus, progress float64) error {
	query := `UPDATE job_items SET status = ?, progress = ? WHERE id = ?`
	_, err := db.Exec(query, status, progress, id)
	return err
}

func (db *DB) UpdateJobItemFilePath(id int64, path string) error {
	query := `UPDATE job_items SET file_path = ? WHERE id = ?`
	_, err := db.Exec(query, path, id)
	return err
}

func (db *DB) GetActiveJobBySourceID(sourceID string, jobType models.JobType) (*models.Job, error) {
	query := `SELECT id, type, status, title, artist, progress, source_id, created_at, updated_at 
		FROM jobs 
		WHERE source_id = ? AND type = ? AND status IN ('queued', 'resolving_tracks', 'downloading')
		LIMIT 1`
	row := db.QueryRow(query, sourceID, jobType)

	job := &models.Job{}
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
	_, err := db.Exec(query, models.JobStatusQueued, time.Now())
	return err
}
