package client

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/google/uuid"
	cerror "github.com/idlab-discover/kebeng/common/cerror"
	cerrorpb "github.com/idlab-discover/kebeng/common/cerror/proto"
	"github.com/idlab-discover/kebeng/services/store/internal/config"
	proto "github.com/idlab-discover/kebeng/services/store/proto"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type StoreClientInterface interface {
	Close()
	RegisterSnapName(snapName string, isPrivate bool, storeName string, dryRun bool, accountId uuid.UUID) *proto.RegisterSnapNameResponse
	GetEntries(entries *proto.GetEntriesRequest) *proto.GetEntriesResponse
	GetRevisions(revisions *proto.GetRevisionsRequest) *proto.GetRevisionsResponse
	GetEntriesByAccountID(accountID string) *proto.GetEntriesResponse
	GetRevisionsByEntryIds(entryIds *proto.GetRevisionsByEntryIdRequests) *proto.GetRevisionsByEntryIdResponses
	GetLatestRevision(snapName, track, channel string) *proto.GetRevisionResponse
	SnapDownload(revisionId string) *proto.SnapDownloadCompleteResponse
	UnscannedUpload(ctx context.Context, snapFile io.Reader) *proto.UnscannedUploadCompleteResponse
	AddUpload(snapName string, entryId uuid.UUID, status string, accountId uuid.UUID, unscannedFileName string) *proto.AddUploadResponse
	GetUploadStatus(uploadId string) *proto.GetUploadStatusResponse
	AddRevision(snapName string, sha3384 string, size uint64, architectures []string, tracksAndChannels []string, unscannedFileName string, minioFilePath string) *proto.AddRevisionResponse
	GetObjectCustomMetadata(bucket string, objectKey string) *proto.GetObjectCustomMetadataResponse
}

var _ StoreClientInterface = (*StoreClient)(nil)

type StoreClient struct {
	conn   *grpc.ClientConn
	client proto.StoreServiceClient
}

func NewStoreClientWithClient(client proto.StoreServiceClient) *StoreClient {
	return &StoreClient{client: client}
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
	err := c.conn.Close()
	if err != nil {
		logrus.Errorf("error closing store client connection: %v", err)
	}
}

func (c *StoreClient) RegisterSnapName(snapName string, isPrivate bool, storeName string, dryRun bool, accountId uuid.UUID) *proto.RegisterSnapNameResponse {
	req := &proto.RegisterSnapNameRequest{
		SnapName:  snapName,
		IsPrivate: isPrivate,
		Store:     storeName,
		DryRun:    dryRun,
		AccountId: accountId.String(),
	}

	resp, err := c.client.RegisterSnapName(context.Background(), req)
	if err != nil {
		resp = &proto.RegisterSnapNameResponse{
			Errors: []*cerrorpb.Error{{
				Code:    cerror.InternalServerError,
				Message: err.Error()},
			},
		}
	}
	return resp
}

func (c *StoreClient) GetEntries(entries *proto.GetEntriesRequest) *proto.GetEntriesResponse {
	resp, err := c.client.GetEntries(context.Background(), entries)
	if err != nil {
		resp = &proto.GetEntriesResponse{
			Errors: []*cerrorpb.Error{{
				Code:    cerror.InternalServerError,
				Message: err.Error()},
			},
		}
	}
	return resp
}

func (c *StoreClient) GetRevisions(revisions *proto.GetRevisionsRequest) *proto.GetRevisionsResponse {
	resp, err := c.client.GetRevisions(context.Background(), revisions)
	if err != nil {
		resp = &proto.GetRevisionsResponse{
			Errors: []*cerrorpb.Error{{
				Code:    cerror.InternalServerError,
				Message: err.Error()},
			},
		}
	}
	return resp
}

func (c *StoreClient) GetEntriesByAccountID(accountID string) *proto.GetEntriesResponse {
	req := &proto.GetEntriesByAccountIdRequest{AccountId: accountID}
	resp, err := c.client.GetEntriesByAccountId(context.Background(), req)
	if err != nil {
		resp = &proto.GetEntriesResponse{
			Errors: []*cerrorpb.Error{{
				Code:    cerror.InternalServerError,
				Message: err.Error()},
			},
		}
	}
	return resp
}

