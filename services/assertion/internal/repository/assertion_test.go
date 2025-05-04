package repository_test

import (
	"encoding/json"
	"io"
	"os"
	"testing"
	"time"

	"github.com/idlab-discover/kebeng/services/assertion/internal/config"
	assertionDB "github.com/idlab-discover/kebeng/services/assertion/internal/database"
	"github.com/idlab-discover/kebeng/services/assertion/internal/model"
	"github.com/idlab-discover/kebeng/services/assertion/internal/repository"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	"github.com/google/uuid"
	"github.com/idlab-discover/kebeng/common/cerror"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestGetAccountKeyAssertionByPublicKeySha(t *testing.T) {
	tests := []struct {
		name              string
		publicKeySha      string
		assertionName     string
		preInsert         bool // if true, insert before retrieval
		expectError       bool
		expectedErrorCode string
	}{
		{
			name:          "Successful Get Account-Key Assertion",
			publicKeySha:  "PUBKEY123",
			assertionName: "MyKeyAssertion",
			preInsert:     true,
			expectError:   false,
		},
		{
			name:              "Nonexistent Public Key",
			publicKeySha:      "NO_SUCH_KEY",
			assertionName:     "",
			preInsert:         false,
			expectError:       true,
			expectedErrorCode: cerror.ResourceNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// optional insert
			if tt.preInsert {
				recID := uuid.New()
				accountID := uuid.New()
				// revision=2 just to test non‑1
				_, err := globalDB.Exec(`
                    INSERT INTO account_key_assertion (
                      id, authority_id, public_key_sha3_384, sign_key_sha3_384,
                      name, revision, account_id, since, until, body, body_length, signature
                    ) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`,
					recID,
					"canonical",
					tt.publicKeySha,
					"STORE_SIGN_KEY_SHA3",
					tt.assertionName,
					2,
					accountID,
					time.Date(2025, 4, 21, 0, 0, 0, 0, time.UTC),
					time.Date(2026, 4, 21, 0, 0, 0, 0, time.UTC),
					[]byte("dummy-body"),
					1234,
					"DUMMY_SIGNATURE",
				)
				require.NoError(t, err, "setup insert")
			}

			el := cerror.NewErrorList()
			rec, cerr := globalRepo.GetAccountKeyAssertionByPublicKeySha(el, tt.publicKeySha)

			if tt.expectError {
				assert.NotNil(t, cerr, "expected error")
				assert.True(t, el.HasError(), "error list should be non‑empty")
				assert.Equal(t, tt.expectedErrorCode, cerr.GetCode())
				assert.Nil(t, rec, "should return no record")
			} else {
				assert.Nil(t, cerr, "did not expect repo error")
				assert.False(t, el.HasError(), "no errors in list")
				require.NotNil(t, rec, "expected record")

				// verify fields
				assert.Equal(t, "canonical", rec.AuthorityID)
				assert.Equal(t, tt.publicKeySha, rec.PublicKeySha3_384Encoded)
				assert.Equal(t, tt.assertionName, rec.Name)
				assert.Equal(t, uint32(2), rec.RevisionSequenceNumber)
				// and so on...
			}

			// clean up
			_, _ = globalDB.Exec(`DELETE FROM account_key_assertion`)
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

func TestAddSnapDeclarationAssertion(t *testing.T) {
	// common inputs
	now := time.Now().UTC().Truncate(time.Second)
	authorityID := "canonical"
	signKey := "sign-key-123"
	assertionID := "snap-abc"
	snapName := "MySnap"
	publisherID := "pub-xyz"
	revision := uint32(42)
	series := "16"
	timestamp := now
	refreshControl := []string{"foo", "bar"}
	aliases := []model.Alias{
		{Name: "alias1", Target: "target1"},
	}
	plugs := model.Plugs{
		"plug1": {"allow-installation": true},
		"plug2": {"deny-installation": false},
	}
	slots := model.Slots{
		"slot1": {"allow-installation": true},
		"slot2": {"deny-installation": false},
	}
	signature := "signed-bytes"

	// pre‐marshal JSONB columns for duplicate‐insert setup
	plugsJSON, _ := json.Marshal(plugs)
	slotsJSON, _ := json.Marshal(slots)

	tests := []struct {
		name         string
		preInsert    bool // insert a conflicting parent before calling
		expectError  bool
		expectedCode string // only for expectError
	}{
		{"successful insertion", false, false, ""},
		{"duplicate parent error", true, true, cerror.AlreadyRegistered},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// if testing duplicate, insert a parent
			if tt.preInsert {
				insert := `
					INSERT INTO snap_declaration_assertion
					  (authority_id, sign_key_sha3_384,
					   snap_id, snap_name, publisher_id,
					   revision, series, timestamp,
					   refresh_control, plugs, slots, signature)
					VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
				`
				_, err := globalDB.Exec(
					insert,
					authorityID, signKey,
					assertionID, snapName, publisherID,
					revision, series, timestamp,
					pq.Array(refreshControl),
					plugsJSON, slotsJSON,
					signature,
				)
				require.NoError(t, err, "setup duplicate parent")
			}

			el := cerror.NewErrorList()
			rec, cerr := globalRepo.AddSnapDeclarationAssertion(
				el,
				authorityID, signKey,
				assertionID, snapName, publisherID,
				revision, series, timestamp,
				refreshControl, aliases,
				plugs, slots,
				signature,
			)

			if tt.expectError {
				assert.NotNil(t, cerr, "expected error")
				assert.True(t, el.HasError(), "error list should be non‑empty")
				if cerr != nil {
					assert.Equal(t, tt.expectedCode, cerr.GetCode())
				}
			} else {
				assert.Nil(t, cerr, "did not expect error")
				assert.False(t, el.HasError(), "error list should be empty")
				assert.NotNil(t, rec, "returned record")
				// spot‑check
				assert.Equal(t, assertionID, rec.SnapID)
				assert.Equal(t, revision, rec.Revision)
				assert.Equal(t, aliases, rec.Aliases)
				assert.Equal(t, plugs, rec.Plugs)
				assert.Equal(t, slots, rec.Slots)
			}

			// clean up for next iteration
			_, _ = globalDB.Exec(`DELETE FROM alias`)
			_, _ = globalDB.Exec(`DELETE FROM snap_declaration_assertion`)
		})
	}
}

func TestGetSnapDeclarationAssertionByID(t *testing.T) {
	tests := []struct {
		name           string
		preInsert      bool
		assertionID    uuid.UUID
		snapID         uuid.UUID
		refreshControl []string
		aliases        []model.Alias
		expectError    bool
		errorCode      string
	}{
		{
			name:           "successful fetch",
			preInsert:      true,
			assertionID:    uuid.New(),
			snapID:         uuid.New(),
			refreshControl: []string{"rc1", "rc2"},
			aliases:        []model.Alias{{Name: "a1", Target: "t1"}, {Name: "a2", Target: "t2"}},
		},
		{
			name:        "not found",
			preInsert:   false,
			assertionID: uuid.New(),
			snapID:      uuid.New(),
			expectError: true,
			errorCode:   cerror.ResourceNotFound,
		},
	}

	now := time.Now().UTC()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.preInsert {
				// insert the parent
				_, err := globalDB.Exec(`
                    INSERT INTO snap_declaration_assertion
                      (id,authority_id, sign_key_sha3_384,
                       snap_id, snap_name, publisher_id,
                       revision, series, timestamp, refresh_control,
                       plugs, slots, signature)
                    VALUES
						($1,'authX','signKeyX',
                       $2,'My Snap','pubX',
                       1,2,$3,$4,
                       '{}'::jsonb,'{}'::jsonb,'sigX')
                `, tt.assertionID, tt.snapID, now, pq.Array(tt.refreshControl))
				assert.NoError(t, err, "failed to insert parent")

				// insert any aliases
				for _, a := range tt.aliases {
					_, err := globalDB.Exec(`
                        INSERT INTO alias (assertion_id,name,target)
                        VALUES ($1,$2,$3)
                    `, tt.assertionID, a.Name, a.Target)
					assert.NoError(t, err, "failed to insert alias %q", a.Name)
				}
			}

			el := cerror.NewErrorList()
			got, cerr := globalRepo.GetSnapDeclarationAssertionBySnapID(el, tt.snapID.String())

			if tt.expectError {
				assert.NotNil(t, cerr, "expected error")
				assert.True(t, el.HasError(), "expected error list")
				assert.Equal(t, tt.errorCode, cerr.GetCode())
			} else {
				assert.Nil(t, cerr, "did not expect error")
				assert.False(t, el.HasError(), "did not expect error list")
				assert.NotNil(t, got, "expected non‑nil result")

				// check the parent fields
				assert.Equal(t, tt.assertionID, got.ID)
				assert.Equal(t, tt.snapID.String(), got.SnapID)
				assert.Equal(t, "authX", got.AuthorityID)
				assert.Equal(t, "signKeyX", got.SignKeySHA3_384)
				assert.Equal(t, uint32(1), got.Revision)
				assert.Equal(t, "2", got.Series)
				assert.Equal(t, now.UTC().Truncate(time.Second), got.Timestamp.UTC().Truncate(time.Second))
				assert.Equal(t, tt.refreshControl, []string(got.RefreshControl))
				assert.Equal(t, "sigX", got.Signature)

				// aliases must come back in name order
				expectedNames := []string{tt.aliases[0].Name, tt.aliases[1].Name}
				var gotNames []string
				for _, a := range got.Aliases {
					gotNames = append(gotNames, a.Name)
				}
				assert.Equal(t, expectedNames, gotNames)
				for i, a := range tt.aliases {
					assert.Equal(t, a.Target, got.Aliases[i].Target)
				}
			}
		})
	}
}

