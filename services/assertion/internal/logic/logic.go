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
	"github.com/idlab-discover/kebeng/services/assertion/internal/model"
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
	if el.HasError() {
		return &proto.SnapRevisionAssertionResponse{
			Errors: el.ConvertToProtoErrorList(),
		}, nil
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
		cerr := cerror.NewCustomError(cerror.Invalid, fmt.Sprintf("failed to sign snapRevisionAssertion: %s", err))
		logrus.Error(cerr)
		el.AddCustomError(cerr)
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
	snapRevisionAssertion.Type = asserts.SnapRevisionType.Name

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
	accountKeyAssertion.Type = asserts.AccountKeyType.Name

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

func (s *AssertionService) AddSnapDeclarationAssertion(ctx context.Context, req *proto.AddSnapDeclarationAssertionRequest) (*proto.SnapDeclarationAssertionResponse, error) {
	el := cerror.NewErrorList()

	var sequenceNumber uint32
	latestSnapDeclarationAssertion, cerr := s.repo.GetLatestSnapDeclarationAssertion(el, req.GetSnapId())
	if cerr != nil && cerr.GetCode() != cerror.ResourceNotFound {
		// should have been logged and added to error list in repo function
		return &proto.SnapDeclarationAssertionResponse{
			Errors: el.ConvertToProtoErrorList(),
		}, nil
	} else if cerr.GetCode() == cerror.ResourceNotFound {
		// remove ResourceNotFound since its not a real error in this case
		el.RemoveErrorWithCode(cerror.ResourceNotFound)
		sequenceNumber = 1
	} else {
		sequenceNumber = latestSnapDeclarationAssertion.Revision + 1
	}

	headers := map[string]any{
		"authority-id": s.cfg.AuthorityID,
		"revision":     fmt.Sprintf("%d", sequenceNumber),
		"series":       fmt.Sprintf("%d", req.GetSeries()),
		"snap-id":      req.GetSnapId(),
		"snap-name":    req.GetSnapName(),
		"publisher-id": req.GetPublisherId(),
		"timestamp":    req.GetTimestamp().AsTime().Format(time.RFC3339),
	}
	signedAssertion, err := s.assertionDB.Sign(asserts.SnapDeclarationType, headers, nil, s.cfg.RootKey.PublicKey().ID())
	if err != nil {
		cerr := cerror.NewCustomError(cerror.Invalid, fmt.Sprintf("failed to sign snap declaration assertion: %s", err))
		logrus.Error(cerr)
		el.AddCustomError(cerr)
		return &proto.SnapDeclarationAssertionResponse{
			Errors: el.ConvertToProtoErrorList(),
		}, nil
	}

	signature := string(asserts.Encode(signedAssertion))
	snapDeclarationAssertion, cerr := s.repo.AddSnapDeclarationAssertion(
		el,
		s.cfg.AuthorityID,
		s.cfg.RootKey.PublicKey().ID(), // this is the sign_key_SHA3_384
		req.GetSnapId(),
		req.GetSnapName(),
		req.GetPublisherId(),
		sequenceNumber,
		req.GetSeries(),
		req.GetTimestamp().AsTime(),
		req.GetRefreshControl(),
		protoAliasToModelAlias(req.GetAliases()),
		protoPlugToModelPlug(req.GetPlugs()),
		protoSlotToModelSlot(req.GetSlots()),
		signature,
	)
	if cerr != nil {
		// should have been logged and added to error list in repo function
		return &proto.SnapDeclarationAssertionResponse{
			Errors: el.ConvertToProtoErrorList(),
		}, nil
	}
	snapDeclarationAssertion.Type = asserts.SnapDeclarationType.Name
	// convert pq.StringArray to []string
	refreshControl := []string(snapDeclarationAssertion.RefreshControl)

	return &proto.SnapDeclarationAssertionResponse{
		Id:              snapDeclarationAssertion.ID.String(),
		AuthorityId:     snapDeclarationAssertion.AuthorityID,
		SignKeySha3_384: snapDeclarationAssertion.SignKeySHA3_384,
		SnapId:          snapDeclarationAssertion.SnapID,
		SnapName:        snapDeclarationAssertion.SnapName,
		PublisherId:     snapDeclarationAssertion.PublisherID,
		Revision:        snapDeclarationAssertion.Revision,
		Series:          snapDeclarationAssertion.Series,
		Timestamp:       timestamppb.New(snapDeclarationAssertion.Timestamp),
		RefreshControl:  refreshControl,
		Aliases:         req.GetAliases(),
		Plugs:           req.GetPlugs(),
		Slots:           req.GetSlots(),
		Signature:       signature,
		Type:            snapDeclarationAssertion.Type,
		Errors:          el.ConvertToProtoErrorList(),
	}, nil
}

func (s *AssertionService) AddAccountAssertion(ctx context.Context, req *proto.AddAccountAssertionRequest) (*proto.AccountAssertionResponse, error) {
	el := cerror.NewErrorList()
	parsedAccountId, err := uuid.Parse(req.GetAccountId())
	if err != nil {
		cerr := cerror.NewCustomError(cerror.Invalid, fmt.Sprintf("failed to parse account id: %s", err))
		logrus.Error(cerr)
		el.AddCustomError(cerr)
		return &proto.AccountAssertionResponse{
			Errors: el.ConvertToProtoErrorList(),
		}, nil
	}

	var sequenceNumber uint32
	latestSnapDeclarationAssertion, cerr := s.repo.GetLatestAccountAssertionByAccountID(el, parsedAccountId)
	if cerr != nil && cerr.GetCode() != cerror.ResourceNotFound {
		// should have been logged and added to error list in repo function
		return &proto.AccountAssertionResponse{
			Errors: el.ConvertToProtoErrorList(),
		}, nil
	} else if cerr.GetCode() == cerror.ResourceNotFound {
		// remove ResourceNotFound since its not a real error in this case
		el.RemoveErrorWithCode(cerror.ResourceNotFound)
		sequenceNumber = 1
	} else {
		sequenceNumber = latestSnapDeclarationAssertion.Revision + 1
	}

	headers := map[string]any{
		"authority-id": s.cfg.AuthorityID,
		"account-id":   req.GetAccountId(),
		"display-name": req.GetDisplayName(),
		"timestamp":    req.GetTimestamp().AsTime().Format(time.RFC3339),
		"revision":     fmt.Sprintf("%d", sequenceNumber),
		"validation":   req.GetValidation(),
		// The 'sign-key-sha3-384' header is generated during signing.
	}
	signedAssertion, err := s.assertionDB.Sign(asserts.AccountType, headers, nil, s.cfg.RootKey.PublicKey().ID())
	if err != nil {
		cerr := cerror.NewCustomError(cerror.Invalid, fmt.Sprintf("failed to sign account assertion: %s", err))
		logrus.Error(cerr)
		el.AddCustomError(cerr)
		return &proto.AccountAssertionResponse{
			Errors: el.ConvertToProtoErrorList(),
		}, nil
	}
	signature := string(asserts.Encode(signedAssertion))
	accountAssertion, cerr := s.repo.AddAccountAssertion(
		el,
		s.cfg.AuthorityID,
		req.GetDisplayName(),
		req.GetUsername(),
		req.GetValidation(),
		parsedAccountId,
		sequenceNumber,
		req.GetTimestamp().AsTime(),
		s.cfg.RootKey.PublicKey().ID(), // this is the sign_key_SHA3_384
		signature,
	)
	if cerr != nil {
		// should have been logged and added to error list in repo function
		return &proto.AccountAssertionResponse{
			Errors: el.ConvertToProtoErrorList(),
		}, nil
	}
	accountAssertion.Type = asserts.AccountType.Name
	return &proto.AccountAssertionResponse{
		Id:              accountAssertion.ID.String(),
		AuthorityId:     accountAssertion.AuthorityID,
		SignKeySha3_384: accountAssertion.SignKeySHA3_384,
		AccountId:       accountAssertion.AccountID.String(),
		Timestamp:       timestamppb.New(accountAssertion.Timestamp),
		Type:            accountAssertion.Type,
		Signature:       signature,
		Errors:          el.ConvertToProtoErrorList(),
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
	snapRevisionAssertion.Type = asserts.SnapRevisionType.Name

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
	accountKeyAssertion.Type = asserts.AccountKeyType.Name

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

func (s *AssertionService) GetSnapDeclarationAssertionBySnapID(ctx context.Context, req *proto.GetSnapDeclarationAssertionBySnapIDRequest) (*proto.SnapDeclarationAssertionResponse, error) {
	el := cerror.NewErrorList()
	if req.GetSnapId() == "" {
		cerr := cerror.NewCustomError(cerror.Invalid, "snap id is required")
		logrus.Error(cerr)
		el.AddCustomError(cerr)
		return &proto.SnapDeclarationAssertionResponse{
			Errors: el.ConvertToProtoErrorList(),
		}, nil
	}
	snapDeclarationAssertion, cerr := s.repo.GetSnapDeclarationAssertionBySnapID(el, req.GetSnapId())
	if cerr != nil {
		// should have been logged and added to error list in repo function
		return &proto.SnapDeclarationAssertionResponse{
			Errors: el.ConvertToProtoErrorList(),
		}, nil
	}
	snapDeclarationAssertion.Type = asserts.SnapDeclarationType.Name
	refreshControl := []string(snapDeclarationAssertion.RefreshControl)
	return &proto.SnapDeclarationAssertionResponse{
		Id:              snapDeclarationAssertion.ID.String(),
		AuthorityId:     snapDeclarationAssertion.AuthorityID,
		SignKeySha3_384: snapDeclarationAssertion.SignKeySHA3_384,
		SnapId:          snapDeclarationAssertion.SnapID,
		SnapName:        snapDeclarationAssertion.SnapName,
		PublisherId:     snapDeclarationAssertion.PublisherID,
		Revision:        snapDeclarationAssertion.Revision,
		Series:          snapDeclarationAssertion.Series,
		Timestamp:       timestamppb.New(snapDeclarationAssertion.Timestamp),
		RefreshControl:  refreshControl,
		Aliases:         modelAliasToProtoAlias(snapDeclarationAssertion.Aliases),
		Plugs:           modelPlugToProtoPlug(snapDeclarationAssertion.Plugs),
		Slots:           modelSlotToProtoSlot(snapDeclarationAssertion.Slots),
		Signature:       snapDeclarationAssertion.Signature,
		Type:            snapDeclarationAssertion.Type,
		Errors:          el.ConvertToProtoErrorList(),
	}, nil
}

func (s *AssertionService) GetAccountAssertionByAccountID(ctx context.Context, req *proto.GetAccountAssertionByAccountIDRequest) (*proto.AccountAssertionResponse, error) {
	el := cerror.NewErrorList()
	if req.GetAccountId() == "" {
		cerr := cerror.NewCustomError(cerror.Invalid, "account id is required")
		logrus.Error(cerr)
		el.AddCustomError(cerr)
		return &proto.AccountAssertionResponse{
			Errors: el.ConvertToProtoErrorList(),
		}, nil
	}
	parsedAccountId, err := uuid.Parse(req.GetAccountId())
	if err != nil {
		cerr := cerror.NewCustomError(cerror.Invalid, fmt.Sprintf("failed to parse account id: %s", err))
		logrus.Error(cerr)
		el.AddCustomError(cerr)
		return &proto.AccountAssertionResponse{
			Errors: el.ConvertToProtoErrorList(),
		}, nil
	}
	accountAssertion, cerr := s.repo.GetAccountAssertionByAccountID(el, parsedAccountId)
	if cerr != nil {
		// should have been logged and added to error list in repo function
		return &proto.AccountAssertionResponse{
			Errors: el.ConvertToProtoErrorList(),
		}, nil
	}
	accountAssertion.Type = asserts.AccountType.Name
	return &proto.AccountAssertionResponse{
		Id:              accountAssertion.ID.String(),
		AuthorityId:     accountAssertion.AuthorityID,
		SignKeySha3_384: accountAssertion.SignKeySHA3_384,
		AccountId:       accountAssertion.AccountID.String(),
		Timestamp:       timestamppb.New(accountAssertion.Timestamp),
		Type:            accountAssertion.Type,
		Signature:       accountAssertion.Signature,
		DisplayName:     accountAssertion.DisplayName,
		Username:        accountAssertion.Username,
		Validation:      accountAssertion.Validation,
		Revision:        accountAssertion.Revision,
		Errors:          el.ConvertToProtoErrorList(),
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

// ############### HELPER FUNCTIONS #################

// convert proto Alias to model Alias
func protoAliasToModelAlias(protoAliases []*proto.Alias) []model.Alias {
	aliases := make([]model.Alias, len(protoAliases))
	for i, protoAlias := range protoAliases {
		aliases[i] = model.Alias{
			Name:   protoAlias.Name,
			Target: protoAlias.Target,
		}
	}
	return aliases
}

// convert proto Plug to model Plug
func protoPlugToModelPlug(protoPlug map[string]*proto.PlugRule) map[string]*model.Plug {
	plugs := make(map[string]*model.Plug)
	for k, v := range protoPlug {
		plugs[k] = &model.Plug{
			AllowInstallation:   v.AllowInstallation,
			DenyInstallation:    v.DenyInstallation,
			AllowConnection:     v.AllowConnection,
			DenyConnection:      v.DenyConnection,
			AllowAutoConnection: v.AllowAutoConnection,
			DenyAutoConnection:  v.DenyAutoConnection,
		}
	}
	return plugs
}

// convert proto Slot to model Slot
func protoSlotToModelSlot(protoSlot map[string]*proto.SlotRule) map[string]*model.Slot {
	slots := make(map[string]*model.Slot)
	for k, v := range protoSlot {
		slots[k] = &model.Slot{
			AllowInstallation:   v.AllowInstallation,
			DenyInstallation:    v.DenyInstallation,
			AllowConnection:     v.AllowConnection,
			DenyConnection:      v.DenyConnection,
			AllowAutoConnection: v.AllowAutoConnection,
			DenyAutoConnection:  v.DenyAutoConnection,
		}
	}
	return slots
}

// convert model Alias to proto Alias
func modelAliasToProtoAlias(modelAliases []model.Alias) []*proto.Alias {
	protoAliases := make([]*proto.Alias, len(modelAliases))
	for i, modelAlias := range modelAliases {
		protoAliases[i] = &proto.Alias{
			Name:   modelAlias.Name,
			Target: modelAlias.Target,
		}
	}
	return protoAliases
}

// convert model Plug to proto Plug
func modelPlugToProtoPlug(modelPlugs map[string]*model.Plug) map[string]*proto.PlugRule {
	protoPlugs := make(map[string]*proto.PlugRule)
	for k, v := range modelPlugs {
		protoPlugs[k] = &proto.PlugRule{
			AllowInstallation:   v.AllowInstallation,
			DenyInstallation:    v.DenyInstallation,
			AllowConnection:     v.AllowConnection,
			DenyConnection:      v.DenyConnection,
			AllowAutoConnection: v.AllowAutoConnection,
			DenyAutoConnection:  v.DenyAutoConnection,
		}
	}
	return protoPlugs
}

// convert model Slot to proto Slot
func modelSlotToProtoSlot(modelSlots map[string]*model.Slot) map[string]*proto.SlotRule {
	protoSlots := make(map[string]*proto.SlotRule)
	for k, v := range modelSlots {
		protoSlots[k] = &proto.SlotRule{
			AllowInstallation:   v.AllowInstallation,
			DenyInstallation:    v.DenyInstallation,
			AllowConnection:     v.AllowConnection,
			DenyConnection:      v.DenyConnection,
			AllowAutoConnection: v.AllowAutoConnection,
			DenyAutoConnection:  v.DenyAutoConnection,
		}
	}
	return protoSlots
}
