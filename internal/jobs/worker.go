package jobs

import (
	"context"
	"log/slog"
	"time"

	"homebooks/internal/database"
	"homebooks/internal/models"
)

// JobHandler is a function that processes a job
type JobHandler func(ctx context.Context, job *models.Job, db *database.DB) error

// Worker processes background jobs from the queue
type Worker struct {
	db       *database.DB
	handlers map[string]JobHandler
	stop     chan struct{}
	done     chan struct{}
	logger   *slog.Logger
	pollInterval time.Duration
}

// NewWorker creates a new job worker
func NewWorker(db *database.DB, logger *slog.Logger) *Worker {
	return &Worker{
		db:           db,
		handlers:     make(map[string]JobHandler),
		stop:         make(chan struct{}),
		done:         make(chan struct{}),
		logger:       logger,
		pollInterval: 2 * time.Second,
	}
}

// Register adds a handler for a job type
func (w *Worker) Register(jobType string, handler JobHandler) {
	w.handlers[jobType] = handler
}

// Start begins processing jobs in a background goroutine
func (w *Worker) Start() {
	go func() {
		defer close(w.done)
		w.logger.Info("job_worker_started")

		for {
			select {
			case <-w.stop:
				w.logger.Info("job_worker_stopping")
				return
			default:
				job, err := w.db.ClaimNextJob()
				if err != nil {
					w.logger.Error("job_claim_error", "error", err.Error())
					time.Sleep(w.pollInterval)
					continue
				}

				if job == nil {
					// No pending jobs, wait before polling again
					time.Sleep(w.pollInterval)
					continue
				}

				w.processJob(job)
			}
		}
	}()
}

// Stop signals the worker to stop and waits for it to finish
func (w *Worker) Stop() {
	close(w.stop)
	<-w.done
	w.logger.Info("job_worker_stopped")
}

func (w *Worker) processJob(job *models.Job) {
	l := w.logger.With("job_id", job.ID, "job_type", job.JobType, "attempt", job.Attempts)
	l.Info("job_processing_started")

	handler, ok := w.handlers[job.JobType]
	if !ok {
		l.Error("job_unknown_type")
		w.db.FailJob(job.ID, "unknown job type: "+job.JobType)
		return
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Run the handler
	err := handler(ctx, job, w.db)

	if err != nil {
		l.Error("job_processing_failed", "error", err.Error())

		if job.Attempts >= job.MaxAttempts {
			l.Warn("job_max_attempts_reached")
			w.db.FailJob(job.ID, err.Error())
		} else {
			l.Info("job_retrying")
			w.db.RetryJob(job.ID)
		}
		return
	}

	l.Info("job_processing_completed")
}
