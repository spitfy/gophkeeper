package client

import (
	"context"
	"golang.org/x/exp/slog"
	"gophkeeper/internal/app/client/config"
	"os"
	"testing"
	"time"
)

func TestSyncService(t *testing.T) {
	// Создаем тестовое приложение
	cfg := &config.Config{
		ConfigDir:     "/tmp/gophkeeper_test",
		ServerAddress: "localhost:8080",
		LogLevel:      "debug",
	}

	log := slog.New(nil)
	app, err := New(cfg, log)
	if err != nil {
		t.Fatalf("Ошибка создания приложения: %v", err)
	}
	defer os.RemoveAll("/tmp/gophkeeper_test")

	// Тестируем предварительные проверки
	ctx := context.Background()

	// Должна быть ошибка, так как нет аутентификации
	err = app.sync.preSyncChecks(ctx)
	if err == nil {
		t.Error("Ожидалась ошибка при отсутствии аутентификации")
	}

	// Тестируем обнаружение конфликтов
	t.Run("DetectConflicts", func(t *testing.T) {
		localRec := &Record{
			ID:        "test1",
			Type:      "password",
			Version:   2,
			UpdatedAt: time.Now(),
			Synced:    false,
		}

		serverRec := &Record{
			ID:        "test1",
			Type:      "password",
			Version:   1,
			UpdatedAt: time.Now().Add(-1 * time.Hour),
			Synced:    true,
		}

		conflict, err := app.sync.checkRecordConflict(localRec, serverRec)
		if err != nil {
			t.Fatalf("Ошибка проверки конфликта: %v", err)
		}

		if conflict == nil {
			t.Error("Ожидался конфликт")
		}

		if conflict.ConflictType != "edit-edit" {
			t.Errorf("Неверный тип конфликта: %s", conflict.ConflictType)
		}
	})

	// Тестируем автоматическое разрешение конфликтов
	t.Run("AutoResolveConflict", func(t *testing.T) {
		conflict := &Conflict{
			RecordID: "test1",
			LocalRecord: &Record{
				ID:        "test1",
				UpdatedAt: time.Now(),
				Version:   3,
			},
			ServerRecord: &Record{
				ID:        "test1",
				UpdatedAt: time.Now().Add(-2 * time.Hour),
				Version:   2,
			},
			ConflictType: "edit-edit",
			CreatedAt:    time.Now(),
		}

		// Тестируем стратегию "newer"
		app.sync.config.ConflictStrategy = "newer"
		resolved, err := app.sync.autoResolveConflict(conflict)
		if err != nil {
			t.Fatalf("Ошибка автоматического разрешения: %v", err)
		}

		if !resolved.Resolved {
			t.Error("Конфликт должен быть разрешен")
		}

		if resolved.Resolution != "client" {
			t.Errorf("Ожидалось разрешение 'client', получено: %s", resolved.Resolution)
		}

		// Тестируем стратегию "server"
		app.sync.config.ConflictStrategy = "server"
		resolved, err = app.sync.autoResolveConflict(conflict)
		if err != nil {
			t.Fatalf("Ошибка автоматического разрешения: %v", err)
		}

		if resolved.Resolution != "server" {
			t.Errorf("Ожидалось разрешение 'server', получено: %s", resolved.Resolution)
		}
	})
}
