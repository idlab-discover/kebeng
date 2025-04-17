package logic

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	cerror "github.com/idlab-discover/kebeng/common/cerror"
	cerrorpb "github.com/idlab-discover/kebeng/common/cerror/proto"
	"github.com/idlab-discover/kebeng/services/assertion/internal/config"
	"github.com/idlab-discover/kebeng/services/assertion/internal/repository"
	proto "github.com/idlab-discover/kebeng/services/assertion/proto"
	"github.com/sirupsen/logrus"
	"github.com/snapcore/snapd/asserts"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// TODO: maybe call this different?
// this should contain the business logic of the service
// so doing checks and stuff and calling the database logic

type AssertionService struct {
	cfg *config.Config
	proto.UnimplementedAssertionServiceServer
	repo        repository.IAssertionRepository
	assertionDB *asserts.Database
}

func NewAssertionLogic(cfg *config.Config, repo repository.IAssertionRepository, assertionDB *asserts.Database) *AssertionService {
	return &AssertionService{cfg: cfg, repo: repo, assertionDB: assertionDB}
}

// ################### ADDERS #####################

func (s *AssertionService) AddSnapRevisionAssertion(ctx context.Context, req *proto.AddSnapRevisionAssertionRequest) (*proto.SnapRevisionAssertionResponse, error) {
	el := cerror.NewErrorList()

	// empty fields are checked at database layer so we don't need to check them here, and we assume a valid request
	parsedDeveloperId, err := uuid.Parse(req.GetDeveloperId())
	if err != nil {
		logrus.Errorf("failed to parse developer id: %s", err)
		el.Add(cerror.Invalid, fmt.Sprintf("invalid developer id could not parse to uuid: %s", err))
	}
	parsedSnapEntryId, err := uuid.Parse(req.GetSnapEntryId())
	if err != nil {
		logrus.Errorf("failed to parse snap entry id: %s", err)
		el.Add(cerror.Invalid, fmt.Sprintf("invalid snap entry id could not parse to uuid: %s", err))
	}

	headers := map[string]any{
		"authority-id":  s.cfg.AuthorityID,
		"snap-sha3-384": req.GetSnapSha3_384(),
		"developer-id":  req.GetDeveloperId(),
		"snap-id":       req.GetSnapEntryId(),
		"snap-revision": fmt.Sprintf("%d", req.GetSnapRevisionSequenceNumber()),
		"snap-size":     fmt.Sprintf("%d", req.GetSnapSize()),
		"timestamp":     req.GetTimestamp().AsTime().Format(time.RFC3339),
		// The 'sign-key-sha3-384' header is generated during signing.
	}
	signedAssertion, err := s.assertionDB.Sign(asserts.SnapRevisionType, headers, nil, s.cfg.RootKey.PublicKey().ID())
	if err != nil {
		logrus.Errorf("failed to sign assertion: %v", err)
		el.Add(cerror.Invalid, fmt.Sprintf("failed to sign assertion: %s", err))
		return &proto.SnapRevisionAssertionResponse{
			Errors: el.ConvertToProtoErrorList()}, nil
	}

	signature := string(asserts.Encode(signedAssertion))

	snapRevisionAssertion, cerr := s.repo.AddSnapRevisionAssertion(
		el,
		s.cfg.AuthorityID,
		req.GetSnapSha3_384(),
		s.cfg.RootKey.PublicKey().ID(), // this is the sign_key_SHA3_384
		parsedDeveloperId,
		parsedSnapEntryId,
		req.GetSnapRevisionSequenceNumber(),
		req.GetSnapSize(),
		req.GetTimestamp().AsTime(),
		signature,
	)
	if cerr != nil {
		// should have been logged and added to error list in repo function
		return nil, fmt.Errorf("failed to add snap revision assertion: %v", cerr)
	}

	return &proto.SnapRevisionAssertionResponse{
		Id:                         snapRevisionAssertion.ID.String(),
		AuthorityId:                snapRevisionAssertion.AuthorityID,
		SignKeySha3_384:            snapRevisionAssertion.SignKeySHA3_384,
		SnapEntryId:                snapRevisionAssertion.SnapEntryID.String(),
		SnapSha3_384:               snapRevisionAssertion.SnapSHA3_384,
		SnapSize:                   snapRevisionAssertion.SnapSize,
		Timestamp:                  timestamppb.New(snapRevisionAssertion.Timestamp),
		SnapRevisionSequenceNumber: snapRevisionAssertion.SnapRevisionSequenceNumber,
		DeveloperId:                snapRevisionAssertion.DeveloperID.String(),
		Type:                       snapRevisionAssertion.Type,
		Signature:                  signature,
		Errors:                     el.ConvertToProtoErrorList(),
	}, nil
}

