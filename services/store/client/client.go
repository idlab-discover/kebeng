package client

import (
	"context"
	"fmt"

	"github.com/idlab-discover/kebeng/services/store/internal/config"
	proto "github.com/idlab-discover/kebeng/services/store/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type StoreClient struct {
	conn   *grpc.ClientConn
	client proto.StoreServiceClient
}

func NewStoreClient() (*StoreClient, error) {
	conn, err := grpc.NewClient(config.GetStoreServiceAddress(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("could not connect: %v", err)
	}

	client := proto.NewStoreServiceClient(conn)
	return &StoreClient{conn, client}, nil
}

func (c *StoreClient) Close() {
	c.conn.Close()
}

func (c *StoreClient) UploadSnap(name string, type_name string, confinement string, base string, file []byte) (*proto.UploadSnapResponse, error) {
	if name == "" || type_name == "" || confinement == "" || base == "" {
		return nil, fmt.Errorf("all string parameters must be non-empty")
	}

	req := &proto.UploadSnapRequest{
		Name:        name,
		Type:        type_name,
		Confinement: confinement,
		Base:        base,
		File:        file,
	}

	resp, err := c.client.UploadSnap(context.Background(), req)
	if err != nil {
		return nil, fmt.Errorf("could not upload snap: %v", err)
	}
	return resp, nil
}
