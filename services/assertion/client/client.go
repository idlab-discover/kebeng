package client

import (
	"fmt"

	"github.com/idlab-discover/kebeng/services/assertion/internal/config"
	proto "github.com/idlab-discover/kebeng/services/assertion/proto"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

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
	c.conn.Close()
}
