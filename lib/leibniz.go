package lib

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/gorhill/cronexpr"
	_ "github.com/lib/pq"
	"movinglake.com/leibniz/database"
)

const (
	LaunchConfigDefaultFilePath = "launch_config.json"
)

type RecurringTask struct {
	Name          string // Name of the task.
	Args          string // Arguments to the task as a single string. Formatting is up to the task runner.
	MaxRetries    int    // Maximum number of retries before marking the task as failed.
	TaskType      string // Type of the task. Used to look up the task runner.
	CronSpec      string // Cron spec for the task e.g. "* * * * *".
	Timeout       int    // Timeout in seconds, 0 means no timeout.
	EnableOverrun bool   // If true, new tasks will be created even if the previous task is still running.
}

type Leibniz struct {
	TaskRunners    map[string]TaskRunner         // Map of task types to task runners.
	Endpoints      map[string]LeibnizHTTPHandler // Map of paths to http.Handler.
	AllowedMethods map[string]map[string]bool    // Map of paths to allowed methods.
	RecurringTasks []RecurringTask               // List of recurring tasks.
}

func New() *Leibniz {
	return &Leibniz{
		TaskRunners:    make(map[string]TaskRunner),
		Endpoints:      make(map[string]LeibnizHTTPHandler),
		AllowedMethods: make(map[string]map[string]bool),
		RecurringTasks: make([]RecurringTask, 0),
	}
}

func (l *Leibniz) AddRecurringTask(task RecurringTask) {
	l.RecurringTasks = append(l.RecurringTasks, task)
}

func (l *Leibniz) AddTaskRunner(taskType string, tr TaskRunner) {
	l.TaskRunners[taskType] = tr
}

func (l *Leibniz) AddEndpoint(method, path string, e LeibnizHTTPHandler) {
	l.Endpoints[path] = e
	if _, ok := l.AllowedMethods[path]; !ok {
		l.AllowedMethods[path] = make(map[string]bool)
	}
	l.AllowedMethods[path][method] = true
}

// readLaunchConfig reads the launch configuration FROM leibniz_a json file at the root of the project.
func readLaunchConfig() (*LaunchConfig, error) {
	log := NewLogger("launch-config-reader")
	// Try to get launch config file path from environment variable.
	filePath := LaunchConfigDefaultFilePath
	if path, ok := os.LookupEnv("LEIBNIZ_LAUNCH_CONFIG_FILE"); ok {
		filePath = path
	}
	log.Info("Reading launch config file at %s", filePath)
	contents, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read launch config file: %s", err)
	}

	cfg := &LaunchConfig{}
	if err = json.Unmarshal(contents, &cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal launch config: %s", err)
	}
	log.Info("Using launch config: %+v", cfg)
	return cfg, nil
}

func (l *Leibniz) launchWorkerPool(ctx context.Context, db *sql.DB, cfg *LaunchConfig) {
	log := NewLogger("worker-launcher")
	tasks := make(chan *database.Task)
	for i := 0; i < cfg.NumWorkers; i++ {
		worker := NewWorker(db, l.TaskRunners)
		go worker.Run(ctx, i, tasks)
	}
	// Read tasks from database and send them to the worker pool.
	for {
		rows, err := db.Query("SELECT * FROM leibniz_tasks WHERE state = $1", database.TASK_PENDING)
		if err != nil {
			log.Error("Failed to fetch tasks: %s", err)
		}

		for rows.Next() {
			task := &database.Task{}
			if err := rows.Scan(
				&task.ID,
				&task.Name,
				&task.Args,
				&task.NumRetries,
				&task.MaxRetries,
				&task.State,
				&task.OwnerWorkerID,
				&task.TimeoutSeconds,
				&task.TaskResult,
				&task.RecurringTaskID,
				&task.TaskType,
				&task.LastUpdated,
				&task.CreatedAt,
			); err != nil {
				log.Error("Failed to scan task: %s", err)
			}
			tasks <- task
		}
		rows.Close()
		time.Sleep((5 * time.Second))
	}
}

