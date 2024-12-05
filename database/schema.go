package database

import (
	"database/sql"
	"time"
)

type TaskState int

const (
	TASK_PENDING TaskState = iota
	TASK_RUNNING
	TASK_FINISHED
	TASK_FAILED // Retries exhausted.
)

type LeibnizTable interface {
	CreateTable() string
}

// Task represents a task in the database. State should be used by workers to lock on tasks.
// Workers will need to update the state of the task as they process it. NumRetries should be
// incremented by 1 each time a task fails. MaxRetries should be set to the maximum number of
// retries allowed for a task.
// LastUpdated should be updated each time the task is updated.
type Task struct {
	ID              int            // ID of the task.
	Name            string         // Name of the task.
	Args            sql.NullString // Arguments to the task as a single string. Formatting is up to the task runner.
	NumRetries      int            // Number of retries so far.
	MaxRetries      int            // Maximum number of retries before marking the task as failed.
	State           int            // State of the task. See TaskState above.
	OwnerWorkerID   sql.NullInt32  // ID of the worker that owns the task.
	TimeoutSeconds  sql.NullInt64  // Timeout in seconds, 0 means no timeout.
	TaskResult      sql.NullString // Result of the task.
	RecurringTaskID sql.NullInt32  // ID of the recurring task if this task is part of a recurring task.
	TaskType        string         // Type of the task. Used to look up the task runner.
	LastUpdated     time.Time      // Last time the task was updated.
	CreatedAt       time.Time      // Time the task was created.
}

func (t *Task) CreateTable() string {
	return `CREATE TABLE IF NOT EXISTS leibniz_tasks (
		id SERIAL PRIMARY KEY,
		name TEXT NOT NULL,
		args TEXT,
		num_retries INTEGER NOT NULL,
		max_retries INTEGER NOT NULL,
		state INTEGER NOT NULL,
		owner_worker_id INTEGER,
		timeout_seconds INTEGER,
		task_result TEXT,
		recurring_task_id INTEGER,
		task_type TEXT NOT NULL,
		last_updated TIMESTAMP NOT NULL,
		created_at TIMESTAMP NOT NULL
	)`
}

type RecurringTask struct {
	ID            int       // ID of the task.
	Name          string    // Name of the task.
	Args          string    // Arguments to the task as a single string. Formatting is up to the task runner.
	MaxRetries    int       // Maximum number of retries before marking the task as failed.
	TaskType      string    // Type of the task. Used to look up the task runner.
	CronSpec      string    // Cron spec for the task e.g. "* * * * *".
	Timeout       int       // Timeout in seconds, 0 means no timeout.
	EnableOverrun bool      // If true, new tasks will be created even if the previous task is still running.
	CreatedAt     time.Time // Time this recurring entry was created.
}

func (r *RecurringTask) CreateTable() string {
	return `CREATE TABLE IF NOT EXISTS leibniz_recurring_tasks (
		id SERIAL PRIMARY KEY,
		name TEXT NOT NULL,
		args TEXT,
		max_retries INTEGER NOT NULL,
		task_type TEXT NOT NULL,
		cron_spec TEXT NOT NULL,
		timeout INTEGER NOT NULL,
		enable_overrun BOOLEAN NOT NULL,
		created_at TIMESTAMP NOT NULL
	)`
}
