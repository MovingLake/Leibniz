package lib

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"movinglake.com/leibniz/database"
)

type TaskRunner interface {
	Run(ctx context.Context, db *sql.DB, task *database.Task) error
}

type Worker struct {
	db        *sql.DB
	cfg       *LaunchConfig
	runnerMap map[string]TaskRunner
}

func NewWorker(db *sql.DB, rm map[string]TaskRunner, cfg *LaunchConfig) *Worker {
	return &Worker{
		db:        db,
		runnerMap: rm,
		cfg:       cfg,
	}
}

func (w *Worker) setupTask(runID int, task *database.Task) error {
	r, err := w.db.Exec("UPDATE leibniz_tasks SET owner_worker_id = $1, state = $2, last_updated = $3, num_retries = $4 WHERE id = $5",
		runID, database.TASK_RUNNING, time.Now(), task.NumRetries+1, task.ID)
	if err != nil {
		return err
	}
	n, err := r.RowsAffected()
	if err != nil || n == 0 {
		return err
	}
	return nil
}

func (w *Worker) resultTaskResult(log *LeibnizLogger, runID int, task *database.Task, state database.TaskState, result string) error {
	if task.NumRetries >= task.MaxRetries {
		log.Warn("worker %d task id %d name %s has exhausted retries", runID, task.ID, task.Name)
		state = database.TASK_FAILED
	}
	r, err := w.db.Exec("UPDATE leibniz_tasks SET owner_worker_id = NULL, state = $1, last_updated = $2, task_result = $3 WHERE id = $4",
		state, time.Now(), result, task.ID)
	if err != nil {
		log.Error("worker %d failed to update task id %d name %s error %v", runID, task.ID, task.Name, err)
		return err
	}
	n, err := r.RowsAffected()
	if err != nil || n == 0 {
		return err
	}
	return nil
}

func (w *Worker) Run(ctx context.Context, id int, tasks <-chan *database.Task) {
	log := NewLogger(fmt.Sprintf("worker-%d", id), w.cfg.LogLevel)
	log.Debug("worker %d started", id)
	for task := range tasks {
		log.Debug("worker %d received task id %d name \"%s\"", id, task.ID, task.Name)
		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := w.setupTask(id, task); err != nil {
				log.Error("worker %d failed to lock task id %d name %s error %v ", id, task.ID, task.Name, err)
				return
			}
			log.Debug("worker %d started task id %d name \"%s\"", id, task.ID, task.Name)

			runnerFunc, ok := w.runnerMap[task.TaskType]
			var result string
			var state database.TaskState
			if !ok {
				log.Error("worker %d failed to find task runner for task id %d name %s task type %s", id, task.ID, task.Name, task.TaskType)
				state = database.TASK_PENDING
			} else {
				if err := runnerFunc.Run(ctx, w.db, task); err != nil {
					log.Error("worker %d failed to run task id %d name %s error %v", id, task.ID, task.Name, err)
					result = err.Error()
					state = database.TASK_PENDING
				} else {
					result = "success"
					state = database.TASK_FINISHED
				}
			}

			log.Debug("worker %d finished task id %d name \"%s\"", id, task.ID, task.Name)
			if err := w.resultTaskResult(log, id, task, state, result); err != nil {
				log.Error("worker %d failed to result task id %d name %s error %v", id, task.ID, task.Name, err)
			}
		}()
		wg.Wait()
	}
}
