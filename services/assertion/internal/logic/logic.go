package logic

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/idlab-discover/kebeng/services/assertion/internal/config"
	"github.com/idlab-discover/kebeng/services/assertion/internal/model"
	"github.com/idlab-discover/kebeng/services/assertion/internal/repository"
	proto "github.com/idlab-discover/kebeng/services/assertion/proto"

	"github.com/google/uuid"
	cerror "github.com/idlab-discover/kebeng/common/cerror"
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
		ctx,
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
	latestAccountKeyAssertion, cerr := s.repo.GetLatestAccountKeyAssertion(ctx, el, parsedAccountId)
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
		"public-key-sha3-384": req.GetPublicKeySha3_384Encoded(),
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
		ctx,
		el,
		s.cfg.AuthorityID,
		req.GetPublicKeySha3_384Encoded(),
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
		Id:                       accountKeyAssertion.ID.String(),
		AuthorityId:              accountKeyAssertion.AuthorityID,
		PublicKeySha3_384Encoded: accountKeyAssertion.PublicKeySha3_384Encoded,
		SignKeySha3_384:          accountKeyAssertion.SignKeySHA3_384,
		AccountId:                accountKeyAssertion.AccountID.String(),
		Name:                     accountKeyAssertion.Name,
		RevisionSequenceNumber:   accountKeyAssertion.RevisionSequenceNumber,
		Since:                    timestamppb.New(accountKeyAssertion.Since),
		Until:                    timestamppb.New(accountKeyAssertion.Since.Add(time.Duration(365 * 24 * time.Hour))), // a key is valid for 1 year
		Body:                     accountKeyAssertion.Body,
		BodyLength:               accountKeyAssertion.BodyLength,
		Signature:                signature,
		Type:                     accountKeyAssertion.Type,
		Errors:                   el.ConvertToProtoErrorList(),
	}, nil
}

