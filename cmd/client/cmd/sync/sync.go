package sync

import (
	"context"
	"fmt"
	"gophkeeper/internal/app/client"
	"time"

	"github.com/spf13/cobra"
)

var (
	forceSync     bool
	syncStatus    bool
	resetStats    bool
	showConflicts bool
)

var SyncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Управление синхронизацией",
	Long: `Синхронизация данных между клиентом и сервером.
	
Команда позволяет управлять процессом синхронизации, просматривать статус
и разрешать конфликты.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		app := cmd.Context().Value("app").(*client.App)
		if app == nil {
			return fmt.Errorf("приложение не инициализировано")
		}

		if syncStatus {
			return showSyncStatus(cmd.Context(), app)
		}

		if resetStats {
			return resetSyncStats(app)
		}

		if showConflicts {
			return showSyncConflicts(cmd.Context(), app)
		}

		// Выполняем синхронизацию
		return runSync(cmd.Context(), app, forceSync)
	},
}

func runSync(ctx context.Context, app *client.App, force bool) error {
	fmt.Println("=== Синхронизация данных ===")

	if !app.IsAuthenticated() {
		return fmt.Errorf("требуется аутентификация. Выполните: gophkeeper auth login")
	}

	if !app.sync.config.Enabled {
		fmt.Println("⚠️  Синхронизация отключена в настройках")
		return nil
	}

	fmt.Println("Проверка соединения с сервером...")
	if err := app.CheckConnection(); err != nil {
		return fmt.Errorf("сервер недоступен: %v", err)
	}

	fmt.Println("Начало синхронизации...")
	start := time.Now()

	result, err := app.sync.Sync(ctx)
	if err != nil {
		return fmt.Errorf("ошибка синхронизации: %w", err)
	}

	duration := time.Since(start)

	fmt.Println()
	fmt.Println("✅ Синхронизация завершена!")
	fmt.Printf("Время выполнения: %v\n", duration.Round(time.Millisecond))
	fmt.Printf("Загружено на сервер: %d записей\n", result.Uploaded)
	fmt.Printf("Загружено с сервера: %d записей\n", result.Downloaded)

	if result.Conflicts > 0 {
		fmt.Printf("Обнаружено конфликтов: %d\n", result.Conflicts)
		fmt.Printf("Разрешено конфликтов: %d\n", result.Resolved)

		if result.Resolved < result.Conflicts {
			fmt.Println("⚠️  Некоторые конфликты не были разрешены автоматически")
			fmt.Println("   Используйте 'gophkeeper sync --conflicts' для просмотра")
		}
	}

	if len(result.Errors) > 0 {
		fmt.Printf("Ошибок при синхронизации: %d\n", len(result.Errors))
		for i, err := range result.Errors {
			if i < 3 { // Показываем только первые 3 ошибки
				fmt.Printf("  • %s: %s\n", err.Operation, err.Error)
			}
		}
		if len(result.Errors) > 3 {
			fmt.Printf("  ... и еще %d ошибок\n", len(result.Errors)-3)
		}
	}

	stats := app.sync.GetStats()
	fmt.Printf("Всего синхронизаций: %d\n", stats.TotalSyncs)
	fmt.Printf("Последняя успешная: %s\n",
		stats.LastSuccessful.Format("2006-01-02 15:04:05"))

	return nil
}

func showSyncStatus(ctx context.Context, app *client.App) error {
	fmt.Println("=== Статус синхронизации ===")

	stats := app.sync.GetStats()

	fmt.Println("📊 Статистика:")
	fmt.Printf("  Всего синхронизаций: %d\n", stats.TotalSyncs)
	fmt.Printf("  Успешных: %d\n", stats.TotalSyncs-stats.TotalErrors)
	fmt.Printf("  С ошибками: %d\n", stats.TotalErrors)
	fmt.Printf("  Загружено на сервер: %d записей\n", stats.TotalUploaded)
	fmt.Printf("  Загружено с сервера: %d записей\n", stats.TotalDownloaded)
	fmt.Printf("  Обнаружено конфликтов: %d\n", stats.TotalConflicts)
	fmt.Printf("  Разрешено конфликтов: %d\n", stats.TotalResolved)
	fmt.Printf("  Среднее время: %.2f сек\n", stats.AvgSyncDuration)

	if !stats.LastSuccessful.IsZero() {
		fmt.Printf("\n⏰ Временные метки:\n")
		fmt.Printf("  Последняя успешная: %s\n",
			stats.LastSuccessful.Format("2006-01-02 15:04:05"))
		fmt.Printf("  Последняя неудачная: %s\n",
			stats.LastFailed.Format("2006-01-02 15:04:05"))
	}

	fmt.Printf("\n⚙️  Конфигурация:\n")
	config := app.sync.config
	fmt.Printf("  Интервал: %v\n", config.Interval)
	fmt.Printf("  Размер пакета: %d записей\n", config.BatchSize)
	fmt.Printf("  Макс. попыток: %d\n", config.MaxRetries)
	fmt.Printf("  Стратегия конфликтов: %s\n", config.ConflictStrategy)
	fmt.Printf("  Авторазрешение: %v\n", config.AutoResolve)
	fmt.Printf("  Включена: %v\n", config.Enabled)

	// Проверяем соединение с сервером
	fmt.Printf("\n🌐 Соединение с сервером: ")
	if err := app.CheckConnection(); err != nil {
		fmt.Printf("❌ Ошибка: %v\n", err)
	} else {
		fmt.Printf("✅ OK\n")
	}

	// Проверяем аутентификацию
	fmt.Printf("🔐 Аутентификация: ")
	if app.IsAuthenticated() {
		fmt.Printf("✅ Выполнена\n")
	} else {
		fmt.Printf("❌ Требуется вход\n")
	}

	return nil
}

func resetSyncStats(app *client.App) error {
	app.sync.ResetStats()
	fmt.Println("✅ Статистика синхронизации сброшена")
	return nil
}

func showSyncConflicts(ctx context.Context, app *client.App) error {
	// TODO: Реализовать отображение конфликтов
	fmt.Println("Просмотр конфликтов будет реализован в будущей версии")
	return nil
}

func init() {
	SyncCmd.Flags().BoolVarP(&forceSync, "force", "f", false, "принудительная синхронизация")
	SyncCmd.Flags().BoolVar(&syncStatus, "status", false, "показать статус синхронизации")
	SyncCmd.Flags().BoolVar(&resetStats, "reset", false, "сбросить статистику синхронизации")
	SyncCmd.Flags().BoolVar(&showConflicts, "conflicts", false, "показать неразрешенные конфликты")
}
