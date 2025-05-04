package client

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/idlab-discover/kebeng/services/assertion/client/model"
	storeModel "github.com/idlab-discover/kebeng/services/store/client/model"
	"github.com/idlab-discover/kebeng/services/assertion/internal/config"
	proto "github.com/idlab-discover/kebeng/services/assertion/proto"

	"github.com/google/uuid"
	cerror "github.com/idlab-discover/kebeng/common/cerror"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	maxDialAttempts = 5
	dialRetryDelay  = 2 * time.Second
	healthTimeout   = 2 * time.Second
)

type AssertionClientInterface interface {
	AddAccountKeyAssertion(encoded_public_key, publicKeySha3_384Encoded, accountId, name string, since time.Time, until time.Time) *proto.AccountKeyAssertionResponse
	AddSnapBuildAssertion(sha3_384Encoded, grade, signKeySha3_384Encoded string, developerId, snapEntryId uuid.UUID, size uint64) *proto.SnapBuildAssertionResponse
	AddSnapRevisionAssertion(snapSha3_384 string, developerId string, snapEntryId string, snapRevisionSequenceNumber uint32, snapSize uint64) *proto.SnapRevisionAssertionResponse
	AddSnapDeclarationAssertion(snapID, snapName, publisherID string, series string, refreshControl []string, aliases []model.Alias, plugs storeModel.Plugs, slots storeModel.Slots) *proto.SnapDeclarationAssertionResponse
	AddAccountAssertion(accountId, displayName, username, validation string, timestamp time.Time) *proto.AccountAssertionResponse

	GetAccountKeyAssertionByPublicKeySha(publicKeySha3_384Encoded string) *proto.AccountKeyAssertionResponse
	GetSnapRevisionAssertionBySHA3_384(snapSha3_384 string) *proto.SnapRevisionAssertionResponse
	GetSnapDeclarationAssertionBySnapID(snapId string) *proto.SnapDeclarationAssertionResponse
	GetAccountAssertionByAccountID(accountId string) *proto.AccountAssertionResponse

	Close()
}

var _ AssertionClientInterface = (*AssertionClient)(nil)

type AssertionClient struct {
	conn   *grpc.ClientConn
	client proto.AssertionServiceClient
}

func NewAssertionClient(assertionHost string, assertionPort int) (*AssertionClient, *cerror.CustomError) {
	logrus.Infof("Connecting to assertion service at %s:%d", assertionHost, assertionPort)
	addr := config.GetAssertionServiceAddress(assertionHost, assertionPort)

	for attempt := 1; attempt <= maxDialAttempts; attempt++ {
		logrus.Infof("Attempt %d/%d: dialing assertion service at %s", attempt, maxDialAttempts, addr)
		conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			logrus.Warnf("Failed to create gRPC client: %v", err)
			time.Sleep(dialRetryDelay)
			continue
		}

		// run quick health check
		hc := grpc_health_v1.NewHealthClient(conn)
		ctx, cancel := context.WithTimeout(context.Background(), healthTimeout)
		defer cancel()

		resp, err := hc.Check(ctx, &grpc_health_v1.HealthCheckRequest{})
		if err == nil && resp.Status == grpc_health_v1.HealthCheckResponse_SERVING {
			logrus.Infof("Successfully connected to assertion service at %s", addr)
			return &AssertionClient{conn, proto.NewAssertionServiceClient(conn)}, nil
		}
		logrus.Warnf("Health check failed: %v", err)
		time.Sleep(dialRetryDelay)
	}
	return nil, cerror.NewCustomError(cerror.InternalServerError, fmt.Sprintf("failed to connect to assertion service after %d attempts", maxDialAttempts))
}

func (c *AssertionClient) Close() {
	err := c.conn.Close()
	if err != nil {
		logrus.Errorf("error closing connection: %v", err)
	}
}

