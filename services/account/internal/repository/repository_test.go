package repository_test

import (
	"context"
	"fmt"
	"io"
	"os"
	"testing"
	"time"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3" // SQLite driver
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"

	_ "github.com/golang-migrate/migrate/v4/source/file" // needed for file source

	accountDB "github.com/idlab-discover/kebeng/services/account/internal"
	"github.com/idlab-discover/kebeng/services/account/internal/config"
	"github.com/idlab-discover/kebeng/services/account/internal/models"
	"github.com/idlab-discover/kebeng/services/account/internal/repository"
)

var (
	globalRepo *repository.AccountRepository
	globalDB   *sqlx.DB
	cleanupDB  func()
)

func setupGlobalTestDB() (*repository.AccountRepository, *sqlx.DB, func()) {
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
	err = accountDB.RunMigrations(db, &cfg)
	if err != nil {
		logrus.Fatalf("failed to run migrations: %v", err)
	}

	repo := repository.NewAccountRepository(db)

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

func TestCreateAccount(t *testing.T) {
	created_at := time.Now()
	updated_at := time.Now()
	tests := []struct {
		name                 string
		account              *models.Account
		expectError          bool
		expectedErrorMessage string
	}{
		{
			name: "Successful account creation",
			account: &models.Account{
				DisplayName:  "Alice",
				Username:     "alice123",
				Email:        "alice@example.com",
				PasswordHash: "hash1",
				CreatedAt:    &created_at,
				UpdatedAt:    &updated_at,
			},
			expectError: false,
		},
		{
			name: "Duplicate username",
			account: &models.Account{
				DisplayName:  "Alice2",
				Username:     "alice123", // duplicate username from first test case
				Email:        "alice2@example.com",
				PasswordHash: "hash2",
				CreatedAt:    &created_at,
				UpdatedAt:    &updated_at,
			},
			expectError:          true,
			expectedErrorMessage: "duplicate key value violates unique constraint \"accounts_username_key\"",
		},
		{
			name: "Duplicate email",
			account: &models.Account{
				DisplayName:  "Alice3",
				Username:     "alice3",
				Email:        "alice@example.com", // duplicate email from first test case
				PasswordHash: "hash3",
				CreatedAt:    &created_at,
				UpdatedAt:    &updated_at,
			},
			expectError:          true,
			expectedErrorMessage: "duplicate key value violates unique constraint \"accounts_email_key\"",
		},
		{
			name: "Missing username",
			account: &models.Account{
				DisplayName:  "Alice Missing Username",
				Username:     "", // missing username
				Email:        "alice_missing_username@example.com",
				PasswordHash: "hashMissing",
				CreatedAt:    &created_at,
				UpdatedAt:    &updated_at,
			},
			expectError:          true,
			expectedErrorMessage: "violates check constraint \"accounts_username_check\"",
		},
		{
			name: "Missing email",
			account: &models.Account{
				DisplayName:  "Alice Missing Email",
				Username:     "aliceMissingEmail",
				Email:        "", // missing email
				PasswordHash: "hashMissing",
				CreatedAt:    &created_at,
				UpdatedAt:    &updated_at,
			},
			expectError:          true,
			expectedErrorMessage: "violates check constraint \"accounts_email_check\"",
		},
		{
			name: "Missing password hash",
			account: &models.Account{
				DisplayName:  "Alice Missing Password",
				Username:     "aliceMissingPass",
				Email:        "alice_missing_pass@example.com",
				PasswordHash: "", // missing password hash
				CreatedAt:    &created_at,
				UpdatedAt:    &updated_at,
			},
			expectError:          true,
			expectedErrorMessage: "violates check constraint \"accounts_password_hash_check\"",
		},
		{
			name: "Missing display name",
			account: &models.Account{
				DisplayName:  "", // missing display name
				Username:     "aliceMissingDisplay",
				Email:        "alice_missing_display@example.com",
				PasswordHash: "hashMissing",
				CreatedAt:    &created_at,
				UpdatedAt:    &updated_at,
			},
			expectError:          true,
			expectedErrorMessage: "violates check constraint \"accounts_display_name_check\"",
		},
	}

	// Execute each test case.
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			createdAccount, err := globalRepo.CreateAccount(context.Background(), tt.account)
			if tt.expectError {
				// We expect an error, so assert that err is non-nil.
				assert.Error(t, err, "Expected an error during account creation")
				if err != nil {
					assert.Contains(t, err.Error(), tt.expectedErrorMessage)
				}
			} else {
				fmt.Printf("created account: %+v\n", createdAccount)
				// No error expected, so assert that err is nil and the returned account is correct.
				assert.NoError(t, err, "Did not expect an error during account creation")
				assert.NotNil(t, createdAccount, "Created account should not be nil")
				assert.Equal(t, tt.account.DisplayName, createdAccount.DisplayName)
				assert.Equal(t, tt.account.Username, createdAccount.Username)
				assert.Equal(t, tt.account.Email, createdAccount.Email)
				assert.Equal(t, tt.account.PasswordHash, createdAccount.PasswordHash)
				assert.Equal(t, tt.account.CreatedAt, createdAccount.CreatedAt)
				assert.Equal(t, tt.account.UpdatedAt, createdAccount.UpdatedAt)
				assert.NotEqual(t, "00000000-0000-0000-0000-000000000000", createdAccount.ID, "Created account ID should be a valid uuid")

			}
		})
	}
}

