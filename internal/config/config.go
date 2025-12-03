package config

type Config struct {
	DB     db
	Server server
	Logger logger
}

type db struct {
	DatabaseURI string `env:"DATABASE_URI"`
}

type server struct {
	RunAddress string `env:"RUN_ADDRESS"`
}

type logger struct {
	LogLevel string `env:"LOG_LEVEL" envDefault:"info"`
}