func (c *AssertionClient) AddAccountKeyAssertion(encoded_public_key, publicKeySha3_384Encoded, accountId, name string, since time.Time, until time.Time) *proto.AccountKeyAssertionResponse {
	el := cerror.NewErrorList()

	// check input
	if encoded_public_key == "" {
		el.Add(cerror.InvalidField, "encoded public key is required")
	}
	if publicKeySha3_384Encoded == "" {
		el.Add(cerror.InvalidField, "public key sha3_384 encoded is required")
	}
	if accountId == "" {
		el.Add(cerror.InvalidField, "account id is required")
	}
	if _, err := uuid.Parse(accountId); accountId != "" && err != nil {
		el.Add(cerror.InvalidField, "account id is not a valid uuid")
	}
	if name == "" {
		el.Add(cerror.InvalidField, "name is required")
	}
	// doesn't allow spaces
	if strings.Contains(name, " ") || strings.Contains(name, "_") {
		el.Add(cerror.InvalidField, fmt.Sprintf("name cannot contain spaces or _ in name: '%s'", name))
	}
	if strings.ToLower(name) != name {
		el.Add(cerror.InvalidField, fmt.Sprintf("name must be lowercase: '%s'", name))
	}
	if since.IsZero() {
		el.Add(cerror.InvalidField, "since is required")
	}
	if el.HasError() {
		return &proto.AccountKeyAssertionResponse{
			Errors: el.ConvertToProtoErrorList(),
		}
	}

	if until.IsZero() {
		// set to one year further than since
		t := since.AddDate(1, 0, 0)
		until = t
	}

	req := &proto.AddAccountKeyAssertionRequest{
		EncodedPublicKey:         encoded_public_key,
		PublicKeySha3_384Encoded: publicKeySha3_384Encoded,
		AccountId:                accountId,
		Name:                     name,
		Since:                    timestamppb.New(since),
		Until:                    timestamppb.New(until),
	}

	resp, err := c.client.AddAccountKeyAssertion(context.Background(), req)
	if err != nil {
		// this means proto request failed not the actual logic
		el.Add(cerror.InternalServerError, err.Error())
		resp = &proto.AccountKeyAssertionResponse{
			Errors: el.ConvertToProtoErrorList(),
		}
	}
	return resp
}

func (c *AssertionClient) AddSnapRevisionAssertion(snapSha3_384, developerId, snapEntryId string, snapRevisionSequenceNumber uint32, snapSize uint64) *proto.SnapRevisionAssertionResponse {
	el := cerror.NewErrorList()

	// check input
	if snapSha3_384 == "" {
		el.Add(cerror.InvalidField, "snap sha3_384 is required")
	}
	if developerId == "" {
		el.Add(cerror.InvalidField, "developer id is required")
	}
	if _, err := uuid.Parse(developerId); developerId != "" && err != nil {
		el.Add(cerror.InvalidField, "developer id is not a valid uuid")
	}
	if snapEntryId == "" {
		el.Add(cerror.InvalidField, "snap entry id is required")
	}
	if _, err := uuid.Parse(snapEntryId); snapEntryId != "" && err != nil {
		el.Add(cerror.InvalidField, "snap entry id is not a valid uuid")
	}
	if snapRevisionSequenceNumber == 0 {
		el.Add(cerror.InvalidField, "snap revision sequence number is required")
	}
	if el.HasError() {
		return &proto.SnapRevisionAssertionResponse{
			Errors: el.ConvertToProtoErrorList(),
		}
	}

	req := &proto.AddSnapRevisionAssertionRequest{
		SnapSha3_384:               snapSha3_384,
		DeveloperId:                developerId,
		SnapEntryId:                snapEntryId,
		SnapRevisionSequenceNumber: snapRevisionSequenceNumber,
		SnapSize:                   snapSize,
		Timestamp:                  timestamppb.New(time.Now()),
	}

	resp, err := c.client.AddSnapRevisionAssertion(context.Background(), req)
	if err != nil {
		el.Add(cerror.InternalServerError, err.Error())
		resp = &proto.SnapRevisionAssertionResponse{
			Errors: el.ConvertToProtoErrorList(),
		}
	}
	return resp
}

