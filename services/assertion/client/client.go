package client

import (
	"context"
	"fmt"

	cerror "github.com/idlab-discover/kebeng/common/cerror"
	"github.com/idlab-discover/kebeng/services/assertion/internal/config"
	proto "github.com/idlab-discover/kebeng/services/assertion/proto"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type AssertionClientInterface interface {
	ProcessSnapBuildAssertion(assertion []byte) *proto.SnapBuildAssertionResponse
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
		logrus.Errorf("Error closing connection: %v", err)
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
			Errors: []*proto.Error{{
				Code:    cerror.InternalServerError,
				Message: err.Error()},
			},
		}
	}
	return resp
}
