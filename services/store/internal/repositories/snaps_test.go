package repositories_test

import (
	"io"
	"os"
	"testing"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3" // SQLite driver
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"

	_ "github.com/golang-migrate/migrate/v4/source/file" // needed for file source

	cerrors "github.com/idlab-discover/kebeng/common/error"
	storeDB "github.com/idlab-discover/kebeng/services/store/internal"
	"github.com/idlab-discover/kebeng/services/store/internal/config"
	"github.com/idlab-discover/kebeng/services/store/internal/repositories"
)

var (
	globalRepo *repositories.SnapsRepository
	globalDB   *sqlx.DB
	cleanupDB  func()
)

func setupGlobalTestDB() (*repositories.SnapsRepository, *sqlx.DB, func()) {
	postgres := embeddedpostgres.NewDatabase(embeddedpostgres.DefaultConfig().
		Port(5433).
		Version(embeddedpostgres.V12).
		Logger(io.Discard),
	)

	if postgresStartErr := postgres.Start(); postgresStartErr != nil {
		logrus.Fatalf("failed to start embedded postgres: %v", postgresStartErr)
	}

	dsn := "postgres://postgres:postgres@localhost:5433/postgres?sslmode=disable"
	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		logrus.Fatalf("failed to connect to embedded postgres: %v", err)
	}

	cfg := config.Config{MigrationPath: "../migrations"}
	err = storeDB.RunMigrations(db, &cfg)
	if err != nil {
		logrus.Fatalf("failed to run migrations: %v", err)
	}

	repo := repositories.NewSnapsRepository(db)

	_, err = repo.RegisterSnap("test", false)
	if err != nil {
		logrus.Fatalf("failed to register existing snap: %v", err)
	}

	cleanup := func() {
		err := db.Close()
		if err != nil {
			logrus.Fatalf("failed to close db: %v", err)
		}
		if err := postgres.Stop(); err != nil {
			logrus.Fatalf("failed to stop embedded postgres: %v", err)
		}
	}
	return repo, db, cleanup
}

func TestMain(m *testing.M) {
	repo, db, cleanup := setupGlobalTestDB()
	globalRepo = repo
	globalDB = db
	cleanupDB = cleanup

	code := m.Run()

	cleanupDB()
	os.Exit(code)
}

func TestGetEntryByName(t *testing.T) {
	tests := []struct {
		name                 string
		entryName            string
		entryPrivate         bool
		expectError          bool
		expectedErrorCode	 string
	}{
		{
			name:                 "Success getting entry by name",
			entryName:            "test",
			entryPrivate:         false,
			expectError:          false,
			expectedErrorCode:    "",
		},
		{
			name:                 "Fail getting entry by name",
			entryName:            "nonexistent",
			entryPrivate:         false,
			expectError:          true,
			expectedErrorCode:    cerrors.ResourceNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry, err := globalRepo.GetEntryByName(tt.entryName, nil)
			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErrorCode)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, entry.ID)
				assert.Equal(t, tt.entryName, entry.Name)
			}
		})
	}
}