func (c *AssertionClient) AddSnapDeclarationAssertion(snapID, snapName, publisherID string, series string, refreshControl []string, aliases []model.Alias, plugs storeModel.Plugs, slots storeModel.Slots) *proto.SnapDeclarationAssertionResponse {
	el := cerror.NewErrorList()
	// check input
	if series == "" {
		el.Add(cerror.InvalidField, "series is required")
	}
	if snapID == "" {
		el.Add(cerror.InvalidField, "snap id is required")
	}
	if snapName == "" {
		el.Add(cerror.InvalidField, "snap name is required")
	}
	if publisherID == "" {
		el.Add(cerror.InvalidField, "publisher id is required")
	}
	if _, err := uuid.Parse(publisherID); publisherID != "" && err != nil {
		el.Add(cerror.InvalidField, "publisher id is not a valid uuid")
	}

	req := &proto.AddSnapDeclarationAssertionRequest{
		Series:         series,
		SnapId:         snapID,
		SnapName:       snapName,
		PublisherId:    publisherID,
		Timestamp:      timestamppb.New(time.Now()),
		RefreshControl: refreshControl,
	}

	req.Aliases = make([]*proto.Alias, 0, len(aliases))
	for _, a := range aliases {
		req.Aliases = append(req.Aliases, &proto.Alias{
			Name:   a.Name,
			Target: a.Target,
		})
	}

	// plugs
	req.Plugs = serializeMap(plugs)

	// slots
	req.Slots = serializeMap(slots)

	// QUESTION: think the other 3 parameters could be empty, assertion of snap package "core" does not have any of the last 3 parameters
	resp, err := c.client.AddSnapDeclarationAssertion(context.Background(), req)
	if err != nil {
		el.Add(cerror.InternalServerError, err.Error())
		resp = &proto.SnapDeclarationAssertionResponse{
			Errors: el.ConvertToProtoErrorList(),
		}
	}

	return resp
}

func (c *AssertionClient) AddAccountAssertion(accountId, displayName, username, validation string, timestamp time.Time) *proto.AccountAssertionResponse {
	el := cerror.NewErrorList()
	// check input
	if accountId == "" {
		el.Add(cerror.InvalidField, "account id is required")
	}
	if _, err := uuid.Parse(accountId); accountId != "" && err != nil {
		el.Add(cerror.InvalidField, "account id is not a valid uuid")
	}
	if displayName == "" {
		el.Add(cerror.InvalidField, "display name is required")
	}
	if username == "" {
		el.Add(cerror.InvalidField, "username is required")
	}
	if validation == "" {
		el.Add(cerror.InvalidField, "validation is required")
	}
	if timestamp.IsZero() {
		el.Add(cerror.InvalidField, "timestamp is required")
	}
	if el.HasError() {
		return &proto.AccountAssertionResponse{
			Errors: el.ConvertToProtoErrorList(),
		}
	}
	req := &proto.AddAccountAssertionRequest{
		AccountId:   accountId,
		DisplayName: displayName,
		Username:    username,
		Validation:  validation,
		Timestamp:   timestamppb.New(timestamp),
	}
	resp, err := c.client.AddAccountAssertion(context.Background(), req)
	if err != nil {
		el.Add(cerror.InternalServerError, err.Error())
		resp = &proto.AccountAssertionResponse{
			Errors: el.ConvertToProtoErrorList(),
		}
	}
	return resp
}

func (c *AssertionClient) AddSnapBuildAssertion(sha3_384Encoded, grade, signKeySha3_384Encoded string, developerId, snapEntryId uuid.UUID, size uint64) *proto.SnapBuildAssertionResponse {
	el := cerror.NewErrorList()

	// check input
	if sha3_384Encoded == "" {
		el.Add(cerror.InvalidField, "sha3_384_encoded is required")
	}
	if grade == "" {
		el.Add(cerror.InvalidField, "grade is required")
	}
	if signKeySha3_384Encoded == "" {
		el.Add(cerror.InvalidField, "sign key sha3_384 encoded is required")
	}
	if developerId == uuid.Nil {
		el.Add(cerror.InvalidField, "developer id is required")
	}
	if snapEntryId == uuid.Nil {
		el.Add(cerror.InvalidField, "snap entry id is required")
	}
	if el.HasError() {
		return &proto.SnapBuildAssertionResponse{
			Errors: el.ConvertToProtoErrorList(),
		}
	}

	req := &proto.AddSnapBuildAssertionRequest{
		Sha3_384Encoded:        sha3_384Encoded,
		Grade:                 grade,
		SignKeySha3_384Encoded: signKeySha3_384Encoded,
		DeveloperId:           developerId.String(),
		SnapEntryId:           snapEntryId.String(),
		SnapSize:              size,
	}

	resp, err := c.client.AddSnapBuildAssertion(context.Background(), req)
	if err != nil {
		el.Add(cerror.InternalServerError, err.Error())
		resp = &proto.SnapBuildAssertionResponse{
			Errors: el.ConvertToProtoErrorList(),
		}
	}
	return resp
}