func (c *StoreClient) GetRevisionsByEntryIds(entryIds *proto.GetRevisionsByEntryIdRequests) *proto.GetRevisionsByEntryIdResponses {
	resp, err := c.client.GetRevisionsByEntryIds(context.Background(), entryIds)
	if err != nil {
		resp = &proto.GetRevisionsByEntryIdResponses{
			Errors: []*cerrorpb.Error{{
				Code:    cerror.InternalServerError,
				Message: err.Error()},
			},
		}
	}
	return resp
}

func (c *StoreClient) GetLatestRevision(snapName, track, channel string) *proto.GetRevisionResponse {
	// if snapName is empty we cant do anything
	if snapName == "" {
		return &proto.GetRevisionResponse{
			Errors: []*cerrorpb.Error{{
				Code:    cerror.MissingField,
				Message: "snapName is required"},
			},
		}
	}
	// Default track to "stable" if not provided
	if track == "" {
		track = "latest"
	}
	// Default channel to "stable" if not provided
	if channel == "" {
		channel = "stable"
	}
	req := &proto.GetLatestRevisionRequest{
		SnapName: snapName,
		Track:    track,
		Channel:  channel,
	}

	resp, err := c.client.GetLatestRevisionByTrackAndChannel(context.Background(), req)
	if err != nil {
		resp = &proto.GetRevisionResponse{
			Errors: []*cerrorpb.Error{{
				Code:    cerror.InternalServerError,
				Message: err.Error()},
			},
		}
	}
	return resp
}

// TODO: fix so that file gets streamed
func (c *StoreClient) UnscannedUpload(ctx context.Context, snapFile io.Reader) *proto.UnscannedUploadCompleteResponse {
	// create a stream to send the file to the server
	stream, err := c.client.UnscannedUpload(context.Background())
	if err != nil {
		return &proto.UnscannedUploadCompleteResponse{
			Errors: []*cerrorpb.Error{{
				Code:    cerror.InternalServerError,
				Message: err.Error(),
			}},
		}
	}

	var size uint64 = 0
	// read the file in chunks of 1MB and send it to the server
	buffer := make([]byte, 1024*1024) // 1 MB buffer
	for {
		n, err := snapFile.Read(buffer)
		if err != nil && err != io.EOF {
			return &proto.UnscannedUploadCompleteResponse{
				Errors: []*cerrorpb.Error{{
					Code:    cerror.InternalServerError,
					Message: err.Error(),
				}},
			}
		}
		size += uint64(n)
		if n == 0 {
			break
		}

		req := &proto.UnscannedUploadRequest{
			Payload: &proto.UnscannedUploadRequest_Data{
				Data: &proto.DataChunk{
					Chunk: buffer[:n],
				},
			},
		}

		if err := stream.Send(req); err != nil {
			return &proto.UnscannedUploadCompleteResponse{
				Errors: []*cerrorpb.Error{{
					Code:    cerror.InternalServerError,
					Message: err.Error(),
				}},
			}
		}
	}

	// Close the stream and receive the server's response
	resp, err := stream.CloseAndRecv()
	if err != nil {
		return &proto.UnscannedUploadCompleteResponse{
			Errors: []*cerrorpb.Error{{
				Code:    cerror.InternalServerError,
				Message: err.Error(),
			}},
		}
	}
	return resp
}

func (c *StoreClient) AddUpload(snapName string, entryId uuid.UUID, status string, accountId uuid.UUID, unscannedFileName string) *proto.AddUploadResponse {
	req := &proto.AddUploadRequest{
		SnapName:          snapName,
		EntryId:           entryId.String(),
		Status:            status,
		AccountId:         accountId.String(),
		UnscannedFileName: unscannedFileName,
	}

	resp, err := c.client.AddUpload(context.Background(), req)
	if err != nil {
		resp = &proto.AddUploadResponse{
			Errors: []*cerrorpb.Error{{
				Code:    cerror.InternalServerError,
				Message: err.Error()},
			},
		}
	}
	return resp
}

