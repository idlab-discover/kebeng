package logic

import (
	"context"
	"encoding/base64"
	"testing"
	"time"

	"github.com/google/uuid"
	cerror "github.com/idlab-discover/kebeng/common/cerror"
	"github.com/idlab-discover/kebeng/services/assertion/internal/config"
	"github.com/idlab-discover/kebeng/services/assertion/internal/model"
	"github.com/idlab-discover/kebeng/services/assertion/internal/repository"
	proto "github.com/idlab-discover/kebeng/services/assertion/proto"
	"github.com/snapcore/snapd/asserts"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/protobuf/types/known/timestamppb"
)

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
				AccountId:         validUUID,
				PublicKeySha3_384: accountKey.PublicKey().ID(),
				EncodedPublicKey:  accountPubBase64,
				Name:              "my-key",
				Since:             timestamppb.New(now),
			},
			mockReturn: map[string]any{
				"GetLatestAccountKeyAssertion": cerror.NewCustomError(cerror.ResourceNotFound, "no previous record found"),
				"AddAccountKeyAssertion": &model.AccountKeyAssertion{
					ID:                     uuid.New(),
					CreatedAt:              now,
					DeletedAt:              nil,
					Type:                   "account-key",
					AuthorityID:            cfg.AuthorityID,
					RevisionSequenceNumber: 1,
					PublicKeySHA3_384:      accountKey.PublicKey().ID(),
					AccountID:              parsedUUID,
					Name:                   "my-key",
					Since:                  now,
					Until:                  now.Add(365 * 24 * time.Hour),
					BodyLength:             uint64(len("body")),
					Body:                   []byte("body"),           // may be updated per actual usage
					SignKeySHA3_384:        privKey.PublicKey().ID(), // store's signature key ID
					Signature:              "signature",
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
				AccountId:         validUUID,
				PublicKeySha3_384: accountKey.PublicKey().ID(),
				EncodedPublicKey:  pubB64,
				Name:              "foo",
				Since:             timestamppb.New(now),
				Until:             timestamppb.New(now.Add(365 * 24 * time.Hour)),
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
				AccountId:         validUUID,
				PublicKeySha3_384: accountKey.PublicKey().ID(),
				EncodedPublicKey:  badB64,
				Name:              "foo",
				Since:             timestamppb.New(now),
				Until:             timestamppb.New(now.Add(365 * 24 * time.Hour)),
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
				AccountId:         validUUID,
				PublicKeySha3_384: accountKey.PublicKey().ID(),
				EncodedPublicKey:  decodeButInvalid,
				Name:              "foo",
				Since:             timestamppb.New(now),
				Until:             timestamppb.New(now.Add(365 * 24 * time.Hour)),
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
				AccountId:         validUUID,
				PublicKeySha3_384: accountKey.PublicKey().ID(),
				EncodedPublicKey:  pubB64,
				Name:              "foo",
				Since:             timestamppb.New(now),
				Until:             timestamppb.New(now.Add(365 * 24 * time.Hour)),
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
				AccountId:         validUUID,
				PublicKeySha3_384: accountKey.PublicKey().ID(),
				EncodedPublicKey:  pubB64,
				Name:              "my-key",
				Since:             timestamppb.New(now),
				Until:             timestamppb.New(now.Add(365 * 24 * time.Hour)),
			},
			mockReturn: map[string]any{
				"GetLatestAccountKeyAssertion": cerror.NewCustomError(cerror.ResourceNotFound, "none"),
				"AddAccountKeyAssertion": &model.AccountKeyAssertion{
					ID:                     uuid.New(),
					CreatedAt:              now,
					DeletedAt:              nil,
					Type:                   "account-key",
					AuthorityID:            cfg.AuthorityID,
					RevisionSequenceNumber: 1,
					PublicKeySHA3_384:      accountKey.PublicKey().ID(),
					AccountID:              parsedUUID,
					Name:                   "my-key",
					Since:                  now,
					Until:                  now.Add(365 * 24 * time.Hour),
					BodyLength:             uint64(len(pubRaw)),
					Body:                   pubRaw,
					SignKeySHA3_384:        privKey.PublicKey().ID(),
					Signature:              "signature",
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
				assert.Equal(t, tc.req.PublicKeySha3_384, resp.PublicKeySha3_384)
				assert.NotEmpty(t, resp.Signature)
			}

			// Ensure all mocked expectations were met
			mockRepo.AssertExpectations(t)
		})
	}
}
