package jobs

import (
	"context"
	"log"
	"time"
)

const (
	maxJobAttempts = 5

	staleRunningAfter = time.Hour

	staleSweepInterval = 5 * time.Minute
)

func (r *Runner) StaleJobsLoop(ctx context.Context) {
	ticker := time.NewTicker(staleSweepInterval)
	defer ticker.Stop()

	r.reclaimStaleJobs(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.reclaimStaleJobs(ctx)
		}
	}
}

func (r *Runner) reclaimStaleJobs(ctx context.Context) {
	tag, err := r.db.Exec(ctx, `
		UPDATE core.jobs
		SET status = 'failed',
		    result = COALESCE(result, '{}'::jsonb) || jsonb_build_object(
		        'error', 'задача брошена и исчерпала попытки'
		    )
		WHERE status = 'running'
		  AND updated_at < now() - $1::interval
		  AND attempts >= $2
	`, staleRunningAfter.String(), maxJobAttempts)
	if err != nil {
		log.Printf("уборка очереди: отказ по исчерпанным попыткам: %v", err)
		return
	}
	if n := tag.RowsAffected(); n > 0 {
		log.Printf("уборка очереди: %d задач(и) уведены в отказ после %d попыток", n, maxJobAttempts)
	}

	tag, err = r.db.Exec(ctx, `
		UPDATE core.jobs
		SET status = 'pending'
		WHERE status = 'running'
		  AND updated_at < now() - $1::interval
	`, staleRunningAfter.String())
	if err != nil {
		log.Printf("уборка очереди: возврат брошенных задач: %v", err)
		return
	}
	if n := tag.RowsAffected(); n > 0 {
		log.Printf("уборка очереди: %d задач(и) возвращены в очередь после простоя", n)
	}
}
