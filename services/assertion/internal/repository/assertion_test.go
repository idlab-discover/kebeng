package repository_test

import (
	"io"
	"os"
	"testing"
	"time"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	"github.com/google/uuid"
	"github.com/idlab-discover/kebeng/common/cerror"
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

func TestAddAccountKeyAssertion(t *testing.T) {
	tests := []struct {
		name        string
		authorityID string
		publicKey   string
		signKey     string
		displayName string
		revision    uint32
		accountID   uuid.UUID
		since       time.Time
		until       time.Time
		body        []byte
		bodyLength  uint64
		Signature   string
		expectError bool
		errorCode   string // expected error code (if any)
	}{
		{
			name:        "Successful Account-Key Assertion Insertion",
			authorityID: "canonical",
			// This is the SHA3-384 of the account's public key.
			publicKey: "BWDEoaqyr25nF5SNCvEv2v7QnM9QsfCc0PBMYD_i2NGSQ32EF2d4D0hqUel3m8ul",
			// This is the SHA3-384 of the signing key.
			signKey:     "-CvQKAwRQ5h3Ffn10FILJoEZUXOv6km9FwA80-Rcj-f-6jadQ89VRswHNiEB9Lxk",
			displayName: "store",
			revision:    2,
			accountID:   uuid.New(),
			since:       time.Date(2016, 4, 1, 0, 0, 0, 0, time.UTC),
			until:       time.Date(2016, 4, 1, 0, 0, 0, 0, time.UTC),
			body:        []byte("test-body-data"),
			bodyLength:  717,
			Signature:   "AcLBtest-signature-data",
			expectError: false,
		},
		{
			name:        "Fail Insertion on Duplicate",
			authorityID: "canonical",
			publicKey:   "BWDEoaqyr25nF5SNCvEv2v7QnM9QsfCc0PBMYD_i2NGSQ32EF2d4D0hqUel3m8ul",
			signKey:     "", // set to empty string to trigger error
			displayName: "store",
			revision:    2,
			accountID:   uuid.New(),
			since:       time.Date(2016, 4, 1, 0, 0, 0, 0, time.UTC),
			until:       time.Date(2016, 4, 1, 0, 0, 0, 0, time.UTC),
			body:        []byte("test-body-data"),
			bodyLength:  717,
			Signature:   "AcLBtest-signature-data",
			expectError: true,
			errorCode:   cerror.InvalidField,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			el := cerror.NewErrorList()
			record, cerr := globalRepo.AddAccountKeyAssertion(el, tt.authorityID, tt.publicKey, tt.signKey, tt.displayName, tt.revision, tt.accountID, tt.since, tt.until, tt.body, tt.bodyLength, tt.Signature)
			if tt.expectError {
				assert.NotNil(t, cerr, "Expected an error during insertion")
				assert.Equal(t, el.HasError(), true, "Expected an error in the list")
				if cerr != nil {
					assert.Equal(t, tt.errorCode, cerr.GetCode())
				}
			} else {
				assert.Nil(t, cerr, "Did not expect an error during insertion")
				assert.Equal(t, el.HasError(), false, "Expected no errors in the list")
				assert.NotNil(t, record, "Expected a non-nil assertion record")
				assert.NotEqual(t, uuid.Nil, record.ID, "Expected a valid UUID for the record ID")
			}
		})
	}
}