func TestAddAccountAssertion(t *testing.T) {
	tests := []struct {
		name            string
		authorityID     string
		displayName     string
		username        string
		validation      string
		accountID       uuid.UUID
		revision        uint32
		timestamp       time.Time
		signKey         string
		signature       string
		expectDuplicate bool   // whether we expect the second insert to fail
		expectedErrCode string // only used if expectDuplicate==true
	}{
		{
			name:        "successful insert",
			authorityID: "auth-1",
			displayName: "Alice",
			username:    "alice123",
			validation:  "verified",
			accountID:   uuid.New(),
			revision:    1,
			timestamp:   time.Now().UTC(),
			signKey:     "signkey-abc",
			signature:   "sigdata-xyz",
			// expectDuplicate false => we only do one insert
		},
		{
			name:            "duplicate insert",
			authorityID:     "auth-1",
			displayName:     "Alice",
			username:        "alice123",
			validation:      "verified",
			accountID:       uuid.New(),
			revision:        1,
			timestamp:       time.Now().UTC(),
			signKey:         "signkey-dup",
			signature:       "sigdata-dup",
			expectDuplicate: true,
			expectedErrCode: cerror.AlreadyRegistered,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			el := cerror.NewErrorList()

			// First insertion should always succeed.
			rec1, cerr := globalRepo.AddAccountAssertion(el,
				tt.authorityID,
				tt.displayName,
				tt.username,
				tt.validation,
				tt.accountID,
				tt.revision,
				tt.timestamp,
				tt.signKey,
				tt.signature,
			)
			require.Nil(t, cerr, "first insertion should succeed")
			require.NotNil(t, rec1, "first insertion should return a record")

			if tt.expectDuplicate {
				// Second insertion with same accountID+revision should now fail.
				rec2, err2 := globalRepo.AddAccountAssertion(el,
					tt.authorityID,
					tt.displayName,
					tt.username,
					tt.validation,
					tt.accountID,
					tt.revision,
					tt.timestamp,
					tt.signKey,
					tt.signature,
				)
				assert.NotNil(t, err2, "second insertion should error")
				assert.True(t, el.HasError(), "error list should be non‑empty")
				assert.Equal(t, tt.expectedErrCode, err2.GetCode(), "error code must match")
				assert.Nil(t, rec2, "no record should be returned on duplicate")
			}

			// Clean up so next subtest starts fresh.
			_, _ = globalDB.Exec(`DELETE FROM account_assertion`)
		})
	}
}

