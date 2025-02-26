package client

import (
	"context"
	"fmt"

	"github.com/idlab-discover/kebeng/services/account/internal/config"
	proto "github.com/idlab-discover/kebeng/services/account/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
    "github.com/sirupsen/logrus"
)

type AccountClient struct {
    conn *grpc.ClientConn
    client proto.AccountServiceClient
}

func NewAccountClient(accountHost string, accountPort int) (*AccountClient, error) {
    // kinda ugly the way config is now but don't want to pass a parameter in this function (easier use)
    logrus.Infof("Connecting to account service at %s:%d", accountHost, accountPort)
    conn, err := grpc.NewClient(config.GetAccountServiceAddress(accountHost,accountPort), grpc.WithTransportCredentials(insecure.NewCredentials()))
    if err != nil {
        return nil, fmt.Errorf("could not connect: %v", err)
    }

    client := proto.NewAccountServiceClient(conn)
    return &AccountClient{conn, client}, nil
}

func (c *AccountClient) Close() {
    c.conn.Close()
}

func (c *AccountClient) CreateAccount(displayName, username, email string) (*proto.Account, error) {
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

func (c *AccountClient) UpdateAccount(id, displayName, username, email string) (*proto.Account, error) {
    req := &proto.UpdateAccountRequest{
        Id:          id,
        DisplayName: displayName,
        Username:    username,
        Email:       email,
    }

    resp, err := c.client.UpdateAccount(context.Background(), req)
    if err != nil {
        return nil, fmt.Errorf("could not update account: %v", err)
    }
    return resp, nil
}

func (c *AccountClient) DeleteAccount(id string) (bool, error) {
    req := &proto.DeleteAccountRequest{Id: id}

    resp, err := c.client.DeleteAccount(context.Background(), req)
    if err != nil {
        return false, fmt.Errorf("could not delete account: %v", err)
    }
    return resp.Success, nil
}

func (c *AccountClient) GetAccountByEmail(email string) (*proto.Account, error) {
    req := &proto.GetAccountByEmailRequest{Email: email}

    resp, err := c.client.GetAccountByEmail(context.Background(), req)
    if err != nil {
        return nil, fmt.Errorf("could not get account by email: %v", err)
    }
    return resp, nil
}

func (c *AccountClient) GetAccountByID(id string) (*proto.Account, error) {
    req := &proto.GetAccountByIDRequest{Id: id}

    resp, err := c.client.GetAccountByID(context.Background(), req)
    if err != nil {
        return nil, fmt.Errorf("could not get account by ID: %v", err)
    }
    return resp, nil
}

func (c *AccountClient) GetAccountByUsername(username string) (*proto.Account, error) {
    req := &proto.GetAccountByUsernameRequest{Username: username}

    resp, err := c.client.GetAccountByUsername(context.Background(), req)
    if err != nil {
        return nil, fmt.Errorf("could not get account by username: %v", err)
    }
    return resp, nil
}

