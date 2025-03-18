package client

import (
	"context"
	"fmt"

	cerror "github.com/idlab-discover/kebeng/common/error"
	"github.com/idlab-discover/kebeng/services/account/internal/config"
	proto "github.com/idlab-discover/kebeng/services/account/proto"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// NOTE: in all the endpoints the cerror of the actual logic are included in the response and NOT returned as an error
// this is because we need every error and not just 1 according to the snapcraft docs
// there is an err field in the proto functions because proto incists on it so if that is != nil the request failed
// but the actual error is in the cerror field

type AccountClient struct {
	conn   *grpc.ClientConn
	client proto.AccountServiceClient
}

func NewAccountClient(accountHost string, accountPort int) (*AccountClient, error) {
	logrus.Infof("Connecting to account service at %s:%d", accountHost, accountPort)
	conn, err := grpc.NewClient(config.GetAccountServiceAddress(accountHost, accountPort), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("could not connect: %v", err)
	}

	client := proto.NewAccountServiceClient(conn)
	return &AccountClient{conn, client}, nil
}

func (c *AccountClient) Close() {
	c.conn.Close()
}

func (c *AccountClient) CreateAccount(displayName, username, email string) *proto.AccountResponse {
	req := &proto.CreateAccountRequest{
		DisplayName: displayName,
		Username:    username,
		Email:       email,
	}

	resp, err := c.client.CreateAccount(context.Background(), req)
	if err != nil {
		// this means proto request failed not the actual logic
		resp = &proto.AccountResponse{
			Errors: []*proto.Error{{
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
			Errors: []*proto.Error{{
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
			Errors: []*proto.Error{{
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
			Errors: []*proto.Error{{
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
			Errors: []*proto.Error{{
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
			Errors: []*proto.Error{{
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
			Errors: []*proto.Error{{
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
			Errors: []*proto.Error{{
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
			Errors: []*proto.Error{{
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
			Errors: []*proto.Error{{
				Code: cerror.InternalServerError, Message: err.Error()},
			},
		}
	}
	return resp
}
