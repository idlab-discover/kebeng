package client

import (
	"context"
	"fmt"

	"github.com/idlab-discover/kebeng/services/account/internal/config"
	proto "github.com/idlab-discover/kebeng/services/account/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type AccountClient struct {
    conn *grpc.ClientConn
    client proto.AccountServiceClient
}

func NewAccountClient() (*AccountClient, error) {
    // kinda ugly the way config is now but don't want to pass a parameter in this function (easier use)
    conn, err := grpc.NewClient(config.GetAccountServiceAddress(), grpc.WithTransportCredentials(insecure.NewCredentials()))
    if err != nil {
        return nil, fmt.Errorf("could not connect: %v", err)
    }

    client := proto.NewAccountServiceClient(conn)
    return &AccountClient{conn, client}, nil
}

func (c *AccountClient) Close() {
    c.conn.Close()
}

func (c *AccountClient) CreateAccount(displayName, username, email string) (*proto.CreateAccountResponse, error) {
    req := &proto.CreateAccountRequest{
        DisplayName: displayName,
        Username: username,
        Email: email,
    }

    resp, err := c.client.CreateAccount(context.Background(), req)
    if err != nil {
        return nil, fmt.Errorf("could not create account: %v", err)
    }
    return resp, nil
}