func TestGetAccountKeyAssertionByAccountName(t *testing.T) {
	tests := []struct {
		name          string
		assertionName string
		preInsert     bool // if true, insert a record into the DB for this accountID before retrieval
		expectError   bool
		errorCode     string // expected error code (if any)
	}{
		{
			name:          "Successful Get Account-Key Assertion",
			assertionName: "KeyAccountAssertion",

			preInsert:   true,
			expectError: false,
		},
		{
			name:          "Fail Get Account-Key Assertion for Nonexistent Account",
			assertionName: "NonExistentAccountAssertion",
			preInsert:     false,
			expectError:   true,
			errorCode:     cerror.ResourceNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// If preInsert is true, insert a record with this accountID.
			if tt.preInsert {
				testID := uuid.New() // ID for the record to be inserted.
				insertQuery := `
				INSERT INTO account_key_assertion (
					id, authority_id, public_key_sha3_384, sign_key_sha3_384,
					name, revision, account_id, since, until, body, body_length, signature
				) VALUES (
					$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
				)`
				_, err := globalDB.Exec(insertQuery,
					testID,
					"canonical",
					"BWDEoaqyr25nF5SNCvEv2v7QnM9QsfCc0PBMYD_i2NGSQ32EF2d4D0hqUel3m8ul",
					"-CvQKAwRQ5h3Ffn10FILJoEZUXOv6km9FwA80-Rcj-f-6jadQ89VRswHNiEB9Lxk",
					tt.assertionName,
					2,
					uuid.New(),
					time.Date(2016, 4, 1, 0, 0, 0, 0, time.UTC),
					time.Date(2016, 4, 1, 0, 0, 0, 0, time.UTC),
					[]byte("test-body-data"),
					717,
					"AcLBtest-signature-data",
				)
				assert.NoError(t, err, "Failed to insert test account key assertion")
			}

			// Now, try to retrieve the assertion by accountID.
			el := cerror.NewErrorList()
			record, cerr := globalRepo.GetAccountKeyAssertionByName(el, tt.assertionName)

			if tt.expectError {
				assert.NotNil(t, cerr, "Expected an error for non-existent account")
				assert.True(t, el.HasError(), "Expected error list to contain errors")
				if cerr != nil {
					assert.Equal(t, tt.errorCode, cerr.GetCode(), "Error code should match expected")
				}
			} else {
				assert.Nil(t, cerr, "Did not expect an error for existing account")
				assert.False(t, el.HasError(), "Expected no errors in the error list")
				assert.NotNil(t, record, "Expected a non-nil assertion record")
				assert.Equal(t, tt.assertionName, record.Name, "Account ID should match")
				assert.Equal(t, "canonical", record.AuthorityID, "AuthorityID should match")
				assert.Equal(t, uint32(2), record.SnapRevisionSequenceNumber, "Revision should match")
			}
		})
	}
}

func TestAddSnapRevisionAssertion(t *testing.T) {
	tests := []struct {
		name                       string
		authorityID                string
		snapSHA3_384               string
		developerID                uuid.UUID
		snapEntryID                uuid.UUID
		snapRevisionSequenceNumber uint32
		timestamp                  time.Time
		signKeySHA3_384            string
		snapSize                   uint64
		Signature                  string
		expectError                bool
		errorCode                  string // expected error code (if any)
	}{
		{
			name:                       "Successful SnapRevision Assertion Insertion",
			authorityID:                "canonical",
			snapSHA3_384:               "BWDEoaqyr25nF5SNCvEv2v7QnM9QsfCc0PBMYD_i2NGSQ32EF2d4D0hqUel3m8ul",
			developerID:                uuid.New(),
			snapEntryID:                uuid.New(),
			snapRevisionSequenceNumber: 2,
			timestamp:                  time.Date(2016, 4, 1, 0, 0, 0, 0, time.UTC),
			signKeySHA3_384:            "-CvQKAwRQ5h3Ffn10FILJoEZUXOv6km9FwA80-Rcj-f-6jadQ89VRswHNiEB9Lxk",
			snapSize:                   1234567,
			Signature:                  "AcLBtest-signature-data",
			expectError:                false,
		},
		{
			name:                       "Fail Insertion on Duplicate",
			authorityID:                "canonical",
			snapSHA3_384:               "BWDEoaqyr25nF5SNCvEv2v7QnM9QsfCc0PBMYD_i2NGSQ32EF2d4D0hqUel3m8ul",
			developerID:                uuid.New(),
			snapEntryID:                uuid.New(),
			snapRevisionSequenceNumber: 2,
			timestamp:                  time.Date(2016, 4, 1, 0, 0, 0, 0, time.UTC),
			signKeySHA3_384:            "", // set to empty string to trigger error
			snapSize:                   1234567,
			Signature:                  "AcLBtest-signature-data",
			expectError:                true,
			errorCode:                  cerror.InvalidField,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			el := cerror.NewErrorList()
			record, cerr := globalRepo.AddSnapRevisionAssertion(el, tt.authorityID, tt.snapSHA3_384, tt.signKeySHA3_384, tt.developerID, tt.snapEntryID, tt.snapRevisionSequenceNumber, tt.snapSize, tt.timestamp, tt.Signature)
			if tt.expectError {
				assert.NotNil(t, cerr, "Expected an error during insertion")
				assert.Equal(t, el.HasError(), true, "Expected an error in the list")
				if cerr != nil {
					assert.Equal(t, tt.errorCode, cerr.GetCode())
				}
			} else {
				assert.Nil(t, cerr, "Did not expect an error during insertion")
				assert.Equal(t, el.HasError(), false, "Expected no errors in the list")
				assert.NotNil(t, record, "Expected a non-nil assertion record")
				assert.NotEqual(t, uuid.Nil, record.ID, "Expected a valid UUID for the record ID")
			}
		})
	}
}

