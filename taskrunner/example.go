package taskrunner

import (
	"context"
	"database/sql"
	"log"
	"time"

	"movinglake.com/leibniz/database"
)

type ExampleTaskRunner struct {
}

func (e *ExampleTaskRunner) Run(ctx context.Context, db *sql.DB, task *database.Task) error {
	log.Printf("Running task: %+v\n", task)
	time.Sleep(5 * time.Second)
	log.Printf("Task finished: %+v\n", task)
	// Add your task logic here.
	return nil
}
