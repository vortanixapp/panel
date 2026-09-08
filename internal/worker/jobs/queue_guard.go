package jobs

import (
	"context"
	"log"
	"time"
)

// Сторожа очереди.
//
// Схема давно знает про attempts и updated_at, но воркер ими не пользовался:
// попытки не считались, предела не было, а задача, взятая в работу перед
// падением процесса, оставалась в состоянии running навсегда. На практике это
// давало две поломки. Ядовитая задача бралась первой после каждого перезапуска
// и валила обработчик по кругу, а очередь конкретного арендатора вставала
// молча — при этом /health воркера продолжал отвечать «ok».
const (
	// maxJobAttempts — после скольких неудач задача уходит в отказ и перестаёт
	// мешать остальным. Пять: сетевые сбои и недоступность ноды переживают
	// один-два повтора, а систематическая ошибка после пяти уже очевидна.
	maxJobAttempts = 5

	// staleRunningAfter — сколько задача может числиться в работе без единого
	// изменения, прежде чем считать её брошенной. Самые долгие задачи здесь —
	// копирование и установка ноды; часа им хватает с запасом.
	staleRunningAfter = time.Hour

	staleSweepInterval = 5 * time.Minute
)

// StaleJobsLoop возвращает в очередь задачи, брошенные упавшим воркером, и
// уводит в отказ те, что исчерпали попытки.
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
	// Исчерпавшие попытки уводим в отказ первыми: иначе они вернутся в
	// очередь и снова займут обработчик.
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
