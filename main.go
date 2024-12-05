package main

import (
	"log"

	"movinglake.com/leibniz/httpendpoints"
	"movinglake.com/leibniz/lib"
	"movinglake.com/leibniz/taskrunner"
)

// Sample main function to start Leibniz.
func main() {
	l := lib.New()
	l.AddEndpoint("GET", "/", &httpendpoints.ExampleEndpoint{})
	l.AddTaskRunner("example_task_runner", &taskrunner.ExampleTaskRunner{})
	l.AddRecurringTask(lib.RecurringTask{
		Name:       "example",
		Args:       "Sample args",
		MaxRetries: 3,
		TaskType:   "example_task_runner", // Task type must match the task runner type.
		CronSpec:   "* * * * *",           // Every minute.
		Timeout:    0,                     // No timeout.
	})
	err := l.Start()
	if err != nil {
		log.Println("Error starting Leibniz: ", err)
	}
}