func (c *StoreClient) SnapDownload(revisionId string) *proto.SnapDownloadCompleteResponse {
	req := &proto.SnapDownloadRequest{
		RevisionId: revisionId,
	}

	stream, err := c.client.SnapDownload(context.Background(), req)
	if err != nil {
		// if err != nil we should read the stream (if we can) to get the actual error and then return it
		if stream != nil {
			if resp, recvErr := stream.Recv(); recvErr == nil && resp != nil && len(resp.Errors) > 0 {
				logrus.Errorf("error starting grpc download stream, received: %v", resp.Errors)
				return &proto.SnapDownloadCompleteResponse{Errors: resp.Errors}
			}
		}
		logrus.Errorf("error starting grpc download stream: %v", err)
		return &proto.SnapDownloadCompleteResponse{
			Errors: []*cerrorpb.Error{{
				Code:    cerror.InternalServerError,
				Message: err.Error(),
			}},
		}
	}

	// create buffer for snap data
	var fileData bytes.Buffer
	response := &proto.SnapDownloadCompleteResponse{}

	// loop over stream until EOF
	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			logrus.Errorf("error receiving grpc download stream: %v", err)
			return &proto.SnapDownloadCompleteResponse{
				Errors: []*cerrorpb.Error{{
					Code:    cerror.InternalServerError,
					Message: err.Error(),
				}},
			}
		}

		// Check for errors embedded in the response message
		if len(resp.Errors) > 0 {
			logrus.Errorf("received grpc stream error response: %v", resp.Errors)
			return &proto.SnapDownloadCompleteResponse{
				Errors: resp.Errors,
			}
		}

		// first message contains revision metadata
		if initial := resp.GetInitial(); initial != nil {
			logrus.Debugf("received revision metadata: %v", initial.Revision)
			response.Revision = initial.Revision
		} else if data := resp.GetData(); data != nil {
			fileData.Write(data.Chunk)
		}
	}

	response.Data = fileData.Bytes()
	return response
}

func (c *StoreClient) GetUploadStatus(uploadId string) *proto.GetUploadStatusResponse {
	req := &proto.GetUploadStatusRequest{
		UploadId: uploadId,
	}

	resp, err := c.client.GetUploadStatus(context.Background(), req)
	if err != nil {
		resp = &proto.GetUploadStatusResponse{
			Errors: []*cerrorpb.Error{{
				Code:    cerror.InternalServerError,
				Message: err.Error()},
			},
		}
	}
	return resp
}

func (c *StoreClient) AddRevision(snapName string, sha3384 string, size uint64, architectures []string, tracksAndChannels []string, unscannedFileName string, minioFilePath string) *proto.AddRevisionResponse {
	req := &proto.AddRevisionRequest{
		SnapName:          snapName,
		Sha3_384:          sha3384,
		Size:              size,
		Architectures:     architectures,
		TracksAndChannels: tracksAndChannels,
		UnscannedFileName: unscannedFileName,
		MinioFilePath:     minioFilePath,
	}

	resp, err := c.client.AddRevision(context.Background(), req)
	if err != nil {
		resp = &proto.AddRevisionResponse{
			Errors: []*cerrorpb.Error{{
				Code:    cerror.InternalServerError,
				Message: err.Error()},
			},
		}
	}
	return resp
}

func (c *StoreClient) GetObjectCustomMetadata(bucket string, objectKey string) *proto.GetObjectCustomMetadataResponse {
	req := &proto.GetObjectCustomMetadataRequest{
		Bucket:    bucket,
		ObjectKey: objectKey,
	}

	resp, err := c.client.GetObjectCustomMetadata(context.Background(), req)
	if err != nil {
		return &proto.GetObjectCustomMetadataResponse{
			Errors: []*cerrorpb.Error{{
				Code:    cerror.InternalServerError,
				Message: err.Error()},
			},
		}
	}
	return resp
}
