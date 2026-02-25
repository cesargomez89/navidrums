package app

import (
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/cesargomez89/navidrums/internal/constants"
	"github.com/cesargomez89/navidrums/internal/domain"
	"github.com/cesargomez89/navidrums/internal/logger"
	"github.com/cesargomez89/navidrums/internal/store"
)

type JobService struct {
	Repo   *store.DB
	Logger *logger.Logger
}

func NewJobService(repo *store.DB, log *logger.Logger) *JobService {
	return &JobService{Repo: repo, Logger: log}
}

func (s *JobService) EnqueueJob(sourceID string, jobType domain.JobType) (*domain.Job, error) {
	existing, err := s.Repo.GetActiveJobBySourceID(sourceID, jobType)
	if err != nil {
		return nil, fmt.Errorf("failed to check for existing job: %w", err)
	}
	if existing != nil {
		s.Logger.Info("Job already exists", "job_id", existing.ID, "source_id", sourceID, "type", jobType)
		return existing, nil
	}

	id := uuid.New().String()
	job := &domain.Job{
		ID:        id,
		Type:      jobType,
		Status:    domain.JobStatusQueued,
		SourceID:  sourceID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.Repo.CreateJob(job); err != nil {
		return nil, err
	}
	s.Logger.Info("Job enqueued", "job_id", job.ID, "source_id", sourceID, "type", jobType)
	return job, nil
}

func (s *JobService) ListJobs() ([]*domain.Job, error) {
	return s.Repo.ListJobs(constants.MaxSearchResults)
}

func (s *JobService) GetJob(id string) (*domain.Job, error) {
	return s.Repo.GetJob(id)
}

func (s *JobService) ListActiveJobs(page, pageSize int) ([]*domain.Job, int, error) {
	offset := (page - 1) * pageSize
	total, err := s.Repo.CountActiveJobs()
	if err != nil {
		return nil, 0, err
	}
	jobs, err := s.Repo.ListActiveJobs(offset, pageSize)
	if err != nil {
		return nil, 0, err
	}
	return jobs, total, nil
}

func (s *JobService) CancelJob(id string) error {
	err := s.Repo.UpdateJobStatus(id, domain.JobStatusCancelled, 0)
	if err != nil {
		return err
	}
	s.Logger.Info("Job cancelled", "job_id", id)
	return nil
}

func (s *JobService) RetryJob(id string) error {
	job, err := s.Repo.GetJob(id)
	if err != nil {
		return fmt.Errorf("failed to get job: %w", err)
	}
	if job == nil {
		return fmt.Errorf("job not found")
	}
	err = s.Repo.ClearJobError(id)
	if err != nil {
		return err
	}
	s.Logger.Info("Job retried", "job_id", id, "type", job.Type, "source_id", job.SourceID)
	return nil
}

func (s *JobService) ListFinishedJobs(page, pageSize int) ([]*domain.Job, int, error) {
	offset := (page - 1) * pageSize
	total, err := s.Repo.CountFinishedJobs()
	if err != nil {
		return nil, 0, err
	}
	jobs, err := s.Repo.ListFinishedJobs(offset, pageSize)
	if err != nil {
		return nil, 0, err
	}
	return jobs, total, nil
}

func (s *JobService) GetJobStats() (*store.JobStats, error) {
	return s.Repo.GetJobStats()
}

func (s *JobService) ClearFinishedJobs() error {
	return s.Repo.ClearFinishedJobs()
}
