package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

const (
	defaultServerAddress = "localhost:8080"
	defaultLogLevel      = "info"
	defaultEnv           = "local"
	defaultMasterKeyPath = ".master.key"
	defaultConfigDir     = ".gophkeeper"
)

type Config struct {
	Env           string `mapstructure:"app_env"`
	ServerAddress string `mapstructure:"server_address"`
	LogLevel      string `mapstructure:"log_level"`
	MasterKeyPath string `mapstructure:"master_key_path"`
	ConfigDir     string `mapstructure:"config_dir"`
	TokenPath     string `mapstructure:"token_path"`
	DataPath      string `mapstructure:"data_path"`
	SyncInterval  int    `mapstructure:"sync_interval_seconds"`
	EnableTLS     bool   `mapstructure:"enable_tls"`
	CACertPath    string `mapstructure:"ca_cert_path"`
}

// MustLoad загружает конфигурацию клиента
func MustLoad() *Config {
	// Определяем путь к .env файлу (относительно места запуска)
	envPath := ".env"
	if _, err := os.Stat(envPath); os.IsNotExist(err) {
		// Пробуем найти .env в родительской директории
		envPath = "../.env"
	}

	// Загружаем .env файл если существует
	if _, err := os.Stat(envPath); err == nil {
		if err := godotenv.Load(envPath); err != nil {
			fmt.Printf("Ошибка загрузки .env файла: %v\n", err)
		}
	}

	viper.AutomaticEnv()

	// Устанавливаем значения по умолчанию
	viper.SetDefault("APP_ENV", defaultEnv)
	viper.SetDefault("SERVER_ADDRESS", defaultServerAddress)
	viper.SetDefault("LOG_LEVEL", defaultLogLevel)
	viper.SetDefault("MASTER_KEY_PATH", defaultMasterKeyPath)
	viper.SetDefault("CONFIG_DIR", defaultConfigDir)
	viper.SetDefault("SYNC_INTERVAL_SECONDS", 30)
	viper.SetDefault("ENABLE_TLS", true)

	// Получаем домашнюю директорию пользователя
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}

	// Вычисляем пути для хранения данных
	configDir := viper.GetString("CONFIG_DIR")
	if configDir == defaultConfigDir {
		configDir = filepath.Join(homeDir, configDir)
	}

	// Создаем директории если их нет
	if err := os.MkdirAll(configDir, 0700); err != nil {
		fmt.Printf("Ошибка создания директории конфигурации: %v\n", err)
	}

	masterKeyPath := viper.GetString("MASTER_KEY_PATH")
	if masterKeyPath == defaultMasterKeyPath {
		masterKeyPath = filepath.Join(configDir, masterKeyPath)
	}

	tokenPath := filepath.Join(configDir, "token")
	dataPath := filepath.Join(configDir, "data.json")

	config := &Config{
		Env:           viper.GetString("APP_ENV"),
		ServerAddress: viper.GetString("SERVER_ADDRESS"),
		LogLevel:      viper.GetString("LOG_LEVEL"),
		MasterKeyPath: masterKeyPath,
		ConfigDir:     configDir,
		TokenPath:     tokenPath,
		DataPath:      dataPath,
		SyncInterval:  viper.GetInt("SYNC_INTERVAL_SECONDS"),
		EnableTLS:     viper.GetBool("ENABLE_TLS"),
		CACertPath:    viper.GetString("CA_CERT_PATH"),
	}

	// Валидация конфигурации
	if err := config.validate(); err != nil {
		panic(fmt.Sprintf("Ошибка конфигурации: %v", err))
	}

	return config
}

func (c *Config) validate() error {
	if c.ServerAddress == "" {
		return fmt.Errorf("server_address не может быть пустым")
	}
	if c.MasterKeyPath == "" {
		return fmt.Errorf("master_key_path не может быть пустым")
	}
	return nil
}

// IsProd проверяет, prod ли окружение
func (c *Config) IsProd() bool {
	return c.Env == "prod"
}

// IsDev проверяет, dev ли окружение
func (c *Config) IsDev() bool {
	return c.Env == "dev"
}

// IsLocal проверяет, local ли окружение
func (c *Config) IsLocal() bool {
	return c.Env == "local" || c.Env == ""
}
