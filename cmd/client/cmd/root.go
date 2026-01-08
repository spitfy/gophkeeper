// cmd/client/cmd/root.go
package cmd

import (
	"fmt"
	"golang.org/x/exp/slog"
	"os"
	"path/filepath"

	"gophkeeper/internal/app/client"
	"gophkeeper/internal/app/client/config"
	"gophkeeper/internal/utils/logger"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile    string
	cfg        *config.Config
	log        *slog.Logger
	app        *client.App
	debug      bool
	jsonOutput bool
	serverURL  string
)

var rootCmd = &cobra.Command{
	Use:   "gophkeeper",
	Short: "GophKeeper - клиент для безопасного хранения секретов",
	Long: `GophKeeper — это клиентское приложение для безопасного хранения 
паролей, заметок, кредитных карт и другой конфиденциальной информации.
	
Все данные шифруются на стороне клиента с использованием мастер-ключа
и синхронизируются с сервером.`,
	PersistentPreRunE: setupApp,
	SilenceUsage:      true,
	SilenceErrors:     true,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка: %v\n", err)
		os.Exit(1)
	}
}

func setupApp(_ *cobra.Command, _ []string) error {
	// Загружаем конфигурацию
	var err error
	cfg, err = loadConfig()
	if err != nil {
		return fmt.Errorf("ошибка загрузки конфигурации: %w", err)
	}

	// Переопределяем настройки из флагов командной строки
	if serverURL != "" {
		cfg.ServerAddress = serverURL
	}

	// Настраиваем логгер
	log = logger.New(cfg.Env)

	// Создаем приложение
	app, err = client.New(cfg, log)
	if err != nil {
		return fmt.Errorf("ошибка инициализации приложения: %w", err)
	}

	return nil
}

func loadConfig() (*config.Config, error) {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		// Ищем конфиг в стандартных местах
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}

		configDir := filepath.Join(home, ".gophkeeper")
		viper.AddConfigPath(configDir)
		viper.AddConfigPath(".")
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
	}

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, err
		}
		// Конфиг не найден, используем значения по умолчанию
	}

	// Загружаем конфигурацию через стандартный метод
	return config.MustLoad(), nil
}

func init() {
	cobra.OnInitialize()

	// Глобальные флаги
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "конфигурационный файл")
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "включить отладочный режим")
	rootCmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "вывод в формате JSON")
	rootCmd.PersistentFlags().StringVar(&serverURL, "server", "", "URL сервера GophKeeper")

	// Команды будут добавлены в init() соответствующих файлов
}
