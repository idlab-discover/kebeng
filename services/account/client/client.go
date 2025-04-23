package client

import (
	"context"
	"fmt"
	"time"

	cerror "github.com/idlab-discover/kebeng/common/cerror"
	cerrorpb "github.com/idlab-discover/kebeng/common/cerror/proto"
	"github.com/idlab-discover/kebeng/services/account/internal/config"
	proto "github.com/idlab-discover/kebeng/services/account/proto"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health/grpc_health_v1"
)

const (
	maxDialAttempts = 5
	dialRetryDelay  = 2 * time.Second
	healthTimeout   = 2 * time.Second
)

// NOTE: in all the endpoints the cerror of the actual logic are included in the response and NOT returned as an error
// this is because we need every error and not just 1 according to the snapcraft docs
// there is an err field in the proto functions because proto incists on it so if that is != nil the request failed
// but the actual error is in the cerror field

type AccountClientInterface interface {
	Close()
	AddAccount(displayName, username, email, hashedPassword string) *proto.AccountResponse
	UpdateAccount(id, displayName, username, email string) *proto.AccountResponse
	DeleteAccount(id string) *proto.DeleteAccountResponse
	GetAccountByEmail(email string) *proto.AccountResponse
	GetAccountByID(id string) *proto.AccountResponse
	GetAccountsByIds(ids []string) *proto.GetAccountsByIdsResponse
	GetAccountByUsername(username string) *proto.AccountResponse
	AddKey(accountEmail, keyName, sha3384, encodedPublicKey string) *proto.KeyResponse
	GetAccountKey(Sha3384 string) *proto.KeyResponse
	GetAccountKeysByAccountID(accountID string) *proto.KeysResponse
	PatchAccountByEmail(email, username string) *proto.PatchAccountByEmailResponse
}

var _ AccountClientInterface = (*AccountClient)(nil)

type AccountClient struct {
	conn   *grpc.ClientConn
	client proto.AccountServiceClient
}

func NewAccountClientWithClient(client proto.AccountServiceClient) *AccountClient {
	return &AccountClient{client: client}
}

// hard to test with unit test
func NewAccountClient(accountHost string, accountPort int) (*AccountClient, *cerror.CustomError) {
	logrus.Infof("Connecting to account service at %s:%d", accountHost, accountPort)
	addr := config.GetAccountServiceAddress(accountHost, accountPort)

	for attempt := 1; attempt <= maxDialAttempts; attempt++ {
		logrus.Infof("Attempt %d/%d: dialing account service at %s", attempt, maxDialAttempts, addr)
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
			logrus.Infof("Successfully connected to account service at %s", addr)
			return &AccountClient{conn, proto.NewAccountServiceClient(conn)}, nil
		}
		time.Sleep(dialRetryDelay)
	}
	return nil, cerror.NewCustomError(cerror.InternalServerError, fmt.Sprintf("failed to connect to account service after %d attempts", maxDialAttempts))
}

func (c *AccountClient) Close() {
	err := c.conn.Close()
	if err != nil {
		logrus.Errorf("error closing account client connection: %v", err)
	}
}

func (c *AccountClient) AddAccount(displayName, username, email, hashedPassword string) *proto.AccountResponse {
	req := &proto.AddAccountRequest{
		DisplayName:    displayName,
		Username:       username,
		Email:          email,
		HashedPassword: hashedPassword,
	}

	resp, err := c.client.AddAccount(context.Background(), req)
	if err != nil {
		// this means proto request failed not the actual logic
		resp = &proto.AccountResponse{
			Errors: []*cerrorpb.Error{{
				Code: cerror.InternalServerError, Message: err.Error()},
			},
		}
	}
	return resp
}

func (c *AccountClient) UpdateAccount(id, displayName, username, email string) *proto.AccountResponse {
	req := &proto.UpdateAccountRequest{
		Id:          id,
		DisplayName: displayName,
		Username:    username,
		Email:       email,
	}

	resp, err := c.client.UpdateAccount(context.Background(), req)
	if err != nil {
		// this means proto request failed not the actual logic
		resp = &proto.AccountResponse{
			Errors: []*cerrorpb.Error{{
				Code: cerror.InternalServerError, Message: err.Error()},
			},
		}
	}
	return resp
}