func (s *AssertionService) AddAccountKeyAssertion(ctx context.Context, req *proto.AddAccountKeyAssertionRequest) (*proto.AccountKeyAssertionResponse, error) {
	el := cerror.NewErrorList()
	parsedAccountId, err := uuid.Parse(req.GetAccountId())
	if err != nil {
		cerr := cerror.NewCustomError(cerror.InvalidField, fmt.Sprintf("failed to parse account id: %s", err))
		logrus.Error(cerr)
		el.AddCustomError(cerr)
		return &proto.AccountKeyAssertionResponse{
			Errors: el.ConvertToProtoErrorList(),
		}, nil
	}

	var sequenceNumber uint32
	latestAccountKeyAssertion, cerr := s.repo.GetLatestAccountKeyAssertion(el, parsedAccountId)
	if cerr != nil && cerr.GetCode() != cerror.ResourceNotFound {
		// should have been logged and added to error list in repo function
		return &proto.AccountKeyAssertionResponse{
			Errors: el.ConvertToProtoErrorList(),
		}, nil
	} else if cerr.GetCode() == cerror.ResourceNotFound {
		// remove ResourceNotFound since its not a real error in this case
		el.RemoveErrorWithCode(cerror.ResourceNotFound)
		sequenceNumber = 1
	} else {
		sequenceNumber = latestAccountKeyAssertion.RevisionSequenceNumber + 1
	}

	decodedPublicKey, err := base64.StdEncoding.DecodeString(req.GetEncodedPublicKey())
	if err != nil {
		cerr := cerror.NewCustomError(cerror.Invalid, fmt.Sprintf("failed to decode public key: %s", err))
		logrus.Error(cerr)
		el.AddCustomError(cerr)
		return &proto.AccountKeyAssertionResponse{
			Errors: el.ConvertToProtoErrorList(),
		}, nil
	}
	pubKey, err := asserts.DecodePublicKey(decodedPublicKey)
	if err != nil {
		cerr := cerror.NewCustomError(cerror.Invalid, fmt.Sprintf("failed to decode public key: %s", err))
		logrus.Error(cerr)
		el.AddCustomError(cerr)
		return &proto.AccountKeyAssertionResponse{
			Errors: el.ConvertToProtoErrorList(),
		}, nil
	}
	bodyBytes, err := asserts.EncodePublicKey(pubKey)
	if err != nil {
		cerr := cerror.NewCustomError(cerror.Invalid, fmt.Sprintf("failed to encode public key: %s", err))
		logrus.Error(cerr)
		el.AddCustomError(cerr)
		return &proto.AccountKeyAssertionResponse{
			Errors: el.ConvertToProtoErrorList(),
		}, nil
	}

	headers := map[string]any{
		"authority-id":        s.cfg.AuthorityID,
		"revision":            fmt.Sprintf("%d", sequenceNumber),
		"public-key-sha3-384": req.GetPublicKeySha3_384(),
		"account-id":          req.GetAccountId(),
		"name":                req.GetName(),
		"since":               req.GetSince().AsTime().Format(time.RFC3339),
		"until":               req.GetSince().AsTime().Add(time.Duration(365 * 24 * time.Hour)).Format(time.RFC3339), // a key is valid for 1 year
		// The 'sign-key-sha3-384' header is generated during signing.
	}
	signedAssertion, err := s.assertionDB.Sign(asserts.AccountKeyType, headers, bodyBytes, s.cfg.RootKey.PublicKey().ID())
	if err != nil {
		cerr := cerror.NewCustomError(cerror.Invalid, fmt.Sprintf("failed to sign account key assertion: %s", err))
		logrus.Error(cerr)
		el.AddCustomError(cerr)
		return &proto.AccountKeyAssertionResponse{
			Errors: el.ConvertToProtoErrorList(),
		}, nil
	}

	signature := string(asserts.Encode(signedAssertion))

	// NOTE: maybe also store encoded public key?
	accountKeyAssertion, cerr := s.repo.AddAccountKeyAssertion(
		el,
		s.cfg.AuthorityID,
		req.GetPublicKeySha3_384(),
		s.cfg.RootKey.PublicKey().ID(), // this is the sign_key_SHA3_384
		req.GetName(),
		sequenceNumber,
		parsedAccountId,
		req.GetSince().AsTime(),
		req.GetUntil().AsTime(),
		bodyBytes,
		uint64(len(bodyBytes)),
		signature,
	)
	if cerr != nil {
		// should have been logged and added to error list in repo function
		return &proto.AccountKeyAssertionResponse{
			Errors: el.ConvertToProtoErrorList(),
		}, nil
	}

	return &proto.AccountKeyAssertionResponse{
		Id:                     accountKeyAssertion.ID.String(),
		AuthorityId:            accountKeyAssertion.AuthorityID,
		PublicKeySha3_384:      accountKeyAssertion.PublicKeySHA3_384,
		SignKeySha3_384:        accountKeyAssertion.SignKeySHA3_384,
		AccountId:              accountKeyAssertion.AccountID.String(),
		Name:                   accountKeyAssertion.Name,
		RevisionSequenceNumber: accountKeyAssertion.RevisionSequenceNumber,
		Since:                  timestamppb.New(accountKeyAssertion.Since),
		Until:                  timestamppb.New(accountKeyAssertion.Since.Add(time.Duration(365 * 24 * time.Hour))), // a key is valid for 1 year
		Body:                   accountKeyAssertion.Body,
		BodyLength:             accountKeyAssertion.BodyLength,
		Signature:              signature,
		Type:                   accountKeyAssertion.Type,
		Errors:                 el.ConvertToProtoErrorList(),
	}, nil
}

