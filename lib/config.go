package lib

type LaunchConfig struct {
	NumWorkers int    `json:"num_workers"`
	DBHost     string `json:"db_host"`
	DBPort     int    `json:"db_port"`
	DBUser     string `json:"db_user"`
	DBPassword string `json:"db_password"`
	DBName     string `json:"db_name"`
	Port       int    `json:"port"`
	LogLevel   string `json:"log_level"`
}
