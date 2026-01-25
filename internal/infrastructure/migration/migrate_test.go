package migration

import (
	"errors"
	"github.com/golang-migrate/migrate/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"

	"gophkeeper/internal/app/server/config"
)

// MockMigrator — мок для интерфейса Migrator
type MockMigrator struct {
	mock.Mock
}

func (m *MockMigrator) Up() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockMigrator) Close() (error, error) {
	args := m.Called()
	return args.Error(0), args.Error(1)
}

func TestMigration_Up_Success(t *testing.T) {
	cfg := &config.Config{
		DB: struct {
			DatabaseURI string `env:"DATABASE_URI"`
			Migrations  string `env:"MIGRATIONS_PATH"`
		}{DatabaseURI: "", Migrations: ""},
	}
	mockM := new(MockMigrator)

	// Настраиваем поведение
	mockM.On("Up").Return(nil)
	mockM.On("Close").Return(nil, nil)

	// Инжектим мок через фабрику
	engine := func(source, db string) (Migrator, error) {
		return mockM, nil
	}

	mg := NewMigration(cfg, engine)
	err := mg.Up()

	assert.NoError(t, err)
	mockM.AssertExpectations(t)
}

func TestMigration_Up_NoChange(t *testing.T) {
	cfg := &config.Config{
		DB: struct {
			DatabaseURI string `env:"DATABASE_URI"`
			Migrations  string `env:"MIGRATIONS_PATH"`
		}{DatabaseURI: "", Migrations: ""},
	}
	mockM := new(MockMigrator)

	// ErrNoChange не должна считаться ошибкой в методе Up()
	mockM.On("Up").Return(migrate.ErrNoChange)
	mockM.On("Close").Return(nil, nil)

	engine := func(source, db string) (Migrator, error) {
		return mockM, nil
	}

	mg := NewMigration(cfg, engine)
	err := mg.Up()

	assert.NoError(t, err)
}

func TestMigration_Up_EngineError(t *testing.T) {
	cfg := &config.Config{
		DB: struct {
			DatabaseURI string `env:"DATABASE_URI"`
			Migrations  string `env:"MIGRATIONS_PATH"`
		}{DatabaseURI: "", Migrations: ""},
	}

	// Ошибка на этапе создания мигратора (например, неверный драйвер)
	engine := func(source, db string) (Migrator, error) {
		return nil, errors.New("engine crash")
	}

	mg := NewMigration(cfg, engine)
	err := mg.Up()

	assert.Error(t, err)
	assert.Equal(t, "engine crash", err.Error())
}
