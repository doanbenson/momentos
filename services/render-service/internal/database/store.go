package database

import (
	"fmt"
	"sync"
	"time"
)

// JobStatus is the persisted render job lifecycle state.
type JobStatus string

const (
	StatusQueued    JobStatus = "queued"
	StatusClaimed   JobStatus = "claimed"
	StatusRunning   JobStatus = "running"
	StatusSucceeded JobStatus = "succeeded"
	StatusFailed    JobStatus = "failed"
)

// Job row: inputs for encode + result metadata.
type Job struct {
	ID                 string     `json:"id"`
	MomentID           string     `json:"momentId"`
	CutID              string     `json:"cutId"`
	CutVersion         int        `json:"cutVersion"`
	SongPickID         string     `json:"songPickId,omitempty"`
	EditAudioURL       string     `json:"editAudioUrl,omitempty"`
	ReflectionQuestion string     `json:"reflectionQuestion,omitempty"` // Act 1 banner (ADR 0020); not on Cut clips
	CutJSON            []byte     `json:"-"`                              // raw Cut from payload when present
	Assets             []AssetRef `json:"assets"`
	Status             JobStatus  `json:"status"`
	Attempts           int        `json:"attempts"`
	MaxAttempts        int        `json:"maxAttempts"`
	OutputKey          string     `json:"outputKey,omitempty"`
	RenderURL          string     `json:"renderUrl,omitempty"`
	LastError          string     `json:"lastError,omitempty"`
	TerminalFail       bool       `json:"terminalFail,omitempty"`
	CreatedAt          time.Time  `json:"createdAt"`
	UpdatedAt          time.Time  `json:"updatedAt"`
}

// AssetRef is Asset metadata shipped on the job (type + download URL).
type AssetRef struct {
	ID  string `json:"id"`
	Type string `json:"type"` // photo | video | voice
	URL string `json:"url"`
}

// Store tracks render job status and output object keys (in-memory MVP).
type Store struct {
	mu   sync.RWMutex
	jobs map[string]*Job
}

func NewStore() *Store {
	return &Store{jobs: make(map[string]*Job)}
}

func (s *Store) Create(job *Job) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.jobs[job.ID]; ok {
		return fmt.Errorf("job %s already exists", job.ID)
	}
	now := time.Now().UTC()
	job.CreatedAt = now
	job.UpdatedAt = now
	if job.Status == "" {
		job.Status = StatusQueued
	}
	if job.MaxAttempts == 0 {
		job.MaxAttempts = 3 // initial + ≤2 retries
	}
	cp := *job
	s.jobs[job.ID] = &cp
	return nil
}

func (s *Store) Get(id string) (*Job, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	j, ok := s.jobs[id]
	if !ok {
		return nil, false
	}
	cp := *j
	return &cp, true
}

// ClaimNext picks the oldest queued job and marks it claimed.
func (s *Store) ClaimNext() (*Job, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var best *Job
	for _, j := range s.jobs {
		if j.Status != StatusQueued {
			continue
		}
		if best == nil || j.CreatedAt.Before(best.CreatedAt) {
			best = j
		}
	}
	if best == nil {
		return nil, false
	}
	best.Status = StatusClaimed
	best.UpdatedAt = time.Now().UTC()
	cp := *best
	return &cp, true
}

func (s *Store) Update(id string, fn func(*Job)) (*Job, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	j, ok := s.jobs[id]
	if !ok {
		return nil, fmt.Errorf("job %s not found", id)
	}
	fn(j)
	j.UpdatedAt = time.Now().UTC()
	cp := *j
	return &cp, nil
}

// Requeue sets a failed-but-retryable job back to queued.
func (s *Store) Requeue(id string, reason string) (*Job, error) {
	return s.Update(id, func(j *Job) {
		j.Status = StatusQueued
		j.LastError = reason
	})
}

func (s *Store) MarkRunning(id string) (*Job, error) {
	return s.Update(id, func(j *Job) {
		j.Status = StatusRunning
		j.Attempts++
	})
}

func (s *Store) MarkSucceeded(id, outputKey, renderURL string) (*Job, error) {
	return s.Update(id, func(j *Job) {
		j.Status = StatusSucceeded
		j.OutputKey = outputKey
		j.RenderURL = renderURL
		j.LastError = ""
	})
}

func (s *Store) MarkFailed(id, reason string, terminal bool) (*Job, error) {
	return s.Update(id, func(j *Job) {
		j.Status = StatusFailed
		j.LastError = reason
		j.TerminalFail = terminal
	})
}