func (l *Leibniz) launchRecurringTasks(db *sql.DB) {
	log := NewLogger("recurring-task-launcher")
	for {
		// Fetch all recurring tasks.
		rows, err := db.Query("SELECT * FROM leibniz_recurring_tasks")
		if err != nil {
			log.Error("Failed to fetch recurring tasks: %s", err)
		}
		currTime := time.Now()
		for rows.Next() {
			task := &database.RecurringTask{}
			if err := rows.Scan(&task.ID, &task.Name, &task.Args, &task.MaxRetries, &task.TaskType, &task.CronSpec, &task.Timeout, &task.EnableOverrun, &task.CreatedAt); err != nil {
				log.Error("Failed to scan recurring task: %s", err)
				continue
			}
			log.Debug("Processing recurring task: %d %s", task.ID, task.Name)
			// Fetch the last time the task was run.
			var lastRun database.Task
			if err := db.QueryRow("SELECT id, state, created_at FROM leibniz_tasks t WHERE t.recurring_task_id = $1 ORDER BY t.created_at DESC LIMIT 1", task.ID).Scan(
				&lastRun.ID, &lastRun.State, &lastRun.CreatedAt); err != nil {
				if err != sql.ErrNoRows {
					log.Error("Failed to fetch last run task for ID %d: %s", task.ID, err)
					continue
				}
				lastRun.State = int(database.TASK_FINISHED) // No previous run.
			}
			if !task.EnableOverrun {
				if lastRun.State == int(database.TASK_RUNNING) {
					log.Warn("Task %d is still running, skipping", lastRun.ID)
					continue
				}
			}
			// Schedule the task if it is time to run.
			expr, err := cronexpr.Parse(task.CronSpec)
			if err != nil {
				log.Error("Failed to parse cron spec %s: %s", task.CronSpec, err)
				continue
			}
			if nextTime := expr.Next(lastRun.CreatedAt); nextTime.Before(currTime) {
				// Run the task.
				log.Info("Inserting new recurring instance task %d: %s", task.ID, task.Name)
				r, err := db.Exec(
					"INSERT INTO leibniz_tasks (name, args, num_retries, max_retries, state, task_type, recurring_task_id, last_updated, created_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)",
					task.Name,
					task.Args,
					0,
					task.MaxRetries,
					database.TASK_PENDING,
					task.TaskType,
					task.ID,
					currTime,
					currTime)
				if err != nil {
					log.Error("Failed to insert task: %s", err)
					continue
				}
				log.Debug("Inserted task: %s", r)
			}
		}
		time.Sleep(60 * time.Second)
	}
}

func (l *Leibniz) createSchema(db *sql.DB) {
	log := NewLogger("schema-creator")
	// Create the schema in the database.
	_, err := db.Exec((&database.Task{}).CreateTable())
	if err != nil {
		log.Fatal("Failed to create leibniz_tasks table: %s", err)
	}
	_, err = db.Exec((&database.RecurringTask{}).CreateTable())
	if err != nil {
		log.Fatal("Failed to create leibniz_recurring_tasks table: %s", err)
	}
}

func (l *Leibniz) Start() error {
	log := NewLogger("leibniz-start")
	ctx := context.Background()
	cfg, err := readLaunchConfig()
	if err != nil {
		return fmt.Errorf("failed to read launch config: %s", err)
	}
	// Connect to the database.
	db, err := sql.Open("postgres", fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable", cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName))
	if err != nil {
		return fmt.Errorf("failed to connect to database: %s", err)
	}
	defer db.Close()
	l.createSchema(db)
	go l.launchWorkerPool(ctx, db, cfg)

	// Write recurring tasks to the database.
	for _, task := range l.RecurringTasks {
		r, err := db.Exec(
			"INSERT INTO leibniz_recurring_tasks (name, args, max_retries, task_type, cron_spec, timeout, enable_overrun, created_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)",
			task.Name,
			task.Args,
			task.MaxRetries,
			task.TaskType,
			task.CronSpec,
			task.Timeout,
			task.EnableOverrun,
			time.Now())
		if err != nil {
			return fmt.Errorf("failed to insert recurring task: %s", err)
		}
		log.Debug("Inserted recurring task: %s", r)
	}

	go l.launchRecurringTasks(db)

	log.Info("Starting HTTP server...")
	if err := http.ListenAndServe(":8080", NewHandler(cfg, db, l.Endpoints, l.AllowedMethods)); err != nil {
		return fmt.Errorf("failed to start HTTP server: %s", err)
	}
	return nil
}
