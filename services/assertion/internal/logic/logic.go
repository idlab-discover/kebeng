package logic

import (
	"context"
	"fmt"
	"strings"

	"github.com/idlab-discover/kebeng/services/assertion/internal/config"
	"github.com/idlab-discover/kebeng/services/assertion/internal/errors"
	"github.com/idlab-discover/kebeng/services/assertion/internal/repositories"
	proto "github.com/idlab-discover/kebeng/services/assertion/proto"
	"github.com/sirupsen/logrus"
)

// TODO: maybe call this different?
// this should contain the business logic of the service
// so doing checks and stuff and calling the database logic

type AssertionService struct {
	config *config.Config
	repo   *repositories.AssertionRepository
	proto.UnimplementedAssertionServiceServer
}

func NewAssertionService(repo *repositories.AssertionRepository, config *config.Config) *AssertionService {
	return &AssertionService{repo: repo, config: config}
}

func (s *AssertionService) ProcessSnapBuildAssertion(ctx context.Context, req *proto.SnapBuildAssertionRequest) (*proto.SnapBuildAssertionResponse, error) {
	errList := make([]*proto.Error, 0)

	if req.Assertion == nil {
		errList = append(errList, &proto.Error{
			Code:    errors.MissingField,
			Message: "Assertion field is required",
		})
		return &proto.SnapBuildAssertionResponse{
			Errors: errList,
		}, nil
	}

	assertion := parseAssertion(string(req.Assertion))

	err := validateSnapBuildAssertion(assertion)
	if err != nil {
		errList = append(errList, &proto.Error{
			Code:    errors.Invalid,
			Message: "not a valid snap-build assertion: " + err.Error(),
		})
		return &proto.SnapBuildAssertionResponse{
			Errors: errList,
		}, nil
	}

	_, err = s.repo.AddAssertion(string(req.Assertion))
	if err != nil {
		logrus.Errorf("Failed to create assertion: %v", err)
		errList = append(errList, &proto.Error{
			Code:    errors.AssertionCreationFailed,
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