func TestGetAccountAssertionByAccountID(t *testing.T) {
	tests := []struct {
		name            string
		insertRecord    bool
		accountID       uuid.UUID
		expectError     bool
		expectedErrCode string
	}{
		{
			name:         "found",
			insertRecord: true,
			accountID:    uuid.New(),
			expectError:  false,
		},
		{
			name:            "not found",
			insertRecord:    false,
			accountID:       uuid.New(),
			expectError:     true,
			expectedErrCode: cerror.ResourceNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			el := cerror.NewErrorList()

			// if we need to test found, insert a row first
			if tt.insertRecord {
				_, err := globalDB.Exec(`
					INSERT INTO account_assertion
					(id, authority_id, display_name, username, validation,
					 account_id, revision, timestamp, sign_key_sha3_384, signature)
					VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
					uuid.New(),
					"auth-X", "Bob", "bob123", "ok",
					tt.accountID, 3, time.Now().UTC(), "signkey-X", "sig-X",
				)
				require.NoError(t, err)
			}

			rec, cerr := globalRepo.GetAccountAssertionByAccountID(el, tt.accountID)

			if tt.expectError {
				assert.NotNil(t, cerr)
				assert.True(t, el.HasError())
				assert.Equal(t, tt.expectedErrCode, cerr.GetCode())
				assert.Nil(t, rec)
			} else {
				assert.Nil(t, cerr)
				assert.False(t, el.HasError())
				require.NotNil(t, rec)
				assert.Equal(t, tt.accountID, rec.AccountID)
				assert.Equal(t, uint32(3), rec.Revision)
				assert.Equal(t, "Bob", rec.DisplayName)
				assert.Equal(t, "bob123", rec.Username)
				assert.Equal(t, "ok", rec.Validation)
			}

			// clean up
			_, _ = globalDB.Exec(`DELETE FROM account_assertion`)
		})
	}
}

func TestGetLatestAccountAssertionByAccountID(t *testing.T) {
	ts1 := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	ts2 := ts1.Add(time.Hour)

	tests := []struct {
		name             string
		setupRows        []model.AccountAssertion
		expectFound      bool
		expectedRevision uint32
		expectedErrCode  string
	}{
		{
			name:            "no assertions",
			setupRows:       nil,
			expectFound:     false,
			expectedErrCode: cerror.ResourceNotFound,
		},
		{
			name: "single assertion",
			setupRows: []model.AccountAssertion{
				{AuthorityID: "authX", DisplayName: "Alice", Username: "alice123", Validation: "ok",
					Revision: 7, Timestamp: ts1, SignKeySHA3_384: "key1", Signature: "sig1"},
			},
			expectFound:      true,
			expectedRevision: 7,
		},
		{
			name: "multiple assertions picks latest revision",
			setupRows: []model.AccountAssertion{
				{AuthorityID: "authX", DisplayName: "Alice", Username: "alice123", Validation: "ok",
					Revision: 3, Timestamp: ts1, SignKeySHA3_384: "key3", Signature: "sig3"},
				{AuthorityID: "authX", DisplayName: "Alice", Username: "alice123", Validation: "ok",
					Revision: 5, Timestamp: ts2, SignKeySHA3_384: "key5", Signature: "sig5"},
			},
			expectFound:      true,
			expectedRevision: 5,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			el := cerror.NewErrorList()

			// clean slate
			_, _ = globalDB.Exec(`DELETE FROM account_assertion`)

			// pick a single accountID for all rows in this case
			accountID := uuid.New()
			for i := range tc.setupRows {
				tc.setupRows[i].AccountID = accountID
				_, err := globalDB.Exec(`
                    INSERT INTO account_assertion
                      (authority_id, display_name, username, validation,
                       account_id, revision, timestamp,
                       sign_key_sha3_384, signature)
                    VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
                `,
					tc.setupRows[i].AuthorityID,
					tc.setupRows[i].DisplayName,
					tc.setupRows[i].Username,
					tc.setupRows[i].Validation,
					tc.setupRows[i].AccountID,
					tc.setupRows[i].Revision,
					tc.setupRows[i].Timestamp,
					tc.setupRows[i].SignKeySHA3_384,
					tc.setupRows[i].Signature,
				)
				require.NoError(t, err)
			}

			rec, cerr := globalRepo.GetLatestAccountAssertionByAccountID(el, accountID)
			if tc.expectFound {
				require.Nil(t, cerr)
				require.NotNil(t, rec)
				assert.Equal(t, tc.expectedRevision, rec.Revision, "should pick highest revision")
			} else {
				assert.NotNil(t, cerr)
				assert.Nil(t, rec)
				assert.Equal(t, tc.expectedErrCode, cerr.GetCode())
			}
		})
	}
}

func TestAddSnapBuildAssertion(t *testing.T) {
	tests := []struct {
		name          string
		authorityID   string
		signKey       string
		snapID        uuid.UUID
		accountID     uuid.UUID
		grade         string
		snapSHA3_384  string
		snapSize      uint64
		signature     string
		timestamp     time.Time
		expectError   bool
		expectedError string // expected error code (if any)
	}{
		{
			name:         "Successful SnapBuild Assertion Insertion",
			authorityID:  "kebeng-id",
			signKey:      "sign-key-123",
			snapID:       uuid.New(),
			accountID:    uuid.New(),
			grade:        "stable",
			snapSHA3_384: "test-snap-sha3-384",
			snapSize:     1234567,
			signature:    "AcLBtest-signature-data",
			timestamp:    time.Now().UTC(),
			expectError:  false,
		},
		{
			name:          "Fail Insertion on Missing Grade",
			authorityID:   "kebeng-id",
			signKey:       "sign-key-123",
			snapID:        uuid.New(),
			accountID:     uuid.New(),
			grade:         "", // Missing grade to trigger error
			snapSHA3_384:  "test-snap-sha3-384",
			snapSize:      1234567,
			signature:     "AcLBtest-signature-data",
			timestamp:     time.Now().UTC(),
			expectError:   true,
			expectedError: cerror.InvalidField,
		},
		{
			name:          "Fail Insertion on Invalid Grade",
			authorityID:   "kebeng-id",
			signKey:       "sign-key-123",
			snapID:        uuid.New(),
			accountID:     uuid.New(),
			grade:         "invalid-grade", // Invalid grade to trigger error
			snapSHA3_384:  "test-snap-sha3-384",
			snapSize:      1234567,
			signature:     "AcLBtest-signature-data",
			timestamp:     time.Now().UTC(),
			expectError:   true,
			expectedError: cerror.InvalidField,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			el := cerror.NewErrorList()
			record, cerr := globalRepo.AddSnapBuildAssertion(
				el,
				tt.authorityID,
				tt.signKey,
				tt.snapID,
				tt.accountID,
				tt.grade,
				tt.snapSHA3_384,
				tt.snapSize,
				tt.signature,
				tt.timestamp,
			)

			if tt.expectError {
				assert.NotNil(t, cerr, "Expected an error during insertion")
				assert.True(t, el.HasError(), "Expected an error in the list")
				if cerr != nil {
					assert.Equal(t, tt.expectedError, cerr.GetCode(), "Error code should match expected")
				}
				assert.Nil(t, record, "No record should be returned on error")
			} else {
				assert.Nil(t, cerr, "Did not expect an error during insertion")
				assert.False(t, el.HasError(), "Expected no errors in the list")
				assert.NotNil(t, record, "Expected a non-nil assertion record")
				assert.NotEqual(t, uuid.Nil, record.ID, "Expected a valid UUID for the record ID")
			}
		})
	}
}