func (c *AccountClient) DeleteAccount(id string) *proto.DeleteAccountResponse {
	req := &proto.DeleteAccountRequest{Id: id}

	resp, err := c.client.DeleteAccount(context.Background(), req)
	if err != nil {
		// this means proto request failed not the actual logic
		resp = &proto.DeleteAccountResponse{
			Errors: []*cerrorpb.Error{{
				Code: cerror.InternalServerError, Message: err.Error()},
			},
		}
	}
	return resp
}

func (c *AccountClient) GetAccountByEmail(email string) *proto.AccountResponse {
	req := &proto.GetAccountByEmailRequest{Email: email}

	resp, err := c.client.GetAccountByEmail(context.Background(), req)
	if err != nil {
		// this means proto request failed not the actual logic
		resp = &proto.AccountResponse{
			Errors: []*cerrorpb.Error{{
				Code: cerror.InternalServerError, Message: err.Error()},
			},
		}
	}
	return resp
}

func (c *AccountClient) GetAccountByID(id string) *proto.AccountResponse {
	req := &proto.GetAccountByIDRequest{Id: id}

	resp, err := c.client.GetAccountByID(context.Background(), req)
	if err != nil {
		// this means proto request failed not the actual logic
		resp = &proto.AccountResponse{
			Errors: []*cerrorpb.Error{{
				Code: cerror.InternalServerError, Message: err.Error()},
			},
		}
	}
	return resp
}

func (c *AccountClient) GetAccountsByIds(ids []string) *proto.GetAccountsByIdsResponse {
	req := &proto.GetAccountsByIdsRequest{Ids: ids}
	resp, err := c.client.GetAccountsByIds(context.Background(), req)
	if err != nil {
		// this means proto request failed not the actual logic
		resp = &proto.GetAccountsByIdsResponse{
			Errors: []*cerrorpb.Error{{
				Code: cerror.InternalServerError, Message: err.Error()},
			},
		}
	}
	return resp
}

func (c *AccountClient) GetAccountByUsername(username string) *proto.AccountResponse {
	req := &proto.GetAccountByUsernameRequest{Username: username}

	resp, err := c.client.GetAccountByUsername(context.Background(), req)
	if err != nil {
		// this means proto request failed not the actual logic
		resp = &proto.AccountResponse{
			Errors: []*cerrorpb.Error{{
				Code: cerror.InternalServerError, Message: err.Error()},
			},
		}
	}
	return resp
}

func (c *AccountClient) AddKey(accountEmail, keyName, sha3384, encodedPublicKey string) *proto.KeyResponse {
	req := &proto.AddKeyRequest{
		AccountEmail:     accountEmail,
		KeyName:          keyName,
		Sha3384:          sha3384,
		EncodedPublicKey: encodedPublicKey,
	}

	resp, err := c.client.AddKey(context.Background(), req)
	if err != nil {
		// this means proto request failed not the actual logic
		resp = &proto.KeyResponse{
			Errors: []*cerrorpb.Error{{
				Code: cerror.InternalServerError, Message: err.Error()},
			},
		}
	}
	return resp
}

func (c *AccountClient) GetAccountKey(Sha3384 string) *proto.KeyResponse {
	req := &proto.GetKeyBySHA3384Request{Sha3384: Sha3384}

	resp, err := c.client.GetKeyBySHA3384(context.Background(), req)
	if err != nil {
		// this means proto request failed not the actual logic
		resp = &proto.KeyResponse{
			Errors: []*cerrorpb.Error{{
				Code: cerror.InternalServerError, Message: err.Error()},
			},
		}
	}
	return resp
}

func (c *AccountClient) GetAccountKeysByAccountID(accountID string) *proto.KeysResponse {
	req := &proto.GetKeysByAccountIdRequest{AccountId: accountID}

	resp, err := c.client.GetKeysByAccountId(context.Background(), req)
	if err != nil {
		// this means proto request failed not the actual logic
		resp = &proto.KeysResponse{
			Errors: []*cerrorpb.Error{{
				Code: cerror.InternalServerError, Message: err.Error()},
			},
		}
	}
	return resp
}

func (c *AccountClient) PatchAccountByEmail(email, username string) *proto.PatchAccountByEmailResponse {
	req := &proto.PatchAccountByEmailRequest{Email: email, Username: username}

	resp, err := c.client.PatchAccountByEmail(context.Background(), req)
	if err != nil {
		// this means proto request failed not the actual logic
		resp = &proto.PatchAccountByEmailResponse{
			Errors: []*cerrorpb.Error{{
				Code: cerror.InternalServerError, Message: err.Error()},
			},
		}
	}
	return resp
}
