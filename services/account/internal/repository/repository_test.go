package repository_test

import (
	"context"
	"io"
	"os"
	"testing"
	"time"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	"github.com/google/uuid"
	cerror "github.com/idlab-discover/kebeng/common/cerror"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3" // SQLite driver
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"

	_ "github.com/golang-migrate/migrate/v4/source/file" // needed for file source

	"github.com/idlab-discover/kebeng/services/account/internal/config"
	accountDB "github.com/idlab-discover/kebeng/services/account/internal/database"
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
	tests := []struct {
		name              string
		account           *models.Account
		expectError       bool
		expectedErrorCode string
	}{
		{
			name: "Successful account creation",
			account: &models.Account{
				DisplayName:  "AliceValid",
				Username:     "alice123",
				Email:        "alice@example.com",
				PasswordHash: "hash1",
			},
			expectError: false,
		},
		{
			name: "Duplicate username",
			account: &models.Account{
				DisplayName:  "AliceDuplicateUsername",
				Username:     "alice123", // Duplicate username
				Email:        "alice2@example.com",
				PasswordHash: "hash2",
			},
			expectError:       true,
			expectedErrorCode: cerror.AlreadyRegistered,
		},
		{
			name: "Duplicate email",
			account: &models.Account{
				DisplayName:  "AliceDuplicateEmail",
				Username:     "alice3",
				Email:        "alice@example.com", // Duplicate email
				PasswordHash: "hash3",
			},
			expectError:       true,
			expectedErrorCode: cerror.AlreadyRegistered,
		},
		{
			name: "Missing username",
			account: &models.Account{
				DisplayName:  "AliceMissingUsername",
				Username:     "", // Invalid
				Email:        "alice_missing_username@example.com",
				PasswordHash: "hashMissing",
			},
			expectError:       true,
			expectedErrorCode: cerror.InvalidField,
		},
		{
			name: "Missing email",
			account: &models.Account{
				DisplayName:  "AliceMissingEmail",
				Username:     "aliceMissingEmail",
				Email:        "", // Invalid
				PasswordHash: "hashMissing",
			},
			expectError:       true,
			expectedErrorCode: cerror.InvalidField,
		},
		{
			name: "Missing password hash",
			account: &models.Account{
				DisplayName:  "AliceMissingPassword",
				Username:     "aliceMissingPass",
				Email:        "alice_missing_pass@example.com",
				PasswordHash: "", // Invalid
			},
			expectError:       true,
			expectedErrorCode: cerror.InvalidField,
		},
		{
			name: "Missing display name",
			account: &models.Account{
				Username:     "aliceMissingDisplay",
				Email:        "alice_missing_display@example.com",
				PasswordHash: "hashMissing",
			},
			expectError:       true,
			expectedErrorCode: cerror.InvalidField,
		},
	}

	// Execute test cases
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			createdAt := time.Now()
			updatedAt := time.Now()
			tt.account.CreatedAt = &createdAt
			tt.account.UpdatedAt = &updatedAt

			createdAccount, err := globalRepo.AddAccount(context.Background(), tt.account)

			if tt.expectError {
				assert.NotNil(t, err, "Expected an error")
				assert.Equal(t, tt.expectedErrorCode, err.GetCode(), "Unexpected error code")
			} else {
				assert.Nil(t, err, "Did not expect an error")
				assert.NotNil(t, createdAccount, "Created account should not be nil")
				assert.Equal(t, tt.account.DisplayName, createdAccount.DisplayName)
				assert.Equal(t, tt.account.Username, createdAccount.Username)
				assert.Equal(t, tt.account.Email, createdAccount.Email)
				assert.Equal(t, tt.account.PasswordHash, createdAccount.PasswordHash)
				assert.NotEqual(t, "00000000-0000-0000-0000-000000000000", createdAccount.ID.String(), "ID should be valid")

				// Compare timestamps within an acceptable range
				assert.WithinDuration(t, *tt.account.CreatedAt, *createdAccount.CreatedAt, time.Second)
				assert.WithinDuration(t, *tt.account.UpdatedAt, *createdAccount.UpdatedAt, time.Second)
			}
		})
	}
}

func ptrTime(t time.Time) *time.Time {
	return &t
}

