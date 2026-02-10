package services

import (
	"fmt"
	"time"

	"github.com/cesargomez89/navidrums/internal/models"
	"github.com/cesargomez89/navidrums/internal/repository"
	"github.com/google/uuid"
)

type JobService struct {
	Repo *repository.DB
}

func NewJobService(repo *repository.DB) *JobService {
	return &JobService{Repo: repo}
}

func (s *JobService) EnqueueJob(sourceID string, jobType models.JobType) (*models.Job, error) {
	// Check if already exists and is active to avoid duplicates
	existing, err := s.Repo.GetActiveJobBySourceID(sourceID, jobType)
	if err != nil {
		return nil, fmt.Errorf("failed to check for existing job: %w", err)
	}
	if existing != nil {
		return existing, nil
	}

	id := uuid.New().String()
	job := &models.Job{
		ID:        id,
		Type:      jobType,
		Status:    models.JobStatusQueued,
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

func (s *JobService) ListJobs() ([]*models.Job, error) {
	return s.Repo.ListJobs(50)
}

func (s *JobService) GetJob(id string) (*models.Job, error) {
	return s.Repo.GetJob(id)
}

func (s *JobService) ListActiveJobs() ([]*models.Job, error) {
	return s.Repo.ListActiveJobs()
}