func (s *AssertionService) AddSnapDeclarationAssertion(ctx context.Context, req *proto.AddSnapDeclarationAssertionRequest) (*proto.SnapDeclarationAssertionResponse, error) {
	el := cerror.NewErrorList()

	var sequenceNumber uint32
	latestSnapDeclarationAssertion, cerr := s.repo.GetLatestSnapDeclarationAssertion(ctx, el, req.GetSnapId())
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
		"series":       req.GetSeries(),
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
		ctx,
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
		deserializePlugs(req.GetPlugs()),
		deserializeSlots(req.GetSlots()),
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

func (s *AssertionService) AddSnapBuildAssertion(ctx context.Context, req *proto.AddSnapBuildAssertionRequest) (*proto.SnapBuildAssertionResponse, error) {
	el := cerror.NewErrorList()
	parsedSnapId, err := uuid.Parse(req.GetSnapEntryId())
	if err != nil {
		cerr := cerror.NewCustomError(cerror.Invalid, fmt.Sprintf("failed to parse snap id: %s", err))
		logrus.Error(cerr)
		el.AddCustomError(cerr)
		return &proto.SnapBuildAssertionResponse{
			Errors: el.ConvertToProtoErrorList(),
		}, nil
	}
	parsedAccountId, err := uuid.Parse(req.GetDeveloperId())
	if err != nil {
		cerr := cerror.NewCustomError(cerror.Invalid, fmt.Sprintf("failed to parse developer id: %s", err))
		logrus.Error(cerr)
		el.AddCustomError(cerr)
		return &proto.SnapBuildAssertionResponse{
			Errors: el.ConvertToProtoErrorList(),
		}, nil
	}

	timestamp := time.Now().Format(time.RFC3339)
	parsedTimestamp, err := time.Parse(time.RFC3339, timestamp)
	if err != nil {
		cerr := cerror.NewCustomError(cerror.Invalid, fmt.Sprintf("failed to parse timestamp: %s", err))
		logrus.Error(cerr)
		el.AddCustomError(cerr)
		return &proto.SnapBuildAssertionResponse{
			Errors: el.ConvertToProtoErrorList(),
		}, nil
	}

	headers := map[string]any{
		"authority-id":      s.cfg.AuthorityID,
		"snap-id":           req.GetSnapEntryId(),
		"developer-id":      req.GetDeveloperId(),
		"snap-size":         fmt.Sprintf("%d", req.GetSnapSize()),
		"snap-sha3-384":     req.GetSha3_384Encoded(),
		"grade":             req.GetGrade(),
		"timestamp":         timestamp,
		"sign-key-sha3-384": req.GetSignKeySha3_384Encoded(),
	}
	signedAssertion, err := s.assertionDB.Sign(asserts.SnapBuildType, headers, nil, s.cfg.RootKey.PublicKey().ID())
	if err != nil {
		cerr := cerror.NewCustomError(cerror.Invalid, fmt.Sprintf("failed to sign snap build assertion: %s", err))
		logrus.Error(cerr)
		el.AddCustomError(cerr)
		return &proto.SnapBuildAssertionResponse{
			Errors: el.ConvertToProtoErrorList(),
		}, nil
	}
	signature := string(asserts.Encode(signedAssertion))

	snapBuildAssertion, cerr := s.repo.AddSnapBuildAssertion(
		ctx,
		el,
		s.cfg.AuthorityID,
		s.cfg.RootKey.PublicKey().ID(), // this is the sign_key_SHA3_384
		parsedSnapId,
		parsedAccountId,
		req.GetGrade(),
		req.GetSha3_384Encoded(),
		req.GetSnapSize(),
		signature,
		parsedTimestamp,
	)
	if cerr != nil {
		// should have been logged and added to error list in repo function
		return &proto.SnapBuildAssertionResponse{
			Errors: el.ConvertToProtoErrorList(),
		}, nil
	}
	snapBuildAssertion.Type = asserts.SnapBuildType.Name
	return &proto.SnapBuildAssertionResponse{
		Id:                     snapBuildAssertion.ID.String(),
		AuthorityId:            snapBuildAssertion.AuthorityID,
		SignKeySha3_384Encoded: snapBuildAssertion.SignKeySHA3_384,
		SnapEntryId:            snapBuildAssertion.SnapEntryID.String(),
		DeveloperId:            snapBuildAssertion.DeveloperID.String(),
		SnapSize:               snapBuildAssertion.SnapSize,
		Sha3_384Encoded:        snapBuildAssertion.SnapSHA3_384,
		Grade:                  snapBuildAssertion.Grade,
		Timestamp:              timestamppb.New(snapBuildAssertion.Timestamp),
		Type:                   snapBuildAssertion.Type,
		Signature:              snapBuildAssertion.Signature,
		Errors:                 el.ConvertToProtoErrorList(),
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
	latestAccountAssertion, cerr := s.repo.GetLatestAccountAssertionByAccountID(ctx, el, parsedAccountId)
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
		sequenceNumber = latestAccountAssertion.Revision + 1
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
		ctx,
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

	snapRevisionAssertion, cerr := s.repo.GetSnapRevisionAssertionBySHA3_384(ctx, el, req.GetSnapSha3_384())
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

func (s *AssertionService) GetAccountKeyAssertionByPublicKeySha(ctx context.Context, req *proto.GetAccountKeyAssertionByPublicKeyShaRequest) (*proto.AccountKeyAssertionResponse, error) {
	el := cerror.NewErrorList()
	if req.GetPublicKeySha3_384Encoded() == "" {
		el.Add(cerror.Invalid, "name is required")
		return nil, fmt.Errorf("name is required")
	}
	accountKeyAssertion, cerr := s.repo.GetAccountKeyAssertionByPublicKeySha(ctx, el, req.GetPublicKeySha3_384Encoded())
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
	snapDeclarationAssertion, cerr := s.repo.GetSnapDeclarationAssertionBySnapID(ctx, el, req.GetSnapId())
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
		Plugs:           serializeMap(snapDeclarationAssertion.Plugs),
		Slots:           serializeMap(snapDeclarationAssertion.Slots),
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
	accountAssertion, cerr := s.repo.GetAccountAssertionByAccountID(ctx, el, parsedAccountId)
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

// ############### HELPER FUNCTIONS #################

// convert proto Alias to model Alias
func protoAliasToModelAlias(protoAliases []*proto.Alias) []model.Alias {
	if len(protoAliases) == 0 {
		return nil
	}
	aliases := make([]model.Alias, len(protoAliases))
	for i, protoAlias := range protoAliases {
		aliases[i] = model.Alias{
			Name:   protoAlias.Name,
			Target: protoAlias.Target,
		}
	}
	return aliases
}

// convert model Alias to proto Alias
func modelAliasToProtoAlias(modelAliases []model.Alias) []*proto.Alias {
	if len(modelAliases) == 0 {
		return nil
	}
	protoAliases := make([]*proto.Alias, len(modelAliases))
	for i, modelAlias := range modelAliases {
		protoAliases[i] = &proto.Alias{
			Name:   modelAlias.Name,
			Target: modelAlias.Target,
		}
	}
	return protoAliases
}

func serializeMap[T model.Plugs | model.Slots](m T) string {
	if len(m) == 0 {
		return ""
	}
	serialized, err := json.Marshal(m)
	if err != nil {
		logrus.Errorf("failed to serialize: %v", err)
		return ""
	}
	return string(serialized)
}

// deserializePlugs converts a JSON string into a model.Plugs object.
func deserializePlugs(plugs string) model.Plugs {
	var deserialized model.Plugs
	err := json.Unmarshal([]byte(plugs), &deserialized)
	if err != nil {
		logrus.Errorf("failed to deserialize plugs: %v", err)
		return nil
	}
	return deserialized
}

// DeserializeSlot converts a JSON string into a model.Slots object.
func deserializeSlots(slots string) model.Slots {
	var deserialized model.Slots
	err := json.Unmarshal([]byte(slots), &deserialized)
	if err != nil {
		logrus.Errorf("failed to deserialize slots: %v", err)
		return nil
	}
	return deserialized
}
