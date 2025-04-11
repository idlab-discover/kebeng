package logic

import (
	"context"
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

	signature, cerr := s.signAssertion(el, req, s.cfg.RootKey.PublicKey().ID())
	if cerr != nil {
		return nil, fmt.Errorf("failed to sign assertion: %v", cerr)
	}
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

// ##################### HELPER FUNCTIONS #####################

func (s *AssertionService) signAssertion(el *cerror.ErrorList, assertionHeaders any, signKeyID string) (string, *cerror.CustomError) {
	var headers map[string]any
	var err error
	var signedAssertion asserts.Assertion

	switch a := assertionHeaders.(type) {
	case *proto.AddSnapRevisionAssertionRequest:
		headers = map[string]any{
			"authority-id":  s.cfg.AuthorityID,
			"snap-sha3-384": a.GetSnapSha3_384(),
			"developer-id":  a.GetDeveloperId(),
			"snap-id":       a.GetSnapEntryId(),
			"snap-revision": a.GetSnapRevisionSequenceNumber(),
			"snap-size":     a.GetSnapSize(),
			"timestamp":     a.GetTimestamp(),
			// The 'sign-key-sha3-384' header is generated during signing.
		}
		signedAssertion, err = s.assertionDB.Sign(asserts.SnapRevisionType, headers, nil, signKeyID)
		if err != nil {
			logrus.Errorf("failed to sign assertion: %v", err)
			el.Add(cerror.Invalid, fmt.Sprintf("failed to sign assertion: %s", err))
			return "", cerror.NewCustomError(cerror.Invalid, fmt.Sprintf("failed to sign assertion: %s", err))
		}
	case *proto.AddAccountKeyAssertionRequest:
		headers = map[string]any{
			"authority-id":        s.cfg.AuthorityID,
			"revision":            a.GetSnapRevisionSequenceNumber(),
			"public-key-sha3-384": a.GetPublicKeySha3_384(),
			"account-id":          a.GetAccountId(),
			"name":                a.GetName(),
			"since":               a.GetSince(),
			"until":               a.GetSince().AsTime().Add(time.Duration(365 * 24 * time.Hour)), // a key is valid for 1 year
			// The 'sign-key-sha3-384' header is generated during signing.
		}
		body := []byte(a.Body)
		signedAssertion, err = s.assertionDB.Sign(asserts.AccountKeyType, headers, body, signKeyID)
		if err != nil {
			logrus.Errorf("failed to sign account key assertion: %v", err)
			el.Add(cerror.Invalid, fmt.Sprintf("failed to sign account key assertion: %s", err))
			return "", cerror.NewCustomError(cerror.Invalid, fmt.Sprintf("failed to sign account key assertion: %s", err))
		}
	default:
		err := fmt.Errorf("unsupported assertion type: %T", assertionHeaders)
		el.Add(cerror.ResourceNotFound, err.Error())
		return "", cerror.NewCustomError(cerror.ResourceNotFound, err.Error())
	}

	assertionBytes := asserts.Encode(signedAssertion)
	return string(assertionBytes), nil
}

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