func TestUpdateAccount(t *testing.T) {
	// Insert an initial account
	createdAt := time.Now()
	updatedAt := time.Now()
	original := &models.Account{
		ID:           uuid.New(),
		DisplayName:  "Ben",
		Username:     "Ben123",
		Email:        "Ben@example.com",
		PasswordHash: "BenHash1",
		CreatedAt:    &createdAt,
		UpdatedAt:    &updatedAt,
	}

	insertQuery := `
		INSERT INTO account (id, display_name, username, email, password_hash, created_at, updated_at)
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
		t.Fatalf("Failed to insert initial account: %v", err)
	}

	validation := "validated"
	tests := []struct {
		name              string
		updateAccount     *models.Account
		expectError       bool
		expectedErrorCode string
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
				ID:           uuid.New(), // Non-existent ID
				DisplayName:  "No One",
				Username:     "noone",
				Email:        "noone@example.com",
				PasswordHash: "hash",
				Validation:   &validation,
				UpdatedAt:    ptrTime(time.Now()),
			},
			expectError:       true,
			expectedErrorCode: cerror.ResourceNotFound, // should not be found
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			updated, err := globalRepo.UpdateAccount(context.Background(), tt.updateAccount)

			if tt.expectError {
				assert.NotNil(t, err, "Expected an error")
				assert.Equal(t, tt.expectedErrorCode, err.GetCode(), "Unexpected error code")
			} else {
				assert.Nil(t, err, "Did not expect an error")
				assert.NotNil(t, updated, "Updated account should not be nil")
				assert.Equal(t, tt.updateAccount.DisplayName, updated.DisplayName)
				assert.Equal(t, tt.updateAccount.Username, updated.Username)
				assert.Equal(t, tt.updateAccount.Email, updated.Email)
				assert.Equal(t, tt.updateAccount.PasswordHash, updated.PasswordHash)
				assert.Equal(t, tt.updateAccount.Validation, updated.Validation)

				// Compare UpdatedAt within a reasonable range
				assert.WithinDuration(t, *tt.updateAccount.UpdatedAt, *updated.UpdatedAt, time.Second)
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
		INSERT INTO account (id, display_name, username, email, password_hash, created_at, updated_at)
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
		cerr := globalRepo.DeleteAccount(context.Background(), original.ID)
		assert.Nil(t, cerr, "expected no error when deleting an existing account")

		var count int
		err := globalDB.GetContext(context.Background(), &count, "SELECT COUNT(*) FROM account WHERE id = $1", original.ID)
		assert.NoError(t, err, "failed to count rows")
		assert.Equal(t, 0, count, "account should be deleted")
	})

	t.Run("delete non-existing account", func(t *testing.T) {
		nonExistentID := uuid.New()
		err := globalRepo.DeleteAccount(context.Background(), nonExistentID)
		assert.Nil(t, err, "expected no error when deleting a non-existent account")
	})
}

func TestGetAccountByEmail(t *testing.T) {
	// create user
	createdAt := time.Now()
	updatedAt := time.Now()
	original := &models.Account{
		ID:           uuid.New(),
		DisplayName:  "Duncan",
		Username:     "duncan123",
		Email:        "duncan@example.com",
		PasswordHash: "duncanHash1",
		CreatedAt:    &createdAt,
		UpdatedAt:    &updatedAt,
	}

	insertQuery := `
		INSERT INTO account (id, display_name, username, email, password_hash, created_at, updated_at)
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

	// Insert an SSH key associated with this account.
	sshKeyID := uuid.New()
	keyValue := "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQC..."
	sshCreatedAt := time.Now()
	sshUpdatedAt := time.Now()

	insertKeyQuery := `
		INSERT INTO ssh_key (id, account_id, public_key_string, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
	`
	_, err = globalDB.ExecContext(context.Background(), insertKeyQuery,
		sshKeyID,
		original.ID,
		keyValue,
		&sshCreatedAt,
		&sshUpdatedAt,
	)
	if err != nil {
		t.Fatalf("failed to insert SSH key: %v", err)
	}

	t.Run("get account by Email no associations", func(t *testing.T) {
		acc, cerr := globalRepo.GetAccountByEmail(context.Background(), original.Email, []string{})
		assert.Nil(t, cerr, "expected no error when getting account without associations")
		assert.Equal(t, original.ID, acc.ID)
		assert.Equal(t, original.Email, acc.Email)
		// Without associations, SSHKeys should be nil or empty.
		assert.True(t, len(acc.SSHKeys) == 0, "expected no SSHKeys without association")
	})

	t.Run("get account by Email with SSHKey association", func(t *testing.T) {
		acc, cerr := globalRepo.GetAccountByEmail(context.Background(), original.Email, []string{models.SSHKEY})
		assert.Nil(t, cerr, "expected no error when getting account with SSHKey association")
		assert.Equal(t, original.ID, acc.ID)
		assert.Equal(t, original.Email, acc.Email)
		// Now we expect SSHKeys to be present.
		assert.NotNil(t, acc.SSHKeys, "expected SSHKeys to be fetched")
		assert.Greater(t, len(acc.SSHKeys), 0, "expected at least one SSHKey")
		// Optionally check that our inserted key is in the list.
		found := false
		for _, k := range acc.SSHKeys {
			if k.ID == sshKeyID {
				found = true
				break
			}
		}
		assert.True(t, found, "expected inserted SSHKey to be found")
	})

	t.Run("get account by Email with ALL associations", func(t *testing.T) {
		acc, cerr := globalRepo.GetAccountByEmail(context.Background(), original.Email, []string{models.ALL})
		assert.Nil(t, cerr, "expected no error when getting account with ALL associations")
		assert.Equal(t, original.ID, acc.ID)
		assert.Equal(t, original.Email, acc.Email)
		// Expect associations to be present.
		assert.NotNil(t, acc.SSHKeys, "expected SSHKeys to be fetched")
		assert.Greater(t, len(acc.SSHKeys), 0, "expected at least one SSHKey")
	})
}

