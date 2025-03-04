package client

import (
	"context"
	"fmt"

	"github.com/idlab-discover/kebeng/services/store/internal/config"
	"github.com/idlab-discover/kebeng/services/store/internal/errors"
	proto "github.com/idlab-discover/kebeng/services/store/proto"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type StoreClient struct {
	conn   *grpc.ClientConn
	client proto.StoreServiceClient
}

func NewStoreClient(storeHost string, storePort int) (*StoreClient, error) {
	logrus.Infof("Connecting to account service at %s:%d", storeHost, storePort)
	conn, err := grpc.NewClient(config.GetStoreServiceAddress(storeHost, storePort), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("could not connect: %v", err)
	}

	client := proto.NewStoreServiceClient(conn)
	return &StoreClient{conn, client}, nil
}

func (c *StoreClient) Close() {
	c.conn.Close()
}

func (c *StoreClient) UploadSnap(name string, type_name string, confinement string, base string, file []byte) *proto.UploadSnapResponse {
	req := &proto.UploadSnapRequest{
		Name:        name,
		Type:        type_name,
		Confinement: confinement,
		Base:        base,
		File:        file,
	}

	resp, err := c.client.UploadSnap(context.Background(), req)
	if err != nil {
		resp = &proto.UploadSnapResponse{
			Errors: []*proto.Error{{
				Code:    errors.InternalServerError,
				Message: err.Error()},
			},
		}
	}
	return resp
}

func (c *StoreClient) RegisterSnapName(snapName string, isPrivate bool, storeName string, dryRun bool) *proto.RegisterSnapNameResponse {
	req := &proto.RegisterSnapNameRequest{
		SnapName:  snapName,
		IsPrivate: isPrivate,
		Store:     storeName,
		DryRun:    dryRun,
	}

	resp, err := c.client.RegisterSnapName(context.Background(), req)
	if err != nil {
		resp = &proto.RegisterSnapNameResponse{
			Errors: []*proto.Error{{
				Code:    errors.InternalServerError,
				Message: err.Error()},
			},
		}
	}
	return resp
}
