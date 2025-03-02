package client

import (
	"context"
	"net"
	"testing"

	proto "github.com/idlab-discover/kebeng/services/store/proto"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

const bufSize = 1024 * 1024

var lis *bufconn.Listener

func init() {
	lis = bufconn.Listen(bufSize)
	s := grpc.NewServer()
	proto.RegisterStoreServiceServer(s, &mockStoreServiceServer{})
	go func() {
		if err := s.Serve(lis); err != nil {
			panic(err)
		}
	}()
}

func bufDialer(ctx context.Context, s string) (net.Conn, error) {
	return lis.Dial()
}

type mockStoreServiceServer struct {
	proto.UnimplementedStoreServiceServer
}

func (m *mockStoreServiceServer) UploadSnap(ctx context.Context, req *proto.UploadSnapRequest) (*proto.UploadSnapResponse, error) {
	return &proto.UploadSnapResponse{Id: "Snap uploaded successfully"}, nil
}

func TestUploadSnap(t *testing.T) {
	conn, err := grpc.DialContext(context.Background(), "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}
	defer conn.Close()

	client := proto.NewStoreServiceClient(conn)
	storeClient := &StoreClient{conn, client}

	t.Run("UploadSnap successfully", func(t *testing.T) {
		resp, err := storeClient.UploadSnap("test-snap", "app", "strict", "core18", []byte("snap content"))
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, "Snap uploaded successfully", resp.Id)
	})

	t.Run("UploadSnap with error", func(t *testing.T) {
		// Simulate an error by passing an invalid request
		resp, err := storeClient.UploadSnap("", "", "", "", nil)
		assert.Error(t, err)
		assert.Nil(t, resp)
	})
}