func (c *AssertionClient) GetAccountKeyAssertionByPublicKeySha(publicKeySha3_384Encoded string) *proto.AccountKeyAssertionResponse {
	el := cerror.NewErrorList()

	// check input
	if publicKeySha3_384Encoded == "" {
		el.Add(cerror.InvalidField, "publicKeySha3_384Encoded is required")
	}
	if el.HasError() {
		return &proto.AccountKeyAssertionResponse{
			Errors: el.ConvertToProtoErrorList(),
		}
	}

	req := &proto.GetAccountKeyAssertionByPublicKeyShaRequest{
		PublicKeySha3_384Encoded: publicKeySha3_384Encoded,
	}

	resp, err := c.client.GetAccountKeyAssertionByPublicKeySha(context.Background(), req)
	if err != nil {
		el.Add(cerror.InternalServerError, err.Error())
		resp = &proto.AccountKeyAssertionResponse{
			Errors: el.ConvertToProtoErrorList(),
		}
	}
	return resp
}

func (c *AssertionClient) GetSnapRevisionAssertionBySHA3_384(snapSha3_384 string) *proto.SnapRevisionAssertionResponse {
	el := cerror.NewErrorList()

	// check input
	if snapSha3_384 == "" {
		el.Add(cerror.InvalidField, "snap sha3_384 is required")
	}
	if el.HasError() {
		return &proto.SnapRevisionAssertionResponse{
			Errors: el.ConvertToProtoErrorList(),
		}
	}

	req := &proto.GetSnapRevisionAssertionBySHA3_384Request{
		SnapSha3_384: snapSha3_384,
	}

	resp, err := c.client.GetSnapRevisionAssertionBySHA3_384(context.Background(), req)
	if err != nil {
		el.Add(cerror.InternalServerError, err.Error())
		resp = &proto.SnapRevisionAssertionResponse{
			Errors: el.ConvertToProtoErrorList(),
		}
	}
	return resp
}

func (c *AssertionClient) GetSnapDeclarationAssertionBySnapID(snapId string) *proto.SnapDeclarationAssertionResponse {
	el := cerror.NewErrorList()

	// check input
	if snapId == "" {
		el.Add(cerror.InvalidField, "snap id is required")
	}
	if el.HasError() {
		return &proto.SnapDeclarationAssertionResponse{
			Errors: el.ConvertToProtoErrorList(),
		}
	}

	req := &proto.GetSnapDeclarationAssertionBySnapIDRequest{
		SnapId: snapId,
	}

	resp, err := c.client.GetSnapDeclarationAssertionBySnapID(context.Background(), req)
	if err != nil {
		el.Add(cerror.InternalServerError, err.Error())
		resp = &proto.SnapDeclarationAssertionResponse{
			Errors: el.ConvertToProtoErrorList(),
		}
	}
	return resp
}

func (c *AssertionClient) GetAccountAssertionByAccountID(accountId string) *proto.AccountAssertionResponse {
	el := cerror.NewErrorList()

	// check input
	if accountId == "" {
		el.Add(cerror.InvalidField, "account id is required")
	}
	if _, err := uuid.Parse(accountId); accountId != "" && err != nil {
		el.Add(cerror.InvalidField, "account id is not a valid uuid")
	}
	if el.HasError() {
		return &proto.AccountAssertionResponse{
			Errors: el.ConvertToProtoErrorList(),
		}
	}

	req := &proto.GetAccountAssertionByAccountIDRequest{
		AccountId: accountId,
	}

	resp, err := c.client.GetAccountAssertionByAccountID(context.Background(), req)
	if err != nil {
		el.Add(cerror.InternalServerError, err.Error())
		resp = &proto.AccountAssertionResponse{
			Errors: el.ConvertToProtoErrorList(),
		}
	}
	return resp
}

// ========== HELPER FUNCTIONS ==========

func serializeMap(a any) string {
	data, err := json.Marshal(a)
	if err != nil {
		logrus.Errorf("failed to serialize: %s", err.Error())
		return ""
	}
	return string(data)
}