func ptrTime(t time.Time) *time.Time {
	return &t
}

func TestUpdateAccount(t *testing.T) {
	// Create an initial account that will be updated.
	createdAt := time.Now()
	updatedAt := time.Now()
	original := &models.Account{
		ID:           uuid.New(), // manually generate a valid UUID
		DisplayName:  "Ben",
		Username:     "Ben123",
		Email:        "Ben@example.com",
		PasswordHash: "BenHash1",
		CreatedAt:    &createdAt,
		UpdatedAt:    &updatedAt,
	}

	insertQuery := `
		INSERT INTO accounts (id, display_name, username, email, password_hash, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	// Use positional parameters with ExecContext.
	_, err := globalDB.ExecContext(context.Background(), insertQuery,
		original.ID.String(),
		original.DisplayName,
		original.Username,
		original.Email,
		original.PasswordHash,
		original.CreatedAt,
		original.UpdatedAt,
	)
	if err != nil {
		t.Fatalf("failed to insert initial account: %v", err)
	}

	validation := "validated"
	// Table-driven tests for UpdateAccount.
	tests := []struct {
		name                 string
		updateAccount        *models.Account
		expectError          bool
		expectedErrorMessage string // Expected substring in the error message.
	}{
		{
			name: "Successful update",
			updateAccount: &models.Account{
				ID:           original.ID,
				DisplayName:  "Alice Updated",
				Username:     "alice_updated",
				Email:        "alice_updated@example.com",
				PasswordHash: "hash_updated",
				Validation:   &validation,
				UpdatedAt:    ptrTime(time.Now()),
			},
			expectError: false,
		},
		{
			name: "Update non-existing account",
			updateAccount: &models.Account{
				ID:           uuid.New(), // non-existent id
				DisplayName:  "No One",
				Username:     "noone",
				Email:        "noone@example.com",
				PasswordHash: "hash",
				Validation:   &validation,
				UpdatedAt:    ptrTime(time.Now()),
			},
			expectError:          true,
			expectedErrorMessage: "no account updated",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			updated, err := globalRepo.UpdateAccount(context.Background(), tt.updateAccount)
			if tt.expectError {
				assert.Error(t, err, "Expected an error during account update")
				if err != nil {
					assert.Contains(t, err.Error(), tt.expectedErrorMessage)
				}
			} else {
				assert.NoError(t, err, "Did not expect an error during account update")
				assert.NotNil(t, updated, "Updated account should not be nil")
				assert.Equal(t, tt.updateAccount.DisplayName, updated.DisplayName)
				assert.Equal(t, tt.updateAccount.Username, updated.Username)
				assert.Equal(t, tt.updateAccount.Email, updated.Email)
				// Check password; note: ensure your query and model field names match.
				assert.Equal(t, tt.updateAccount.PasswordHash, updated.PasswordHash)
				assert.Equal(t, tt.updateAccount.Validation, updated.Validation)
				// Optionally, you can also compare UpdatedAt if needed.
			}
		})
	}
}

func TestDeleteAccount(t *testing.T) {
	// Insert an initial account using globalDB with positional parameters.
	createdAt := time.Now()
	updatedAt := time.Now()
	original := &models.Account{
		ID:           uuid.New(),
		DisplayName:  "Charlie",
		Username:     "charlie123",
		Email:        "charlie@example.com",
		PasswordHash: "charlieHash1",
		CreatedAt:    &createdAt,
		UpdatedAt:    &updatedAt,
	}

	insertQuery := `
		INSERT INTO accounts (id, display_name, username, email, password_hash, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := globalDB.ExecContext(context.Background(), insertQuery,
		original.ID,
		original.DisplayName,
		original.Username,
		original.Email,
		original.PasswordHash,
		original.CreatedAt,
		original.UpdatedAt,
	)
	if err != nil {
		t.Fatalf("failed to insert initial account: %v", err)
	}
	t.Logf("Inserted initial account: %+v", original)

	t.Run("delete existing account", func(t *testing.T) {
		err := globalRepo.DeleteAccount(context.Background(), original.ID)
		assert.NoError(t, err, "expected no error when deleting an existing account")

		var count int
		err = globalDB.GetContext(context.Background(), &count, "SELECT COUNT(*) FROM accounts WHERE id = $1", original.ID)
		assert.NoError(t, err, "failed to count rows")
		assert.Equal(t, 0, count, "account should be deleted")
	})

	t.Run("delete non-existing account", func(t *testing.T) {
		nonExistentID := uuid.New()
		err := globalRepo.DeleteAccount(context.Background(), nonExistentID)
		assert.NoError(t, err, "expected no error when deleting a non-existent account")
	})
}