// ################### GETTERS #####################

func (s *AssertionService) GetSnapRevisionAssertionBySHA3_384(ctx context.Context, req *proto.GetSnapRevisionAssertionBySHA3_384Request) (*proto.SnapRevisionAssertionResponse, error) {
	el := cerror.NewErrorList()
	if req.GetSnapSha3_384() == "" {
		el.Add(cerror.Invalid, "snap sha3_384 is required")
		return nil, fmt.Errorf("snap sha3_384 is required")
	}

	snapRevisionAssertion, cerr := s.repo.GetSnapRevisionAssertionBySHA3_384(el, req.GetSnapSha3_384())
	if cerr != nil {
		// should have been logged and added to error list in repo function
		return nil, fmt.Errorf("failed to get snap revision assertion: %v", cerr)
	}

	return &proto.SnapRevisionAssertionResponse{
		Id:                         snapRevisionAssertion.ID.String(),
		AuthorityId:                snapRevisionAssertion.AuthorityID,
		SignKeySha3_384:            snapRevisionAssertion.SignKeySHA3_384,
		SnapEntryId:                snapRevisionAssertion.SnapEntryID.String(),
		SnapSha3_384:               snapRevisionAssertion.SnapSHA3_384,
		SnapSize:                   snapRevisionAssertion.SnapSize,
		Timestamp:                  timestamppb.New(snapRevisionAssertion.Timestamp),
		SnapRevisionSequenceNumber: snapRevisionAssertion.SnapRevisionSequenceNumber,
		DeveloperId:                snapRevisionAssertion.DeveloperID.String(),
		Type:                       snapRevisionAssertion.Type,
		Signature:                  snapRevisionAssertion.Signature,
		Errors:                     el.ConvertToProtoErrorList(),
	}, nil
}

func (s *AssertionService) GetAccountKeyAssertionByName(ctx context.Context, req *proto.GetAccountKeyAssertionByNameRequest) (*proto.AccountKeyAssertionResponse, error) {
	el := cerror.NewErrorList()
	if req.GetName() == "" {
		el.Add(cerror.Invalid, "name is required")
		return nil, fmt.Errorf("name is required")
	}
	accountKeyAssertion, cerr := s.repo.GetAccountKeyAssertionByName(el, req.GetName())
	if cerr != nil {
		// should have been logged and added to error list in repo function
		return nil, fmt.Errorf("failed to get account key assertion: %v", cerr)
	}

	return &proto.AccountKeyAssertionResponse{
		Id:                     accountKeyAssertion.ID.String(),
		AuthorityId:            accountKeyAssertion.AuthorityID,
		SignKeySha3_384:        accountKeyAssertion.SignKeySHA3_384,
		AccountId:              accountKeyAssertion.AccountID.String(),
		Name:                   accountKeyAssertion.Name,
		RevisionSequenceNumber: accountKeyAssertion.RevisionSequenceNumber,
		Since:                  timestamppb.New(accountKeyAssertion.Since),
		Until:                  timestamppb.New(accountKeyAssertion.Until),
		Body:                   accountKeyAssertion.Body,
		BodyLength:             accountKeyAssertion.BodyLength,
		Signature:              accountKeyAssertion.Signature,
		Type:                   accountKeyAssertion.Type,
		Errors:                 el.ConvertToProtoErrorList(),
	}, nil
}

