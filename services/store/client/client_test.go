package client

import (
	"context"

	proto "github.com/idlab-discover/kebeng/services/store/proto"
	"google.golang.org/grpc"
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

type mockStoreServiceServer struct {
	proto.UnimplementedStoreServiceServer
}

func (m *mockStoreServiceServer) UploadSnap(ctx context.Context, req *proto.UploadSnapRequest) (*proto.UploadSnapResponse, error) {
	return &proto.UploadSnapResponse{Id: "Snap uploaded successfully"}, nil
}