func TestGetAccountByID(t *testing.T) {
	// Create and insert an account.
	createdAt := time.Now()
	updatedAt := time.Now()
	original := &models.Account{
		ID:           uuid.New(),
		DisplayName:  "TestUser",
		Username:     "testuser",
		Email:        "testuser@example.com",
		PasswordHash: "hash1",
		CreatedAt:    &createdAt,
		UpdatedAt:    &updatedAt,
	}

	insertQuery := `
		INSERT INTO account (id, display_name, username, email, password_hash, created_at, updated_at)
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
	t.Logf("Inserted account: %+v", original)

	// Sub-test 1: Retrieve the account by ID without associations.
	t.Run("Get account by ID without associations", func(t *testing.T) {
		account, cerr := globalRepo.GetAccountByID(context.Background(), original.ID, []string{})
		assert.Nil(t, cerr, "expected no error retrieving account")
		assert.NotNil(t, account, "expected an account to be returned")
		assert.Equal(t, original.ID, account.ID)
		assert.Equal(t, original.Email, account.Email)
		// Without associations, SSHKeys should be nil or empty.
		assert.True(t, len(account.SSHKeys) == 0, "expected no SSHKeys without association")
	})

	// Insert an SSH key associated with this account.
	sshKeyID := uuid.New()
	keyValue := "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQC..."
	sshCreatedAt := time.Now()
	sshUpdatedAt := time.Now()
	insertKeyQuery := `
		INSERT INTO ssh_key (id, account_id, public_key_string, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
	`
	_, err = globalDB.ExecContext(context.Background(), insertKeyQuery,
		sshKeyID,
		original.ID,
		keyValue,
		&sshCreatedAt,
		&sshUpdatedAt,
	)
	if err != nil {
		t.Fatalf("failed to insert SSH key: %v", err)
	}

	// Sub-test 2: Retrieve the account by ID with the SSH key association.
	t.Run("Get account by ID with SSHKey association", func(t *testing.T) {
		account, cerr := globalRepo.GetAccountByID(context.Background(), original.ID, []string{models.SSHKEY})
		assert.Nil(t, cerr, "expected no error retrieving account with SSHKey association")
		assert.NotNil(t, account, "expected an account to be returned")
		// Now we expect SSHKeys to be populated.
		assert.NotNil(t, account.SSHKeys, "expected SSHKeys to be fetched")
		assert.Greater(t, len(account.SSHKeys), 0, "expected at least one SSHKey")
		// Optionally, verify that the inserted key is found.
		found := false
		for _, k := range account.SSHKeys {
			if k.ID == sshKeyID {
				found = true
				break
			}
		}
		assert.True(t, found, "expected inserted SSHKey to be found")
	})
}

func TestGetAccountByUsername(t *testing.T) {
	// Create an initial account record with unique values.
	createdAt := time.Now()
	updatedAt := time.Now()
	original := &models.Account{
		ID:           uuid.New(),
		DisplayName:  "UniqueUserDisplay1",
		Username:     "uniqueuser1",
		Email:        "uniqueuser1@example.com",
		PasswordHash: "uniqueHash1",
		CreatedAt:    &createdAt,
		UpdatedAt:    &updatedAt,
	}

	insertQuery := `
		INSERT INTO account (id, display_name, username, email, password_hash, created_at, updated_at)
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

	sshKeyID := uuid.New()
	keyValue := "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQC..."
	sshCreatedAt := time.Now()
	sshUpdatedAt := time.Now()
	insertKeyQuery := `
		INSERT INTO ssh_key (id, account_id, public_key_string, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
	`
	_, err = globalDB.ExecContext(context.Background(), insertKeyQuery,
		sshKeyID,
		original.ID,
		keyValue,
		&sshCreatedAt,
		&sshUpdatedAt,
	)
	if err != nil {
		t.Fatalf("failed to insert SSH key: %v", err)
	}

	t.Run("get account by Username no associations", func(t *testing.T) {
		acc, cerr := globalRepo.GetAccountByUsername(context.Background(), original.Username, []string{})
		assert.Nil(t, cerr, "expected no error when getting account without associations")
		assert.Equal(t, original.ID, acc.ID)
		assert.Equal(t, original.Email, acc.Email)
		// Without associations, expect SSHKeys (and others) to be nil or empty.
		assert.True(t, len(acc.SSHKeys) == 0, "expected no SSHKeys without association")
	})

	t.Run("get account by Username with ALL associations", func(t *testing.T) {
		acc, cerr := globalRepo.GetAccountByUsername(context.Background(), original.Username, []string{models.ALL})
		assert.Nil(t, cerr, "expected no error when getting account with ALL associations")
		assert.Equal(t, original.ID, acc.ID)
		assert.Equal(t, original.Email, acc.Email)
		// For ALL associations, we expect SSHKeys and others (if implemented) to be present.
		assert.NotNil(t, acc.SSHKeys, "expected SSHKeys to be fetched")
		// You may add further checks here if your associations include more data.
		found := false
		for _, k := range acc.SSHKeys {
			if k.ID == sshKeyID {
				found = true
				break
			}
		}
		assert.True(t, found, "expected inserted SSHKey to be found")
	})
}

func TestAddKeyToAccountByEmail(t *testing.T) {
	// Create an initial account record.
	createdAt := time.Now()
	updatedAt := time.Now()
	account := &models.Account{
		ID:           uuid.New(),
		DisplayName:  "TestUserUnique",
		Username:     "testuser_unique",
		Email:        "testuser_unique@example.com",
		PasswordHash: "uniqueHash1",
		CreatedAt:    &createdAt,
		UpdatedAt:    &updatedAt,
	}

	insertAccountQuery := `
		INSERT INTO account (id, display_name, username, email, password_hash, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := globalDB.ExecContext(context.Background(), insertAccountQuery,
		account.ID,
		account.DisplayName,
		account.Username,
		account.Email,
		account.PasswordHash,
		account.CreatedAt,
		account.UpdatedAt,
	)
	if err != nil {
		t.Fatalf("failed to insert initial account: %v", err)
	}
	t.Logf("Inserted initial account: %+v", account)

	t.Run("successful add key", func(t *testing.T) {
		// Use unique values for the key.
		name := "SSHKeyUnique1"
		sha3384 := "dummySHA3384Unique"
		encodedPublicKey := "dummyEncodedPublicKeyUnique"
		key, cerr := globalRepo.AddKeyToAccountByEmail(context.Background(), name, sha3384, encodedPublicKey, account.Email)
		assert.Nil(t, cerr, "expected no error when adding key")
		assert.NotNil(t, key, "expected key to be returned")
		assert.Equal(t, name, key.Name)
		assert.Equal(t, sha3384, key.SHA3384)
		assert.Equal(t, encodedPublicKey, key.EncodedPublicKey)
		assert.Equal(t, account.ID, key.AccountID)
		// Check that key.Until is roughly one year in the future.
		expectedUntil := time.Now().AddDate(1, 0, 0)
		// Allow a small delta due to execution time.
		assert.NotNil(t, key.Until, "expected key.Until to be set")
		delta := expectedUntil.Sub(*key.Until)
		if delta < 0 {
			delta = -delta
		}
		assert.Less(t, delta, time.Minute, "key.Until should be approximately one year from now")
	})

	t.Run("account not found", func(t *testing.T) {
		// Use an email that doesn't exist.
		key, cerr := globalRepo.AddKeyToAccountByEmail(context.Background(), "AnotherKey", "shaDummy", "encodedDummy", "nonexistent@example.com")
		assert.NotNil(t, cerr, "expected error when account not found")
		assert.Nil(t, key, "expected key to be nil when error occurs")
		if cerr != nil {
			// Directly check the custom error code.
			assert.Equal(t, cerror.ResourceNotFound, cerr.Code)
		}
	})
}

func TestGetKeyBySHA3384(t *testing.T) {
	// First, insert an account so that the key can reference it.
	createdAt := time.Now()
	updatedAt := time.Now()
	account := &models.Account{
		ID:           uuid.New(),
		DisplayName:  "KeyTestUser",
		Username:     "keytestuser",
		Email:        "keytestuser@example.com",
		PasswordHash: "someHashValue",
		CreatedAt:    &createdAt,
		UpdatedAt:    &updatedAt,
	}

	insertAccountQuery := `
		INSERT INTO account (id, display_name, username, email, password_hash, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := globalDB.ExecContext(context.Background(), insertAccountQuery,
		account.ID,
		account.DisplayName,
		account.Username,
		account.Email,
		account.PasswordHash,
		account.CreatedAt,
		account.UpdatedAt,
	)
	if err != nil {
		t.Fatalf("failed to insert account: %v", err)
	}
	t.Logf("Inserted account: %+v", account)

	// Now, insert an SSH key associated with the inserted account.
	createdAtKey := time.Now()
	updatedAtKey := time.Now()
	until := time.Now().AddDate(1, 0, 0)
	keyID := uuid.New().String()
	name := "UniqueSSHKeyForTest"
	sha3384 := "uniqueSHA3384Test"
	encodedPublicKey := "uniqueEncodedPublicKeyTest"

	insertKeyQuery := `
		INSERT INTO key (id, name, sha3384, encoded_public_key, account_id, until, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err = globalDB.ExecContext(context.Background(), insertKeyQuery,
		keyID,
		name,
		sha3384,
		encodedPublicKey,
		account.ID,
		&until,
		&createdAtKey,
		&updatedAtKey,
	)
	if err != nil {
		t.Fatalf("failed to insert SSH key: %v", err)
	}
	t.Logf("Inserted SSH key with SHA3384: %s", sha3384)

	t.Run("get key by valid SHA3384", func(t *testing.T) {
		key, cerr := globalRepo.GetKeyBySHA3384(context.Background(), sha3384)
		assert.Nil(t, cerr, "expected no error when getting key by valid SHA3384")
		assert.NotNil(t, key, "expected key to be returned")
		assert.Equal(t, name, key.Name)
		assert.Equal(t, sha3384, key.SHA3384)
		assert.Equal(t, encodedPublicKey, key.EncodedPublicKey)
		// Optionally check that AccountID matches.
		assert.Equal(t, account.ID, key.AccountID)
		assert.NotNil(t, key.Until, "expected key.Until to be set")
	})

	t.Run("get key by non-existent SHA3384", func(t *testing.T) {
		nonExistentSHA := "nonexistentSHAValue"
		key, cerr := globalRepo.GetKeyBySHA3384(context.Background(), nonExistentSHA)
		assert.NotNil(t, cerr, "expected an error when key not found")
		assert.Nil(t, key, "expected key to be nil when not found")
		// Check that the custom error code is ResourceNotFound.
		assert.Equal(t, cerror.ResourceNotFound, cerr.Code)
	})
}

func TestGetKeysByAccountID(t *testing.T) {
	// Insert an account.
	createdAt := time.Now()
	updatedAt := time.Now()
	account := &models.Account{
		ID:           uuid.New(),
		DisplayName:  "TestAccountForKeys",
		Username:     "testkeysuser",
		Email:        "testkeysuser@example.com",
		PasswordHash: "testhash",
		CreatedAt:    &createdAt,
		UpdatedAt:    &updatedAt,
	}

	insertAccountQuery := `
		INSERT INTO account (id, display_name, username, email, password_hash, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := globalDB.ExecContext(context.Background(), insertAccountQuery,
		account.ID,
		account.DisplayName,
		account.Username,
		account.Email,
		account.PasswordHash,
		account.CreatedAt,
		account.UpdatedAt,
	)
	if err != nil {
		t.Fatalf("failed to insert account: %v", err)
	}
	t.Logf("Inserted account: %+v", account)

	// Insert two key for that account.
	keyCreatedAt := time.Now()
	keyUpdatedAt := time.Now()
	until := time.Now().AddDate(1, 0, 0)
	key1ID := uuid.New()
	key2ID := uuid.New()
	insertKeyQuery := `
		INSERT INTO key (id, name, sha3384, encoded_public_key, account_id, until, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err = globalDB.ExecContext(context.Background(), insertKeyQuery,
		key1ID,
		"KeyOne",
		"shaKeyOneUnique",
		"encodedKeyOneUnique",
		account.ID,
		&until,
		&keyCreatedAt,
		&keyUpdatedAt,
	)
	if err != nil {
		t.Fatalf("failed to insert key1: %v", err)
	}

	_, err = globalDB.ExecContext(context.Background(), insertKeyQuery,
		key2ID,
		"KeyTwo",
		"shaKeyTwoUnique",
		"encodedKeyTwoUnique",
		account.ID,
		&until,
		&keyCreatedAt,
		&keyUpdatedAt,
	)
	if err != nil {
		t.Fatalf("failed to insert key2: %v", err)
	}

	// Test case: key found for the account.
	t.Run("key found", func(t *testing.T) {
		key, cerr := globalRepo.GetKeysByAccountID(context.Background(), account.ID)
		assert.Nil(t, cerr, "expected no error when key exist for account")
		assert.NotNil(t, key, "expected key to be returned")
		assert.Greater(t, len(key), 0, "expected at least one key")
		// Check that both inserted key are found.
		foundKeyOne := false
		foundKeyTwo := false
		for _, key := range key {
			if key.ID == key1ID {
				foundKeyOne = true
			}
			if key.ID == key2ID {
				foundKeyTwo = true
			}
		}
		assert.True(t, foundKeyOne, "expected key1 to be found")
		assert.True(t, foundKeyTwo, "expected key2 to be found")
	})

	// Test case: no key found for a non-existent account.
	t.Run("no key found", func(t *testing.T) {
		nonExistentID := uuid.New() // new account ID that doesn't exist
		key, cerr := globalRepo.GetKeysByAccountID(context.Background(), nonExistentID)
		assert.NotNil(t, cerr, "expected an error when no key exist for account")
		assert.Nil(t, key, "expected key to be nil when not found")
		// Check that the error code is ResourceNotFound.
		assert.Equal(t, cerror.ResourceNotFound, cerr.Code)
	})
}

func TestFilterKeys(t *testing.T) {
	// --- Setup: Insert an account and a key record ---
	// Insert an account.
	accCreatedAt := time.Now()
	accUpdatedAt := time.Now()
	account := &models.Account{
		ID:           uuid.New(), // using uuid.UUID type
		DisplayName:  "FilterTestAccount",
		Username:     "filter_test",
		Email:        "filter_test@example.com",
		PasswordHash: "filterHash",
		CreatedAt:    &accCreatedAt,
		UpdatedAt:    &accUpdatedAt,
	}
	insertAccountQuery := `
		INSERT INTO account (id, display_name, username, email, password_hash, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := globalDB.ExecContext(context.Background(), insertAccountQuery,
		account.ID,
		account.DisplayName,
		account.Username,
		account.Email,
		account.PasswordHash,
		account.CreatedAt,
		account.UpdatedAt,
	)
	if err != nil {
		t.Fatalf("failed to insert account: %v", err)
	}
	t.Logf("Inserted account: %+v", account)

	// Insert a key.
	keyCreatedAt := time.Now()
	keyUpdatedAt := time.Now()
	until := time.Now().AddDate(1, 0, 0)
	keyRecord := &models.Key{
		ID:               uuid.New(), // generate key id
		Name:             "FilterTestKey",
		SHA3384:          "FilterTestSHA",
		EncodedPublicKey: "FilterTestEncodedKey",
		AccountID:        account.ID,
		Until:            &until,
		CreatedAt:        &keyCreatedAt,
		UpdatedAt:        &keyUpdatedAt,
		// DeletedAt remains nil.
	}

	insertKeyQuery := `
		INSERT INTO key (id, name, sha3384, encoded_public_key, account_id, until, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err = globalDB.ExecContext(context.Background(), insertKeyQuery,
		keyRecord.ID,
		keyRecord.Name,
		keyRecord.SHA3384,
		keyRecord.EncodedPublicKey,
		keyRecord.AccountID,
		keyRecord.Until,
		keyRecord.CreatedAt,
		keyRecord.UpdatedAt,
	)
	if err != nil {
		t.Fatalf("failed to insert key: %v", err)
	}
	t.Logf("Inserted key: %+v", keyRecord)

	// Insert a deleted key
	deletedAt := time.Now().Add(-30 * time.Minute) // Ensure unique timestamp
	createdAt := time.Now().Add(-2 * time.Hour)    // Ensure unique timestamp
	updatedAt := time.Now().Add(-time.Hour)
	untilDeleted := time.Now().Add(24 * time.Hour) // Future expiration for the key

	deletedKey := &models.Key{
		ID:               uuid.New(),
		Name:             "DeletedTestKey",
		SHA3384:          "uniquehash1234567890abcdef", // Ensure a unique SHA3384
		EncodedPublicKey: "public-key-encoded-string",
		AccountID:        account.ID, // Link to an existing account
		Until:            &untilDeleted,
		CreatedAt:        &createdAt,
		UpdatedAt:        &updatedAt,
		DeletedAt:        &deletedAt,
	}

	insertQuery := `
	INSERT INTO key (id, name, sha3384, encoded_public_key, account_id, until, created_at, updated_at, deleted_at)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	_, err = globalDB.ExecContext(context.Background(), insertQuery,
		deletedKey.ID,
		deletedKey.Name,
		deletedKey.SHA3384,
		deletedKey.EncodedPublicKey,
		deletedKey.AccountID,
		deletedKey.Until,
		deletedKey.CreatedAt,
		deletedKey.UpdatedAt,
		deletedKey.DeletedAt,
	)
	if err != nil {
		t.Fatalf("failed to insert deleted key: %v", err)
	}
	t.Logf("Inserted deleted key: %+v", deletedKey)

	// --- Table-driven tests for FilterKeys ---
	tests := []struct {
		name          string
		filter        models.Key
		expectSuccess bool
	}{
		{
			name: "Filter by ID",
			filter: models.Key{
				ID: keyRecord.ID,
			},
			expectSuccess: true,
		},
		{
			name: "Filter by Name",
			filter: models.Key{
				Name: keyRecord.Name,
			},
			expectSuccess: true,
		},
		{
			name: "Filter by SHA3384",
			filter: models.Key{
				SHA3384: keyRecord.SHA3384,
			},
			expectSuccess: true,
		},
		{
			name: "Filter by EncodedPublicKey",
			filter: models.Key{
				EncodedPublicKey: keyRecord.EncodedPublicKey,
			},
			expectSuccess: true,
		},
		{
			name: "Filter by AccountID",
			filter: models.Key{
				AccountID: keyRecord.AccountID,
			},
			expectSuccess: true,
		},
		{
			name: "Filter by Until",
			filter: models.Key{
				Until: keyRecord.Until,
			},
			expectSuccess: true,
		},
		{
			name: "Filter by CreatedAt",
			filter: models.Key{
				CreatedAt: keyRecord.CreatedAt,
			},
			expectSuccess: true,
		},
		{
			name: "Filter by UpdatedAt",
			filter: models.Key{
				UpdatedAt: keyRecord.UpdatedAt,
			},
			expectSuccess: true,
		},
		{
			name: "Filter with no matching criteria",
			filter: models.Key{
				Name: "nonexistent",
			},
			expectSuccess: false,
		}, {
			name: "Filter by DeletedAt (should return deleted key)",
			filter: models.Key{
				DeletedAt: deletedKey.DeletedAt,
			},
			expectSuccess: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Call FilterKeys with the filter and takeFirst=true.
			result, cerr := globalRepo.FilterKeys(context.Background(), &tt.filter, true)

			if tt.expectSuccess {
				assert.Nil(t, cerr, "expected no error for successful filter")
				assert.NotNil(t, result, "expected a key result")

				// Ensure we compare against the correct key (deletedKey or keyRecord)
				expectedKey := keyRecord
				if tt.name == "Filter by DeletedAt (should return deleted key)" {
					expectedKey = deletedKey
				}

				assert.Equal(t, expectedKey.ID, result.ID, "Expected key ID to match")
				assert.Equal(t, expectedKey.Name, result.Name, "Expected key Name to match")
				assert.Equal(t, expectedKey.SHA3384, result.SHA3384, "Expected key SHA3384 to match")
				assert.Equal(t, expectedKey.EncodedPublicKey, result.EncodedPublicKey, "Expected key EncodedPublicKey to match")

			} else {
				assert.NotNil(t, cerr, "expected an error when no key matches")
				assert.Nil(t, result, "expected nil result when no key matches")
			}
		})
	}
}

func TestFilterAccounts(t *testing.T) {
	// Insert a known account record.
	createdAt := time.Now().Add(-2 * time.Hour) // Ensure unique timestamp
	updatedAt := time.Now().Add(-time.Hour)
	validation := "verified"
	account := &models.Account{
		ID:           uuid.New(),
		DisplayName:  "UniqueTestAccount",
		Username:     "uniquetestuser",
		Email:        "uniquetest@example.com",
		PasswordHash: "securehash456",
		CreatedAt:    &createdAt,
		UpdatedAt:    &updatedAt,
		Validation:   &validation,
	}
	insertQuery := `
		INSERT INTO account (id, display_name, username, email, password_hash, created_at, updated_at, validation)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err := globalDB.ExecContext(context.Background(), insertQuery,
		account.ID,
		account.DisplayName,
		account.Username,
		account.Email,
		account.PasswordHash,
		account.CreatedAt,
		account.UpdatedAt,
		account.Validation,
	)
	if err != nil {
		t.Fatalf("failed to insert account: %v", err)
	}
	t.Logf("Inserted account: %+v", account)

	// Insert a deleted account
	deletedAt := time.Now().Add(-30 * time.Minute) // Ensure unique timestamp
	deletedAccount := &models.Account{
		ID:           uuid.New(),
		DisplayName:  "DeletedTestAccount",
		Username:     "deleteduser",
		Email:        "deleted@example.com",
		PasswordHash: "securehash789",
		CreatedAt:    &createdAt,
		UpdatedAt:    &updatedAt,
		DeletedAt:    &deletedAt,
		Validation:   &validation,
	}
	insertQuery = `
		INSERT INTO account (id, display_name, username, email, password_hash, created_at, updated_at, deleted_at, validation)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	_, err = globalDB.ExecContext(context.Background(), insertQuery,
		deletedAccount.ID,
		deletedAccount.DisplayName,
		deletedAccount.Username,
		deletedAccount.Email,
		deletedAccount.PasswordHash,
		deletedAccount.CreatedAt,
		deletedAccount.UpdatedAt,
		deletedAccount.DeletedAt,
		deletedAccount.Validation,
	)
	if err != nil {
		t.Fatalf("failed to insert deleted account: %v", err)
	}
	t.Logf("Inserted deleted account: %+v", deletedAccount)

	// Define table-driven tests.
	tests := []struct {
		name          string
		filter        models.Account
		expectSuccess bool
	}{
		{
			name: "Filter by ID",
			filter: models.Account{
				ID: account.ID,
			},
			expectSuccess: true,
		},
		{
			name: "Filter by DisplayName",
			filter: models.Account{
				DisplayName: account.DisplayName,
			},
			expectSuccess: true,
		},
		{
			name: "Filter by Username",
			filter: models.Account{
				Username: account.Username,
			},
			expectSuccess: true,
		},
		{
			name: "Filter by Email",
			filter: models.Account{
				Email: account.Email,
			},
			expectSuccess: true,
		},
		{
			name: "Filter by Validation",
			filter: models.Account{
				Validation: account.Validation,
			},
			expectSuccess: true,
		},
		{
			name: "Filter by non-existent display name",
			filter: models.Account{
				DisplayName: "NoMatchAccount",
			},
			expectSuccess: false,
		},
		{
			name: "Filter by non-existent username",
			filter: models.Account{
				Username: "randomuser123",
			},
			expectSuccess: false,
		},
		{
			name: "Filter by CreatedAt",
			filter: models.Account{
				CreatedAt: account.CreatedAt,
			},
			expectSuccess: true,
		},
		{
			name: "Filter by UpdatedAt",
			filter: models.Account{
				UpdatedAt: account.UpdatedAt,
			},
			expectSuccess: true,
		},
		{
			name: "Filter by DeletedAt (should return deleted account)",
			filter: models.Account{
				DeletedAt: deletedAccount.DeletedAt,
			},
			expectSuccess: true,
		}, {
			name: "Filter with no matching criteria",
			filter: models.Account{
				DisplayName: "nonexistent",
			},
			expectSuccess: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, cerr := globalRepo.FilterAccounts(context.Background(), &tt.filter, true)
			if tt.expectSuccess {
				assert.Nil(t, cerr, "expected no error for successful filter")
				assert.NotNil(t, results, "expected an account result")
				assert.Greater(t, len(results), 0, "expected at least one matching result")
			} else {
				assert.NotNil(t, cerr, "expected an error when no account matches")
				assert.Nil(t, results, "expected nil result when no account matches")
				assert.Equal(t, cerror.ResourceNotFound, cerr.Code)
			}
		})
	}
}