func TestGetSnapRevisionAssertionBySHA3_384(t *testing.T) {
	tests := []struct {
		name                               string
		snapEntryID                        uuid.UUID
		preInsert                          bool
		expectError                        bool
		errorCode                          string
		expectedAuthorityID                string
		expectedSnapSHA3_384               string
		signKeySHA3_384                    string
		expectedDeveloperID                uuid.UUID
		expectedSnapRevisionSequenceNumber uint32
		expectedTimestamp                  time.Time
		expectedSignature                  string
	}{
		{
			name:                               "Successful Get Snap Revision Assertion",
			snapEntryID:                        uuid.New(),
			preInsert:                          true,
			expectError:                        false,
			expectedAuthorityID:                "canonical",
			expectedSnapSHA3_384:               "test-snap-sha3-384",
			expectedSnapRevisionSequenceNumber: 2,
			signKeySHA3_384:                    "test-sign-key-sha3-384",
			expectedTimestamp:                  time.Date(2016, 4, 1, 0, 0, 0, 0, time.UTC),
			expectedSignature:                  "AcLBtest-signature-data",
		},
		{
			name:        "Fail Get Snap Revision Assertion for Nonexistent SnapEntry",
			snapEntryID: uuid.New(),
			preInsert:   false,
			expectError: true,
			errorCode:   cerror.ResourceNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.preInsert {

				devID := uuid.New()
				tt.expectedDeveloperID = devID

				insertQuery := `
					INSERT INTO snap_revision_assertion (
						id, authority_id,sign_key_sha3_384, snap_sha3_384, developer_id, snap_entry_id,
						snap_revision_sequence_number, timestamp, snap_size, signature
					) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
				`

				recordID := uuid.New()
				_, err := globalDB.Exec(insertQuery,
					recordID,
					tt.expectedAuthorityID,
					tt.signKeySHA3_384,
					tt.expectedSnapSHA3_384,
					devID,
					tt.snapEntryID,
					tt.expectedSnapRevisionSequenceNumber,
					tt.expectedTimestamp,
					1234567,
					tt.expectedSignature,
				)
				assert.NoError(t, err, "Failed to insert test snap revision assertion")
			}

			el := cerror.NewErrorList()
			record, cerr := globalRepo.GetSnapRevisionAssertionBySHA3_384(el, tt.expectedSnapSHA3_384)

			if tt.expectError {
				assert.NotNil(t, cerr, "Expected error for missing snap revision assertion")
				assert.True(t, el.HasError(), "Expected error list to contain errors")
				if cerr != nil {
					assert.Equal(t, tt.errorCode, cerr.GetCode(), "Error code should match expected")
				}
			} else {
				assert.Nil(t, cerr, "Did not expect an error for existing snap revision assertion")
				assert.False(t, el.HasError(), "Expected no errors in the error list")
				assert.NotNil(t, record, "Expected a non-nil snap revision assertion record")

				assert.Equal(t, tt.expectedAuthorityID, record.AuthorityID, "AuthorityID should match")
				assert.Equal(t, tt.expectedSnapSHA3_384, record.SnapSHA3_384, "SnapSHA3_384 should match")
				assert.Equal(t, tt.expectedDeveloperID, record.DeveloperID, "DeveloperID should match")
				assert.Equal(t, tt.expectedSnapRevisionSequenceNumber, record.SnapRevisionSequenceNumber, "Revision sequence number should match")
				assert.Equal(t, tt.expectedTimestamp, record.Timestamp.UTC(), "Timestamp should match")
			}
		})
	}
}