// ####################### SHOULD BE REMOVED #########################

// TODO: remove all this and use better structure
func (s *AssertionService) ProcessSnapBuildAssertion(ctx context.Context, req *proto.SnapBuildAssertionRequest) (*proto.SnapBuildAssertionResponse, error) {
	errList := make([]*cerrorpb.Error, 0)

	if req.Assertion == nil {
		errList = append(errList, &cerrorpb.Error{
			Code:    cerror.MissingField,
			Message: "Assertion field is required",
		})
		return &proto.SnapBuildAssertionResponse{
			Errors: errList,
		}, nil
	}

	assertion := parseAssertion(string(req.Assertion))

	err := validateSnapBuildAssertion(assertion)
	if err != nil {
		errList = append(errList, &cerrorpb.Error{
			Code:    cerror.Invalid,
			Message: "not a valid snap-build assertion: " + err.Error(),
		})
		return &proto.SnapBuildAssertionResponse{
			Errors: errList,
		}, nil
	}

	snapEntryId := assertion["snap-id"]
	parsedUUID, err := uuid.Parse(snapEntryId)
	if err != nil {
		logrus.Errorf("Failed to parse snap-id: %v", err)
		errList = append(errList, &cerrorpb.Error{
			Code:    cerror.Invalid,
			Message: "Invalid snap-id",
		})
		return &proto.SnapBuildAssertionResponse{
			Errors: errList,
		}, nil
	}

	_, err2 := s.repo.AddAssertion(parsedUUID, string(req.Assertion))
	if err2 != nil {
		logrus.Errorf("Failed to create assertion: %v", err2)
		errList = append(errList, &cerrorpb.Error{
			Code:    cerror.AssertionCreationFailed,
			Message: "Failed to create assertion",
		})
		return &proto.SnapBuildAssertionResponse{
			Errors: errList,
		}, nil
	}

	// TODO: add logic to fill in the fields in the response
	// This info will be present in the assertion object
	return &proto.SnapBuildAssertionResponse{
		AuthorityId:     assertion["authority-id"],
		Grade:           assertion["grade"],
		SignKeySha3_384: assertion["sign-key-sha3-384"],
		SnapId:          assertion["snap-id"],
		SnapSha3_384:    assertion["snap-sha3-384"],
		SnapSize:        assertion["snap-size"],
		Timestamp:       assertion["timestamp"],
		Revision:        assertion["revision"],
		Type:            assertion["type"],
		DeveloperId:     assertion["developer-id"],
		Errors:          errList,
	}, nil
}

// TODO: implement this function to check if the assertion is a valid snap-build assertion
// this is a placeholder for now
// because currently no idea how to validate this
func validateSnapBuildAssertion(assertion map[string]string) error {
	requiredFields := []string{
		"type",
		"authority-id",
		"snap-sha3-384",
		"developer-id",
		"grade",
		"snap-id",
		"snap-size",
		"timestamp",
		"revision",
		"sign-key-sha3-384",
	}

	for _, field := range requiredFields {
		if _, ok := assertion[field]; !ok {
			return fmt.Errorf("missing required field: %s", field)
		}
	}

	if assertion["type"] != "snap-build" {
		return fmt.Errorf("invalid type: %s", assertion["type"])
	}

	return nil
}

// parseAssertion parses a string containing key-value pairs separated by colons
// and returns a map where the keys are the parsed keys and the values are the
// parsed values. Each key-value pair should be on a new line.
//
// The function ignores empty lines and lines that start with "AcLB".
//
// Parameters:
//   - data: A string containing the key-value pairs to be parsed.
//
// Returns:
//
//	A map[string]string where the keys are the parsed keys and the values are
//	the parsed values.
func parseAssertion(data string) map[string]string {
	lines := strings.Split(data, "\n")
	result := make(map[string]string)

	for _, line := range lines {
		// Ignore empty lines and signature block
		if line == "" || strings.HasPrefix(line, "AcLB") {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			result[key] = value
		}
	}
	return result
}
