package logic

import (
	"context"
	"encoding/base64"
	"testing"
	"time"

	"github.com/idlab-discover/kebeng/services/assertion/internal/config"
	"github.com/idlab-discover/kebeng/services/assertion/internal/model"
	"github.com/idlab-discover/kebeng/services/assertion/internal/repository"
	proto "github.com/idlab-discover/kebeng/services/assertion/proto"

	"github.com/google/uuid"
	cerror "github.com/idlab-discover/kebeng/common/cerror"
	"github.com/lib/pq"
	"github.com/snapcore/snapd/asserts"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestAddSnapRevisionAssertion(t *testing.T) {
	rootKey, err := asserts.GenerateKey()
	assert.NoError(t, err)

	assertionDB, err := asserts.OpenDatabase(&asserts.DatabaseConfig{})
	assert.NoError(t, err)

	cfg := &config.Config{
		AuthorityID: "kebeng",
		RootKey:     rootKey,
	}
	assert.NoError(t, assertionDB.ImportKey(rootKey))

	mockRepo := new(repository.MockAssertionRepository)
	svc := &AssertionService{
		cfg:         cfg,
		assertionDB: assertionDB,
		repo:        mockRepo,
	}

	now := time.Now().UTC()
	validDev := uuid.New().String()
	validSnap := uuid.New().String()

	// generate a valid multihash to use as snap‐sha3‐384
	snapKey, err := asserts.GenerateKey()
	assert.NoError(t, err)
	validHash := snapKey.PublicKey().ID()

	// a “golden” model record
	golden := &model.SnapRevisionAssertion{
		ID:                         uuid.New(),
		AuthorityID:                cfg.AuthorityID,
		SignKeySHA3_384:            rootKey.PublicKey().ID(),
		SnapSHA3_384:               validHash,
		DeveloperID:                uuid.MustParse(validDev),
		SnapEntryID:                uuid.MustParse(validSnap),
		SnapRevisionSequenceNumber: 1,
		SnapSize:                   42,
		Timestamp:                  now,
		Type:                       asserts.SnapRevisionType.Name,
		Signature:                  "signature-bytes",
	}

	tests := []struct {
		name               string
		req                *proto.AddSnapRevisionAssertionRequest
		mockReturn         map[string]any // only "AddSnapRevisionAssertion" key
		expectError        bool           // Go‐error != nil
		expectProtoErrors  bool           // response.Errors non‐empty
		expectProtoErrCode string         // if expectProtoErrors
	}{
		{
			name: "happy path",
			req: &proto.AddSnapRevisionAssertionRequest{
				DeveloperId:                validDev,
				SnapEntryId:                validSnap,
				SnapSha3_384:               validHash,
				SnapRevisionSequenceNumber: 1,
				SnapSize:                   42,
				Timestamp:                  timestamppb.New(now),
			},
			mockReturn: map[string]any{
				"AddSnapRevisionAssertion": golden,
			},
		},
		{
			name: "invalid developer id",
			req: &proto.AddSnapRevisionAssertionRequest{
				DeveloperId:                "not-a-uuid",
				SnapEntryId:                validSnap,
				SnapSha3_384:               validHash,
				SnapRevisionSequenceNumber: 1,
				SnapSize:                   42,
				Timestamp:                  timestamppb.New(now),
			},
			mockReturn:         nil,
			expectProtoErrors:  true,
			expectProtoErrCode: cerror.Invalid,
		},
		{
			name: "invalid snap entry id",
			req: &proto.AddSnapRevisionAssertionRequest{
				DeveloperId:                validDev,
				SnapEntryId:                "not-a-uuid",
				SnapSha3_384:               validHash,
				SnapRevisionSequenceNumber: 1,
				SnapSize:                   42,
				Timestamp:                  timestamppb.New(now),
			},
			mockReturn:         nil,
			expectProtoErrors:  true,
			expectProtoErrCode: cerror.Invalid,
		},
		{
			name: "signing failure (bad hash length)",
			req: &proto.AddSnapRevisionAssertionRequest{
				DeveloperId:                validDev,
				SnapEntryId:                validSnap,
				SnapSha3_384:               "too-short",
				SnapRevisionSequenceNumber: 1,
				SnapSize:                   42,
				Timestamp:                  timestamppb.New(now),
			},
			mockReturn:         nil, // no repo call expected
			expectProtoErrors:  true,
			expectProtoErrCode: cerror.Invalid,
		},
		{
			name: "repo error",
			req: &proto.AddSnapRevisionAssertionRequest{
				DeveloperId:                validDev,
				SnapEntryId:                validSnap,
				SnapSha3_384:               validHash,
				SnapRevisionSequenceNumber: 1,
				SnapSize:                   42,
				Timestamp:                  timestamppb.New(now),
			},
			mockReturn: map[string]any{
				"AddSnapRevisionAssertion": cerror.NewCustomError(cerror.DatabaseError, "insert failed"),
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// set up mock expectations
			for fn, ret := range tc.mockReturn {
				switch fn {
				case "AddSnapRevisionAssertion":
					switch v := ret.(type) {
					case *model.SnapRevisionAssertion:
						mockRepo.
							On("AddSnapRevisionAssertion",
								mock.Anything, cfg.AuthorityID, tc.req.GetSnapSha3_384(),
								rootKey.PublicKey().ID(), mock.Anything, mock.Anything,
								tc.req.GetSnapRevisionSequenceNumber(), tc.req.GetSnapSize(),
								tc.req.GetTimestamp().AsTime(), mock.Anything,
							).
							Return(v, nil).
							Once()
					case *cerror.CustomError:
						mockRepo.
							On("AddSnapRevisionAssertion",
								mock.Anything, mock.Anything, mock.Anything,
								mock.Anything, mock.Anything, mock.Anything,
								mock.Anything, mock.Anything, mock.Anything, mock.Anything,
							).
							Return(nil, v).
							Once()
					default:
						t.Fatalf("unknown mockReturn[%s] type %T", fn, v)
					}
				}
			}

			resp, err := svc.AddSnapRevisionAssertion(context.Background(), tc.req)

			if tc.expectError {
				assert.Error(t, err, tc.name)
				assert.Nil(t, resp, tc.name)
			} else {
				// service always returns (resp, nil) except repo‐error
				assert.NoError(t, err, tc.name)
				assert.NotNil(t, resp, tc.name)

				if tc.expectProtoErrors {
					assert.NotEmpty(t, resp.Errors, tc.name)
					found := false
					for _, e := range resp.Errors {
						if e.Code == tc.expectProtoErrCode {
							found = true
							break
						}
					}
					assert.True(t, found, "expected error code %q", tc.expectProtoErrCode)
				} else {
					assert.Empty(t, resp.Errors, tc.name)
					// verify the happy‐path result matches our golden
					assert.Equal(t, golden.ID.String(), resp.Id)
					assert.Equal(t, golden.SignKeySHA3_384, resp.SignKeySha3_384)
					assert.Equal(t, validDev, resp.DeveloperId)
					assert.Equal(t, validSnap, resp.SnapEntryId)
					assert.Equal(t, validHash, resp.SnapSha3_384)
					assert.Equal(t, uint64(42), resp.SnapSize)
					assert.Equal(t, uint32(1), resp.SnapRevisionSequenceNumber)
					assert.NotEmpty(t, resp.Signature)
				}
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestAddAccountKeyAssertion(t *testing.T) {
	privKey, err := asserts.GenerateKey()
	assert.NoError(t, err)

	assertionDB, err := asserts.OpenDatabase(&asserts.DatabaseConfig{})
	assert.NoError(t, err)

	cfg := &config.Config{
		AuthorityID: "kebeng",
		RootKey:     privKey,
	}
	err = assertionDB.ImportKey(privKey)
	assert.NoError(t, err)

	mockRepo := new(repository.MockAssertionRepository)
	svc := &AssertionService{cfg: cfg, repo: mockRepo, assertionDB: assertionDB}

	now := time.Now().UTC()
	validUUID := "3fa85f64-5717-4562-b3fc-2c963f66afa6"
	parsedUUID, _ := uuid.Parse(validUUID)

	// Before defining your test table, generate a valid account key.
	accountKey, err := asserts.GenerateKey()
	assert.NoError(t, err)

	// Get the serialized bytes of the public key.
	serializedPub, err := asserts.EncodePublicKey(accountKey.PublicKey())
	assert.NoError(t, err)

	accountPubBase64 := base64.StdEncoding.EncodeToString(serializedPub)
	pubRaw, _ := asserts.EncodePublicKey(accountKey.PublicKey())
	pubB64 := base64.StdEncoding.EncodeToString(pubRaw)
	badB64 := "!!!notbase64!!!"
	decodeButInvalid := base64.StdEncoding.EncodeToString([]byte("foobar"))

	tests := []struct {
		name              string
		req               *proto.AddAccountKeyAssertionRequest
		mockReturn        map[string]any // ...
		expectedError     bool
		expectedErrorCode string
		expectedRevision  uint32
	}{
		{
			name: "Valid no previous record",
			req: &proto.AddAccountKeyAssertionRequest{
				AccountId:                validUUID,
				PublicKeySha3_384Encoded: accountKey.PublicKey().ID(),
				EncodedPublicKey:         accountPubBase64,
				Name:                     "my-key",
				Since:                    timestamppb.New(now),
			},
			mockReturn: map[string]any{
				"GetLatestAccountKeyAssertion": cerror.NewCustomError(cerror.ResourceNotFound, "no previous record found"),
				"AddAccountKeyAssertion": &model.AccountKeyAssertion{
					ID:                       uuid.New(),
					CreatedAt:                now,
					DeletedAt:                nil,
					Type:                     "account-key",
					AuthorityID:              cfg.AuthorityID,
					RevisionSequenceNumber:   1,
					PublicKeySha3_384Encoded: accountKey.PublicKey().ID(),
					AccountID:                parsedUUID,
					Name:                     "my-key",
					Since:                    now,
					Until:                    now.Add(365 * 24 * time.Hour),
					BodyLength:               uint64(len("body")),
					Body:                     []byte("body"),           // may be updated per actual usage
					SignKeySHA3_384:          privKey.PublicKey().ID(), // store's signature key ID
					Signature:                "signature",
				},
			},
			expectedError:    false,
			expectedRevision: 1,
		},
		{
			name: "Invalid account id",
			req: &proto.AddAccountKeyAssertionRequest{
				AccountId: "not-a-uuid",
			},
			mockReturn:        nil,
			expectedError:     true,
			expectedErrorCode: cerror.InvalidField,
		},
		{
			name: "Repo GetLatest unexpected error",
			req: &proto.AddAccountKeyAssertionRequest{
				AccountId:                validUUID,
				PublicKeySha3_384Encoded: accountKey.PublicKey().ID(),
				EncodedPublicKey:         pubB64,
				Name:                     "foo",
				Since:                    timestamppb.New(now),
				Until:                    timestamppb.New(now.Add(365 * 24 * time.Hour)),
			},
			mockReturn: map[string]any{
				"GetLatestAccountKeyAssertion": cerror.NewCustomError(cerror.DatabaseError, "db down"),
			},
			expectedError:     true,
			expectedErrorCode: cerror.DatabaseError,
		},
		{
			name: "Invalid base64 in EncodedPublicKey",
			req: &proto.AddAccountKeyAssertionRequest{
				AccountId:                validUUID,
				PublicKeySha3_384Encoded: accountKey.PublicKey().ID(),
				EncodedPublicKey:         badB64,
				Name:                     "foo",
				Since:                    timestamppb.New(now),
				Until:                    timestamppb.New(now.Add(365 * 24 * time.Hour)),
			},
			mockReturn: map[string]any{
				"GetLatestAccountKeyAssertion": cerror.NewCustomError(cerror.ResourceNotFound, "none"),
			},
			expectedError:     true,
			expectedErrorCode: cerror.Invalid,
		},
		{
			name: "DecodePublicKey packet error",
			req: &proto.AddAccountKeyAssertionRequest{
				AccountId:                validUUID,
				PublicKeySha3_384Encoded: accountKey.PublicKey().ID(),
				EncodedPublicKey:         decodeButInvalid,
				Name:                     "foo",
				Since:                    timestamppb.New(now),
				Until:                    timestamppb.New(now.Add(365 * 24 * time.Hour)),
			},
			mockReturn: map[string]any{
				"GetLatestAccountKeyAssertion": cerror.NewCustomError(cerror.ResourceNotFound, "none"),
			},
			expectedError:     true,
			expectedErrorCode: cerror.Invalid,
		},
		{
			name: "Repo AddAccountKeyAssertion error",
			req: &proto.AddAccountKeyAssertionRequest{
				AccountId:                validUUID,
				PublicKeySha3_384Encoded: accountKey.PublicKey().ID(),
				EncodedPublicKey:         pubB64,
				Name:                     "foo",
				Since:                    timestamppb.New(now),
				Until:                    timestamppb.New(now.Add(365 * 24 * time.Hour)),
			},
			mockReturn: map[string]any{
				"GetLatestAccountKeyAssertion": cerror.NewCustomError(cerror.ResourceNotFound, "none"),
				"AddAccountKeyAssertion":       cerror.NewCustomError(cerror.DatabaseError, "insert fail"),
			},
			expectedError:     true,
			expectedErrorCode: cerror.DatabaseError,
		},
		{
			name: "Successful path",
			req: &proto.AddAccountKeyAssertionRequest{
				AccountId:                validUUID,
				PublicKeySha3_384Encoded: accountKey.PublicKey().ID(),
				EncodedPublicKey:         pubB64,
				Name:                     "my-key",
				Since:                    timestamppb.New(now),
				Until:                    timestamppb.New(now.Add(365 * 24 * time.Hour)),
			},
			mockReturn: map[string]any{
				"GetLatestAccountKeyAssertion": cerror.NewCustomError(cerror.ResourceNotFound, "none"),
				"AddAccountKeyAssertion": &model.AccountKeyAssertion{
					ID:                       uuid.New(),
					CreatedAt:                now,
					DeletedAt:                nil,
					Type:                     "account-key",
					AuthorityID:              cfg.AuthorityID,
					RevisionSequenceNumber:   1,
					PublicKeySha3_384Encoded: accountKey.PublicKey().ID(),
					AccountID:                parsedUUID,
					Name:                     "my-key",
					Since:                    now,
					Until:                    now.Add(365 * 24 * time.Hour),
					BodyLength:               uint64(len(pubRaw)),
					Body:                     pubRaw,
					SignKeySHA3_384:          privKey.PublicKey().ID(),
					Signature:                "signature",
				},
			},
			expectedError:    false,
			expectedRevision: 1,
		},
		// ... additional test cases
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			for function, ret := range tc.mockReturn {
				switch function {
				case "GetLatestAccountKeyAssertion":
					if errVal, ok := ret.(*cerror.CustomError); ok {
						mockRepo.
							On("GetLatestAccountKeyAssertion", mock.Anything, mock.Anything).
							Return(nil, errVal).Once()
					} else if assertionRecord, ok := ret.(*model.AccountKeyAssertion); ok {
						mockRepo.
							On("GetLatestAccountKeyAssertion", mock.Anything, mock.Anything).
							Return(assertionRecord, nil).Once()
					} else {
						t.Fatalf("Invalid type for mock return of GetLatestAccountKeyAssertion")
					}
				case "AddAccountKeyAssertion":
					if errVal, ok := ret.(*cerror.CustomError); ok {
						mockRepo.
							On("AddAccountKeyAssertion", mock.Anything, mock.Anything, mock.Anything,
								mock.Anything, mock.Anything, mock.Anything, mock.Anything,
								mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
							Return(nil, errVal).Once()
					} else if assertionRecord, ok := ret.(*model.AccountKeyAssertion); ok {
						mockRepo.
							On("AddAccountKeyAssertion", mock.Anything, mock.Anything, mock.Anything,
								mock.Anything, mock.Anything, mock.Anything, mock.Anything,
								mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
							Return(assertionRecord, nil).Once()
					} else {
						t.Fatalf("Invalid type for mock return of AddAccountKeyAssertion")
					}
				default:
					t.Fatalf("Unknown function key: %s", function)
				}
			}

			resp, err := svc.AddAccountKeyAssertion(context.Background(), tc.req)

			if tc.expectedError {
				assert.NotNil(t, resp)
				assert.NotEmpty(t, resp.Errors)
				found := false
				for _, e := range resp.Errors {
					if e.Code == tc.expectedErrorCode {
						found = true
						break
					}
				}
				assert.True(t, found, "expected error code %s not found", tc.expectedErrorCode)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedRevision, resp.RevisionSequenceNumber)
				assert.Equal(t, tc.req.Name, resp.Name)
				assert.Equal(t, tc.req.AccountId, resp.AccountId)
				assert.Equal(t, tc.req.PublicKeySha3_384Encoded, resp.PublicKeySha3_384Encoded)
				assert.NotEmpty(t, resp.Signature)
			}

			// Ensure all mocked expectations were met
			mockRepo.AssertExpectations(t)
		})
	}
}

func TestAddSnapDeclarationAssertion(t *testing.T) {
	rootKey, err := asserts.GenerateKey()
	assert.NoError(t, err)

	assertionDB, err := asserts.OpenDatabase(&asserts.DatabaseConfig{})
	assert.NoError(t, err)
	assert.NoError(t, assertionDB.ImportKey(rootKey))

	cfg := &config.Config{
		AuthorityID: "kebeng",
		RootKey:     rootKey,
	}

	mockRepo := new(repository.MockAssertionRepository)
	svc := &AssertionService{
		cfg:         cfg,
		assertionDB: assertionDB,
		repo:        mockRepo,
	}

	now := time.Now().UTC()

	// a “golden” record for the happy‑path
	goldenModel := &model.SnapDeclarationAssertion{
		ID:              uuid.New(),
		AuthorityID:     cfg.AuthorityID,
		SignKeySHA3_384: rootKey.PublicKey().ID(),
		SnapID:          "snap-123",
		SnapName:        "test-snap",
		PublisherID:     "pub-456",
		Revision:        1, // first revision
		Series:          42,
		Timestamp:       now,
		RefreshControl:  pq.StringArray{"refresh-control"},
		Aliases:         []model.Alias{{Name: "alias1", Target: "target1"}, {Name: "alias2", Target: "target2"}},
		Plugs:           map[string]*model.Plug{"foo": {AllowConnection: boolPtr(true)}},
		Slots:           map[string]*model.Slot{"bar": {AllowInstallation: boolPtr(false)}},
		Type:            asserts.SnapDeclarationType.Name,
		Signature:       "deadbeef",
	}

	notFoundErr := cerror.NewCustomError(cerror.ResourceNotFound, "none")
	dbErr := cerror.NewCustomError(cerror.DatabaseError, "db down")

	tests := []struct {
		name             string
		req              *proto.AddSnapDeclarationAssertionRequest
		mockGetLatest    any // *model.SnapDeclarationAssertion or *cerror.CustomError
		mockAdd          any // *model.SnapDeclarationAssertion or *cerror.CustomError
		wantProtoErr     bool
		wantProtoErrCode string
	}{
		{
			name: "happy path",
			req: &proto.AddSnapDeclarationAssertionRequest{
				SnapId:         "snap-123",
				SnapName:       "test-snap",
				PublisherId:    "pub-456",
				Series:         42,
				Timestamp:      timestamppb.New(now),
				RefreshControl: []string{"refresh-control"},
				Aliases:        []*proto.Alias{{Name: "alias1", Target: "target1"}, {Name: "alias2", Target: "target2"}},
				Plugs:          map[string]*proto.PlugRule{"foo": {AllowConnection: boolPtr(true)}},
				Slots:          map[string]*proto.SlotRule{"bar": {AllowInstallation: boolPtr(false)}},
			},
			mockGetLatest: notFoundErr, // triggers sequence = 1
			mockAdd:       goldenModel, // insert succeeds
		},
		{
			name: "GetLatest error",
			req: &proto.AddSnapDeclarationAssertionRequest{
				SnapId:         "snap-123",
				SnapName:       "test-snap",
				PublisherId:    "pub-456",
				Series:         42,
				Timestamp:      timestamppb.New(now),
				RefreshControl: []string{"refresh-control"},
			},
			mockGetLatest: dbErr, // unexpected error
			// we never reach AddSnapDeclarationAssertion
			wantProtoErr:     true,
			wantProtoErrCode: cerror.DatabaseError,
		},
		{
			name: "AddSnapDeclarationAssertion error",
			req: &proto.AddSnapDeclarationAssertionRequest{
				SnapId:         "snap-123",
				SnapName:       "test-snap",
				PublisherId:    "pub-456",
				Series:         42,
				Timestamp:      timestamppb.New(now),
				RefreshControl: []string{"refresh-control"},
			},
			mockGetLatest:    notFoundErr,
			mockAdd:          dbErr, // insert fails
			wantProtoErr:     true,
			wantProtoErrCode: cerror.DatabaseError,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// --- mock GetLatestSnapDeclarationAssertion ---
			switch ret := tc.mockGetLatest.(type) {
			case *model.SnapDeclarationAssertion:
				mockRepo.
					On("GetLatestSnapDeclarationAssertion", mock.Anything, tc.req.GetSnapId()).
					Return(ret, nil).
					Once()
			case *cerror.CustomError:
				mockRepo.
					On("GetLatestSnapDeclarationAssertion", mock.Anything, tc.req.GetSnapId()).
					Return(nil, ret).
					Once()
			default:
				t.Fatalf("unexpected mockGetLatest type %T", ret)
			}

			// --- mock AddSnapDeclarationAssertion if needed ---
			if tc.mockAdd != nil {
				switch ret := tc.mockAdd.(type) {
				case *model.SnapDeclarationAssertion:
					mockRepo.
						On("AddSnapDeclarationAssertion",
							mock.Anything, mock.Anything, mock.Anything,
							mock.Anything, mock.Anything, mock.Anything,
							mock.Anything, mock.Anything,
							mock.Anything, mock.Anything, mock.Anything,
							mock.Anything, mock.Anything, mock.Anything,
						).
						Return(ret, nil).
						Once()
				case *cerror.CustomError:
					mockRepo.
						On("AddSnapDeclarationAssertion",
							mock.Anything, mock.Anything, mock.Anything,
							mock.Anything, mock.Anything, mock.Anything,
							mock.Anything, mock.Anything,
							mock.Anything, mock.Anything, mock.Anything,
							mock.Anything, mock.Anything, mock.Anything,
						).
						Return(nil, ret).
						Once()
				default:
					t.Fatalf("unexpected mockAdd type %T", ret)
				}
			}

			// --- call under test ---
			resp, err := svc.AddSnapDeclarationAssertion(context.Background(), tc.req)

			// --- verify ---
			if tc.wantProtoErr {
				assert.NotNil(t, resp)
				assert.NotEmpty(t, resp.Errors)
				found := false
				for _, e := range resp.Errors {
					if e.Code == tc.wantProtoErrCode {
						found = true
					}
				}
				assert.True(t, found, "expected proto error %q", tc.wantProtoErrCode)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, resp)
				assert.Empty(t, resp.Errors)

				// happy‑path fields
				assert.Equal(t, goldenModel.ID.String(), resp.Id)
				assert.Equal(t, goldenModel.SignKeySHA3_384, resp.SignKeySha3_384)
				assert.Equal(t, goldenModel.SnapID, resp.SnapId)
				assert.Equal(t, goldenModel.SnapName, resp.SnapName)
				assert.Equal(t, goldenModel.PublisherID, resp.PublisherId)
				assert.Equal(t, uint32(1), resp.Revision) // first sequence
				assert.Equal(t, goldenModel.Series, resp.Series)
				assert.Equal(t, tc.req.GetRefreshControl(), resp.RefreshControl)
				assert.Equal(t, tc.req.GetAliases(), resp.Aliases)
				assert.Equal(t, tc.req.GetPlugs(), resp.Plugs)
				assert.Equal(t, tc.req.GetSlots(), resp.Slots)
				assert.NotEmpty(t, resp.Signature)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

// helper to get a *bool
func boolPtr(b bool) *bool { return &b }

func TestAddAccountAssertion(t *testing.T) {
	// set up signing DB and root key
	rootKey, err := asserts.GenerateKey()
	assert.NoError(t, err)
	assertionDB, err := asserts.OpenDatabase(&asserts.DatabaseConfig{})
	assert.NoError(t, err)
	assert.NoError(t, assertionDB.ImportKey(rootKey))

	cfg := &config.Config{
		AuthorityID: "test-auth",
		RootKey:     rootKey,
	}

	mockRepo := new(repository.MockAssertionRepository)
	svc := &AssertionService{
		cfg:         cfg,
		assertionDB: assertionDB,
		repo:        mockRepo,
	}

	now := time.Now().UTC().Truncate(time.Second)
	validAccountID := uuid.New()

	// a “golden” model for the happy‑path
	goldenModel := &model.AccountAssertion{
		ID:              uuid.New(),
		AuthorityID:     cfg.AuthorityID,
		AccountID:       validAccountID,
		Revision:        1,
		Timestamp:       now,
		SignKeySHA3_384: rootKey.PublicKey().ID(),
		Type:            asserts.AccountType.Name,
		Signature:       "ignored-in-test",
	}

	notFoundErr := cerror.NewCustomError(cerror.ResourceNotFound, "none")
	dbErr := cerror.NewCustomError(cerror.DatabaseError, "store failure")

	tests := []struct {
		name             string
		req              *proto.AddAccountAssertionRequest
		mockGetLatest    any // *model.AccountAssertion or *cerror.CustomError
		mockAdd          any // *model.AccountAssertion or *cerror.CustomError
		wantProtoErr     bool
		wantProtoErrCode string
	}{
		{
			name: "invalid-uuid",
			req:  &proto.AddAccountAssertionRequest{AccountId: "not-a-uuid", DisplayName: "dn", Username: "un", Validation: "v", Timestamp: timestamppb.New(now)},
			// no repo calls at all
			wantProtoErr:     true,
			wantProtoErrCode: cerror.Invalid,
		},
		{
			name:          "happy-path",
			req:           &proto.AddAccountAssertionRequest{AccountId: validAccountID.String(), DisplayName: "dn", Username: "un", Validation: "v", Timestamp: timestamppb.New(now)},
			mockGetLatest: notFoundErr,
			mockAdd:       goldenModel,
			wantProtoErr:  false,
		},
		{
			name:             "get-latest-error",
			req:              &proto.AddAccountAssertionRequest{AccountId: validAccountID.String(), DisplayName: "dn", Username: "un", Validation: "v", Timestamp: timestamppb.New(now)},
			mockGetLatest:    dbErr,
			wantProtoErr:     true,
			wantProtoErrCode: cerror.DatabaseError,
		},
		{
			name:             "add-error",
			req:              &proto.AddAccountAssertionRequest{AccountId: validAccountID.String(), DisplayName: "dn", Username: "un", Validation: "v", Timestamp: timestamppb.New(now)},
			mockGetLatest:    notFoundErr,
			mockAdd:          dbErr,
			wantProtoErr:     true,
			wantProtoErrCode: cerror.DatabaseError,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// --- mock GetLatestAccountAssertionByAccountID ---
			if tc.mockGetLatest != nil {
				switch ret := tc.mockGetLatest.(type) {
				case *model.AccountAssertion:
					mockRepo.
						On("GetLatestAccountAssertionByAccountID", mock.Anything, validAccountID).
						Return(ret, nil).
						Once()
				case *cerror.CustomError:
					mockRepo.
						On("GetLatestAccountAssertionByAccountID", mock.Anything, validAccountID).
						Return(nil, ret).
						Once()
				default:
					// for invalid-uuid case we don't expect a call
				}
			}

			// --- mock AddAccountAssertion if needed ---
			if tc.mockAdd != nil {
				switch ret := tc.mockAdd.(type) {
				case *model.AccountAssertion:
					mockRepo.
						On("AddAccountAssertion",
							mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything,
							mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything,
						).
						Return(ret, nil).
						Once()
				case *cerror.CustomError:
					mockRepo.
						On("AddAccountAssertion",
							mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything,
							mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything,
						).
						Return(nil, ret).
						Once()
				}
			}

			// --- call under test ---
			resp, err := svc.AddAccountAssertion(context.Background(), tc.req)
			assert.NoError(t, err, "service returns nil error always")

			if tc.wantProtoErr {
				assert.NotNil(t, resp)
				assert.NotEmpty(t, resp.Errors, "expected non‑empty Errors")
				found := false
				for _, e := range resp.Errors {
					if e.Code == tc.wantProtoErrCode {
						found = true
						break
					}
				}
				assert.Truef(t, found, "expected error code %q in response", tc.wantProtoErrCode)
			} else {
				assert.Empty(t, resp.Errors, "expected no errors")
				// check happy‑path fields
				assert.Equal(t, goldenModel.ID.String(), resp.Id)
				assert.Equal(t, goldenModel.SignKeySHA3_384, resp.SignKeySha3_384)
				assert.Equal(t, goldenModel.AccountID.String(), resp.AccountId)
				assert.Equal(t, asserts.AccountType.Name, resp.Type)
				// timestamp matches truncated second
				gotTs := resp.Timestamp.AsTime().UTC().Truncate(time.Second)
				assert.WithinDuration(t, now, gotTs, time.Second)
				assert.NotEmpty(t, resp.Signature)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}
