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

type Migration struct {
	cfg *config.Config
}

func NewMigration(conf *config.Config) *Migration {
	return &Migration{conf}
}

func (mg *Migration) Up() error {
	m, err := migrate.New(
		"file://"+mg.cfg.DB.Migrations,
		mg.cfg.DB.DatabaseURI,
	)
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
