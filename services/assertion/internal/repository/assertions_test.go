package repository_test

import (
	"io"
	"os"
	"testing"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	"github.com/google/uuid"
	"github.com/idlab-discover/kebeng/services/assertion/internal/config"
	assertionDB "github.com/idlab-discover/kebeng/services/assertion/internal/database"
	"github.com/idlab-discover/kebeng/services/assertion/internal/repository"
	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

var (
	globalRepo        repository.IAssertionRepository
	globalDB          *sqlx.DB
	cleanupDB         func()
	mockUUID          = uuid.New()
	existingAssertion = `type: snap-build
authority-id: test-authority-id
snap-sha3-384: test-snap-sha3-384
developer-id: test-developer-id
grade: stable
snap-id: test-snap-id
snap-size: 1234567
timestamp: 2023-01-01T00:00:00+00:00
sign-key-sha3-384: test-sign-key-sha3-384

AcLBtest-signature-data`
)

func setupGlobalTestDB() (repository.IAssertionRepository, *sqlx.DB, func()) {
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
	err = assertionDB.RunMigrations(db, &cfg)
	if err != nil {
		logrus.Fatalf("failed to run migrations: %v", err)
	}

	repo := repository.NewAssertionRepository(db)

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

	mockData(globalDB)

	code := m.Run()

	cleanupDB()
	os.Exit(code)
}

func TestAddAssertion(t *testing.T) {
	tests := []struct {
		name          string
		SnapID        uuid.UUID
		assertion     string
		expectedError bool
		errorString   string
	}{
		{
			name:   "Succes uploading assertion",
			SnapID: mockUUID,
			assertion: `type: snap-build
			authority-id: test-authority-id
			snap-sha3-384: test-snap-sha3-384
			developer-id: test-developer-id-test
			grade: stable
			snap-id: test-snap-id
			snap-size: 12345
			timestamp: 2025-01-01T00:00:00+00:00
			sign-key-sha3-384: test-sign-key-sha3-384
			AcLBtest-signature-data-test`,
			expectedError: false,
		},
		{
			name:          "Fail uploading assertion for already existing assertion",
			SnapID:        mockUUID,
			assertion:     existingAssertion,
			expectedError: true,
			errorString:   "database-error",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertion, err := globalRepo.AddAssertion(tt.SnapID, tt.assertion)
			if tt.expectedError {
				assert.NotNil(t, err)
				if err != nil {
					assert.Equal(t, tt.errorString, err.GetCode())
				}
			} else {
				assert.Nil(t, err)
				assert.NotNil(t, assertion)
			}
		})
	}
}

// Helper function to insert mock data into the database
func mockData(db *sqlx.DB) {
	assertion := `type: snap-build
authority-id: test-authority-id
snap-sha3-384: test-snap-sha3-384
developer-id: test-developer-id
grade: stable
snap-id: test-snap-id
snap-size: 1234567
timestamp: 2023-01-01T00:00:00+00:00
sign-key-sha3-384: test-sign-key-sha3-384

AcLBtest-signature-data`

	_, err := db.Exec("INSERT INTO assertions (id, snap_entry_id, assertion) VALUES ($1, $2, $3)", mockUUID, mockUUID, assertion)
	if err != nil {
		logrus.Fatalf("failed to insert mock data: %v", err)
	}
}
