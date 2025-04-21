package client

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	cerror "github.com/idlab-discover/kebeng/common/cerror"
	cerrorpb "github.com/idlab-discover/kebeng/common/cerror/proto"
	"github.com/idlab-discover/kebeng/services/assertion/client/model"
	"github.com/idlab-discover/kebeng/services/assertion/internal/config"
	proto "github.com/idlab-discover/kebeng/services/assertion/proto"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type AssertionClientInterface interface {
	ProcessSnapBuildAssertion(assertion []byte) *proto.SnapBuildAssertionResponse

	AddAccountKeyAssertion(encoded_public_key, publicKeySha3_384, accountId, name string, since time.Time, until time.Time) *proto.AccountKeyAssertionResponse
	AddSnapRevisionAssertion(snapSha3_384 string, developerId string, snapEntryId string, snapRevisionSequenceNumber uint32, snapSize uint64, timestamp time.Time) *proto.SnapRevisionAssertionResponse
	AddSnapDeclarationAssertion(snapID, snapName, publisherID string, series uint32, timestamp time.Time, refreshControl []string, aliases []model.Alias, plugs model.PlugMap, slots model.SlotMap) *proto.SnapDeclarationAssertionResponse
	AddAccountAssertion(accountId, displayName, username, validation string, timestamp time.Time) *proto.AccountAssertionResponse

	GetAccountKeyAssertionByPublicKeySha(publicKeySha3_384 string) *proto.AccountKeyAssertionResponse
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

func NewAssertionClient(assertionHost string, assertionPort int) (*AssertionClient, error) {
	logrus.Infof("Connecting to account service at %s:%d", assertionHost, assertionPort)
	conn, err := grpc.NewClient(config.GetAssertionServiceAddress(assertionHost, assertionPort), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("could not connect: %v", err)
	}

	client := proto.NewAssertionServiceClient(conn)
	return &AssertionClient{conn, client}, nil
}

func (c *AssertionClient) Close() {
	err := c.conn.Close()
	if err != nil {
		logrus.Errorf("error closing connection: %v", err)
	}
}

func (c *AssertionClient) ProcessSnapBuildAssertion(assertion []byte) *proto.SnapBuildAssertionResponse {
	req := &proto.SnapBuildAssertionRequest{
		Assertion: assertion,
	}

	resp, err := c.client.ProcessSnapBuildAssertion(context.Background(), req)
	// err is not nil if something goes wrong with the client
	// cerror regarding the request are in the response
	if err != nil {
		resp = &proto.SnapBuildAssertionResponse{
			Errors: []*cerrorpb.Error{{
				Code:    cerror.InternalServerError,
				Message: err.Error()},
			},
		}
	}
	return resp
}

func (c *AssertionClient) AddAccountKeyAssertion(encoded_public_key, publicKeySha3_384, accountId, name string, since time.Time, until time.Time) *proto.AccountKeyAssertionResponse {
	el := cerror.NewErrorList()

	// check input
	if encoded_public_key == "" {
		el.Add(cerror.InvalidField, "encoded public key is required")
	}
	if publicKeySha3_384 == "" {
		el.Add(cerror.InvalidField, "public key sha3_384 is required")
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
		EncodedPublicKey:  encoded_public_key,
		PublicKeySha3_384: publicKeySha3_384,
		AccountId:         accountId,
		Name:              name,
		Since:             timestamppb.New(since),
		Until:             timestamppb.New(until),
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

func (c *AssertionClient) AddSnapRevisionAssertion(snapSha3_384, developerId, snapEntryId string, snapRevisionSequenceNumber uint32, snapSize uint64, timestamp time.Time) *proto.SnapRevisionAssertionResponse {
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
		Timestamp:                  timestamppb.New(timestamp),
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

func (c *AssertionClient) AddSnapDeclarationAssertion(snapID, snapName, publisherID string, series uint32, timestamp time.Time, refreshControl []string, aliases []model.Alias, plugs model.PlugMap, slots model.SlotMap) *proto.SnapDeclarationAssertionResponse {
	el := cerror.NewErrorList()
	// check input
	if series == 0 {
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
	if timestamp.IsZero() {
		el.Add(cerror.InvalidField, "timestamp is required")
	}

	req := &proto.AddSnapDeclarationAssertionRequest{
		Series:         series,
		SnapId:         snapID,
		SnapName:       snapName,
		PublisherId:    publisherID,
		Timestamp:      timestamppb.New(timestamp),
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
	req.Plugs = make(map[string]*proto.PlugRule, len(plugs))
	for iface, r := range plugs {
		req.Plugs[iface] = &proto.PlugRule{
			AllowInstallation:   r.AllowInstallation,
			DenyInstallation:    r.DenyInstallation,
			AllowConnection:     r.AllowConnection,
			DenyConnection:      r.DenyConnection,
			AllowAutoConnection: r.AllowAutoConnection,
			DenyAutoConnection:  r.DenyAutoConnection,
		}
	}

	// slots
	req.Slots = make(map[string]*proto.SlotRule, len(slots))
	for iface, r := range slots {
		req.Slots[iface] = &proto.SlotRule{
			AllowInstallation:   r.AllowInstallation,
			DenyInstallation:    r.DenyInstallation,
			AllowConnection:     r.AllowConnection,
			DenyConnection:      r.DenyConnection,
			AllowAutoConnection: r.AllowAutoConnection,
			DenyAutoConnection:  r.DenyAutoConnection,
		}
	}

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

func (c *AssertionClient) GetAccountKeyAssertionByPublicKeySha(publicKeySha3_384 string) *proto.AccountKeyAssertionResponse {
	el := cerror.NewErrorList()

	// check input
	if publicKeySha3_384 == "" {
		el.Add(cerror.InvalidField, "name is required")
	}
	if el.HasError() {
		return &proto.AccountKeyAssertionResponse{
			Errors: el.ConvertToProtoErrorList(),
		}
	}

	req := &proto.GetAccountKeyAssertionByPublicKeyShaRequest{
		PublicKeySha3_384: publicKeySha3_384,
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
