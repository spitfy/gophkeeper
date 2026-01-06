package config

import (
	"github.com/joho/godotenv"
	"github.com/spf13/viper"
	"log"
)

const (
	envPath   = "../../.env"
	SecretKey = "SecRetKey"
	EnvLocal  = "local"
	EnvDev    = "dev"
	EnvProd   = "prod"
)

type Config struct {
	Env    string
	DB     db
	Server server
	Logger logger
}

type defaultConfig struct {
	RunAddress      string
	AccrualAddress  string
	AccrualInterval int
	DatabaseURI     string
	LogLevel        string
	Secret          string
	Env             string
	Migrations      string
}

type db struct {
	DatabaseURI string `env:"DATABASE_URI"`
	Migrations  string `env:"MIGRATIONS_PATH"`
}

type server struct {
	RunAddress string `env:"RUN_ADDRESS"`
}

type logger struct {
	LogLevel string `env:"LOG_LEVEL" envDefault:"info"`
}

func MustLoad() *Config {
	if err := godotenv.Load(envPath); err != nil {
		log.Fatalln("No .env file found, relying on environment variables")
	}

	viper.AutomaticEnv()
	d := defaultConfig{
		RunAddress:  viper.GetString("run_address"),
		DatabaseURI: viper.GetString("database_uri"),
		LogLevel:    viper.GetString("log_level"),
		Secret:      viper.GetString("secret"),
		Env:         viper.GetString("app_env"),
		Migrations:  viper.GetString("migrations_path"),
	}
	if d.Secret == "" {
		d.Secret = SecretKey
	}

	config := Config{
		Env: d.Env,
		DB: db{
			DatabaseURI: d.DatabaseURI,
			Migrations:  d.Migrations,
		},
		Server: server{RunAddress: d.RunAddress},
		Logger: logger{LogLevel: d.LogLevel},
	}

	return &config
}

func SaveToken(token string) interface{} {
	_ = token
	return nil
}
