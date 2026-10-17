package business

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/doanbenson/momentos/services/render-service/internal/database"
)

// Worker is an in-process render job consumer (GRILL Q6).
// Poll/claim against the local Store until moment-service broker HTTP is ready.
type Worker struct {
	Service     *Service
	Concurrency int
	PollEvery   time.Duration

	// PollBroker is optional. When set, called each tick to pull remote jobs
	// into the local Store. TODO: implement against moment-service job API.
	PollBroker func(ctx context.Context) error
}

// Start launches Concurrency poll loops. Blocks until ctx is cancelled, then waits for in-flight work.
func (w *Worker) Start(ctx context.Context) {
	n := w.Concurrency
	if n < 1 {
		n = 1
	}
	if n > 2 {
		n = 2 // MVP: 1–2 concurrent encodes
	}
	every := w.PollEvery
	if every <= 0 {
		every = 2 * time.Second
	}

	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			w.loop(ctx, workerID, every)
		}(i)
	}
	wg.Wait()
}

func (w *Worker) loop(ctx context.Context, workerID int, every time.Duration) {
	log.Printf("render-service: worker %d started (poll=%s)", workerID, every)
	ticker := time.NewTicker(every)
	defer ticker.Stop()

	for {
		w.tick(ctx, workerID)
		select {
		case <-ctx.Done():
			log.Printf("render-service: worker %d stopping", workerID)
			return
		case <-ticker.C:
		}
	}
}

func (w *Worker) tick(ctx context.Context, workerID int) {
	if w.PollBroker != nil {
		if err := w.PollBroker(ctx); err != nil {
			log.Printf("render-service: worker %d broker poll: %v", workerID, err)
		}
	}
	// TODO(broker): claim via moment-service HTTP when enqueue/claim API exists.
	// Until then, ClaimNext reads locally enqueued jobs (POST /v1/internal/render-jobs).

	job, ok := w.Service.Store.ClaimNext()
	if !ok {
		return
	}
	log.Printf("render-service: worker %d claimed job=%s moment=%s", workerID, job.ID, job.MomentID)
	err := w.Service.ProcessJob(job)
	w.Service.FinishAttempt(job.ID, err)
}

// EnqueueAndWake stores a job for the next poll tick.
func (w *Worker) EnqueueAndWake(job *database.Job) (*database.Job, error) {
	return w.Service.Enqueue(job)
}
