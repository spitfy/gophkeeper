package migration

import (
	"errors"
	"fmt"
	"gophkeeper/internal/app/server/config"

	"github.com/golang-migrate/migrate/v4"
	// Blank import required for PostgreSQL driver registration for migrations
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// Migrator — интерфейс для самой библиотеки migrate.Migrate
type Migrator interface {
	Up() error
	Close() (error, error)
}

// MigrationEngine — фабрика для создания мигратора (чтобы не лезть в ФС и БД в тестах)
type MigrationEngine func(sourceURL, databaseURL string) (Migrator, error)

type Migration struct {
	cfg    *config.Config
	engine MigrationEngine
}

func NewMigration(conf *config.Config, engine MigrationEngine) *Migration {
	return &Migration{
		cfg:    conf,
		engine: engine,
	}
}

// DefaultEngine — реальная реализация для продакшена
func DefaultEngine(sourceURL, databaseURL string) (Migrator, error) {
	return migrate.New(sourceURL, databaseURL)
}

func (mg *Migration) Up() error {
	m, err := mg.engine("file://"+mg.cfg.DB.Migrations, mg.cfg.DB.DatabaseURI)
	if err != nil {
		return err
	}
	defer func() {
		serr, dberr := m.Close()
		if serr != nil {
			if err != nil {
				err = fmt.Errorf("%w; migration source error: %v", err, serr)
			} else {
				err = serr
			}
		}
		if dberr != nil {
			if err != nil {
				err = fmt.Errorf("%w; migration database error: %v", err, dberr)
			} else {
				err = dberr
			}
		}
	}()
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("%w; migration up error", err)
	}
	return nil
}
