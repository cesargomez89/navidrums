package app

import (
	"fmt"
	"time"

	"github.com/cesargomez89/navidrums/internal/constants"
	"github.com/cesargomez89/navidrums/internal/domain"
	"github.com/cesargomez89/navidrums/internal/store"
	"github.com/google/uuid"
)

type JobService struct {
	Repo *store.DB
}

func NewJobService(repo *store.DB) *JobService {
	return &JobService{Repo: repo}
}

func (s *JobService) EnqueueJob(sourceID string, jobType domain.JobType) (*domain.Job, error) {
	// Check if already exists and is active to avoid duplicates
	existing, err := s.Repo.GetActiveJobBySourceID(sourceID, jobType)
	if err != nil {
		return nil, fmt.Errorf("failed to check for existing job: %w", err)
	}
	if existing != nil {
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
		Title:     fmt.Sprintf("Pending %s %s", jobType, sourceID), // Will be updated by worker
	}

	if err := s.Repo.CreateJob(job); err != nil {
		return nil, err
	}
	return job, nil
}

func (s *JobService) ListJobs() ([]*domain.Job, error) {
	return s.Repo.ListJobs(constants.MaxSearchResults)
}

func (s *JobService) GetJob(id string) (*domain.Job, error) {
	return s.Repo.GetJob(id)
}

func (s *JobService) ListActiveJobs() ([]*domain.Job, error) {
	return s.Repo.ListActiveJobs()
}

func (s *JobService) CancelJob(id string) error {
	return s.Repo.UpdateJobStatus(id, domain.JobStatusCancelled, 0)
}

func (s *JobService) RetryJob(id string) error {
	job, err := s.Repo.GetJob(id)
	if err != nil {
		return fmt.Errorf("failed to get job: %w", err)
	}
	if job == nil {
		return fmt.Errorf("job not found")
	}
	return s.Repo.UpdateJobStatus(id, domain.JobStatusQueued, 0)
}

func (s *JobService) ListFinishedJobs(limit int) ([]*domain.Job, error) {
	return s.Repo.ListFinishedJobs(limit)
}

func (s *JobService) GetJobStats() (*store.JobStats, error) {
	return s.Repo.GetJobStats()
}

func (s *JobService) ClearFinishedJobs() error {
	return s.Repo.ClearFinishedJobs()
}
