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

	if !app.IsMasterKeyUnlocked() {
		fmt.Println("❌ Мастер-ключ заблокирован")
		fmt.Println()
		fmt.Println("Для синхронизации необходимо разблокировать мастер-ключ.")
		fmt.Println("Выполните команду: gophkeeper unlock")
		return fmt.Errorf("мастер-ключ заблокирован")
	}

	syncService := app.GetSyncService()

	fmt.Println("Проверка соединения с сервером...")
	if err := app.CheckConnection(); err != nil {
		return fmt.Errorf("сервер недоступен: %v", err)
	}

	fmt.Println("Начало синхронизации...")
	start := time.Now()

	result, err := app.Sync(ctx)
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

	stats := syncService.GetStats()
	fmt.Printf("Всего синхронизаций: %d\n", stats.TotalSyncs)
	if !stats.LastSync.IsZero() {
		fmt.Printf("Последняя синхронизация: %s\n",
			stats.LastSync.Format("2006-01-02 15:04:05"))
	}

	return nil
}

func showSyncStatus(ctx context.Context, app *client.App) error {
	fmt.Println("=== Статус синхронизации ===")

	syncService := app.GetSyncService()
	stats := syncService.GetStats()

	fmt.Println("📊 Статистика:")
	fmt.Printf("  Всего синхронизаций: %d\n", stats.TotalSyncs)
	fmt.Printf("  Успешных: %d\n", stats.TotalSyncs-stats.TotalErrors)
	fmt.Printf("  С ошибками: %d\n", stats.TotalErrors)
	fmt.Printf("  Загружено на сервер: %d записей\n", stats.TotalUploads)
	fmt.Printf("  Загружено с сервера: %d записей\n", stats.TotalDownloads)
	fmt.Printf("  Обнаружено конфликтов: %d\n", stats.TotalConflicts)
	fmt.Printf("  Разрешено конфликтов: %d\n", stats.TotalResolved)
	fmt.Printf("  Среднее время: %.2f сек\n", stats.AvgSyncDuration)

	if !stats.LastSync.IsZero() {
		fmt.Printf("\n⏰ Временные метки:\n")
		fmt.Printf("  Последняя синхронизация: %s\n",
			stats.LastSync.Format("2006-01-02 15:04:05"))
	}

	fmt.Printf("\n⚙️  Конфигурация: (используйте файл sync_config.json для настройки)\n")

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
	syncService := app.GetSyncService()
	syncService.ResetStats()
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
