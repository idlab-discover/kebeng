package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/idlab-discover/kebeng/services/store/client/model"
	"github.com/idlab-discover/kebeng/services/store/internal/config"
	proto "github.com/idlab-discover/kebeng/services/store/proto"

	"github.com/google/uuid"
	cerror "github.com/idlab-discover/kebeng/common/cerror"
	cerrorpb "github.com/idlab-discover/kebeng/common/cerror/proto"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health/grpc_health_v1"
)

const (
	maxDialAttempts = 5
	dialRetryDelay  = 2 * time.Second
	healthTimeout   = 2 * time.Second
)

type StoreClientInterface interface {
	Close()
	RegisterSnapName(snapName string, snapType string, confinement string, base string, isPrivate bool, status string, price float64, storeName string, iconUrl string, dryRun bool, accountId uuid.UUID) *proto.RegisterSnapNameResponse
	GetEntries(entries *proto.GetEntriesRequest) *proto.GetEntriesResponse
	GetRevisions(revisions *proto.GetRevisionsRequest) *proto.GetRevisionsResponse
	GetEntriesByAccountID(accountID string) *proto.GetEntriesResponse
	GetEntriesByQuery(query string, architectureList []string, channelList []string, confinementsList []string, fieldsList []string, private bool, publisherId string) *proto.GetEntriesResponse
	GetRevisionsByEntryIds(entryIds *proto.GetRevisionsByEntryIdRequests) *proto.GetRevisionsByEntryIdResponses
	GetLatestRevisionByTrackAndChannel(snapName, track, channel string) *proto.GetRevisionResponse
	SnapDownloadStream(revisionId string) (proto.StoreService_SnapDownloadClient, error)
	UnscannedUpload(ctx context.Context, snapFile io.Reader) *proto.UnscannedUploadCompleteResponse
	AddUpload(entryId uuid.UUID, accountId uuid.UUID, snapName string, status string, unscannedFileName string, revision uint32) *proto.AddUploadResponse
	GetUploadStatus(uploadId string) *proto.GetUploadStatusResponse
	AddRevision(snapName string, sha3_384_encoded string, size uint64, architectures []string, tracksAndChannels []string, unscannedFileName string) *proto.AddRevisionResponse
	GetObjectCustomMetadata(bucket string, objectKey string) *model.Metadata
	UpdateUploadStatus(uploadId string, status string, revision uint32, el *cerror.ErrorList) *proto.UpdateUploadStatusResponse
	UpdateSnapEntryWithMetadata(snapEntryId uuid.UUID, metadata *model.Metadata) *proto.UpdateEntryResponse
}

var _ StoreClientInterface = (*StoreClient)(nil)

type StoreClient struct {
	conn   *grpc.ClientConn
	client proto.StoreServiceClient
}

func NewStoreClientWithClient(client proto.StoreServiceClient) *StoreClient {
	return &StoreClient{client: client}
}

func NewStoreClient(storeHost string, storePort int) (*StoreClient, *cerror.CustomError) {
	logrus.Infof("Connecting to store service at %s:%d", storeHost, storePort)
	addr := config.GetStoreServiceAddress(storeHost, storePort)

	for attempt := 1; attempt <= maxDialAttempts; attempt++ {
		logrus.Infof("Attempt %d/%d: dialing store service at %s", attempt, maxDialAttempts, addr)
		conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			logrus.Warnf("Failed to create gRPC client: %v", err)
			time.Sleep(dialRetryDelay)
			continue
		}

		// run quick health check
		hc := grpc_health_v1.NewHealthClient(conn)
		ctx, cancel := context.WithTimeout(context.Background(), healthTimeout)
		defer cancel()

		resp, err := hc.Check(ctx, &grpc_health_v1.HealthCheckRequest{})
		if err == nil && resp.Status == grpc_health_v1.HealthCheckResponse_SERVING {
			logrus.Infof("Successfully connected to store service at %s", addr)
			return &StoreClient{conn, proto.NewStoreServiceClient(conn)}, nil
		}
		logrus.Warnf("Health check failed: %v", err)
		time.Sleep(dialRetryDelay)
	}
	return nil, cerror.NewCustomError(cerror.InternalServerError, fmt.Sprintf("failed to connect to store service after %d attempts", maxDialAttempts))
}

func (c *StoreClient) Close() {
	err := c.conn.Close()
	if err != nil {
		logrus.Errorf("error closing store client connection: %v", err)
	}
}

func (c *StoreClient) RegisterSnapName(snapName string, snapType string, confinement string, base string, isPrivate bool, status string, price float64, storeName string, iconUrl string, dryRun bool, accountId uuid.UUID) *proto.RegisterSnapNameResponse {
	if !checkValidName(snapName) {
		return &proto.RegisterSnapNameResponse{
			Errors: []*cerrorpb.Error{{
				Code:    cerror.InvalidField,
				Message: "snap name is invalid, it should only have ASCII lowercase letters, numbers, and hyphens, and must have at least one letter: " + snapName},
			},
		}
	}
	req := &proto.RegisterSnapNameRequest{
		SnapName:    snapName,
		SnapType:    snapType,
		Confinement: confinement,
		Base:        base,
		IsPrivate:   isPrivate,
		Status:      status,
		Price:       price,
		Store:       storeName,
		IconUrl:     iconUrl,
		DryRun:      dryRun,
		AccountId:   accountId.String(),
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

func (c *StoreClient) GetEntriesByQuery(query string, architectureList []string, channelList []string, confinementsList []string, fieldsList []string, private bool, publisherId string) *proto.GetEntriesResponse {
	req := &proto.GetEntriesByQueryRequest{
		Query: query,
		ArchitectureList: architectureList,
		ChannelList: channelList,
		ConfinementsList: confinementsList,
		FieldsList: fieldsList,
		PublisherId: publisherId,
		Private: private,
	}
	resp, err := c.client.GetEntriesByQuery(context.Background(), req)
	if err != nil {
		resp = &proto.GetEntriesResponse{
			Errors: []*cerrorpb.Error{{
				Code: cerror.InternalServerError,
				Message: err.Error(),
			}},
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

func (c *StoreClient) GetLatestRevisionByTrackAndChannel(snapName, track, channel string) *proto.GetRevisionResponse {
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
	buffer := make([]byte, 64*1024) // 1 MB buffer
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

func (c *StoreClient) AddUpload(entryId, accountId uuid.UUID, snapName, status, unscannedFileName string, revision uint32) *proto.AddUploadResponse {
	req := &proto.AddUploadRequest{
		EntryId:           entryId.String(),
		AccountId:         accountId.String(),
		UnscannedFileName: unscannedFileName,
		SnapName:          snapName,
		Status:            status,
		Revision:          revision,
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

func (c *StoreClient) SnapDownloadStream(revisionId string) (proto.StoreService_SnapDownloadClient, error) {
	req := &proto.SnapDownloadRequest{
		RevisionId: revisionId,
	}
	return c.client.SnapDownload(context.Background(), req)
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

// UpdateUploadStatus updates the status of an upload
// It takes the upload ID, status, revision number, and a list of errors
// Errors are stored in the database and are retrieved later by Snapcraft to check if any errors occurred during the upload
func (c *StoreClient) UpdateUploadStatus(uploadId string, status string, revision uint32, el *cerror.ErrorList) *proto.UpdateUploadStatusResponse {
	req := &proto.UpdateUploadStatusRequest{
		UploadId: uploadId,
		Status:   status,
		Revision: revision,
		Errors:   el.ConvertToProtoErrorList(),
	}

	resp, err := c.client.UpdateUploadStatus(context.Background(), req)
	if err != nil {
		resp = &proto.UpdateUploadStatusResponse{
			Errors: []*cerrorpb.Error{{
				Code:    cerror.InternalServerError,
				Message: err.Error()},
			},
		}
	}
	return resp
}

func (c *StoreClient) AddRevision(snapName string, sha3_384_encoded string, size uint64, architectures []string, tracksAndChannels []string, unscannedFileName string) *proto.AddRevisionResponse {
	el := cerror.NewErrorList()
	req := &proto.AddRevisionRequest{
		SnapName:          snapName,
		Sha3_384Encoded:   sha3_384_encoded,
		Size:              size,
		Architectures:     architectures,
		TracksAndChannels: tracksAndChannels,
		UnscannedFileName: unscannedFileName,
	}

	resp, err := c.client.AddRevision(context.Background(), req)
	if err != nil {
		el.Add(cerror.InternalServerError, err.Error())
		resp = &proto.AddRevisionResponse{
			Errors: el.ConvertToProtoErrorList(),
		}
	}
	return resp
}

func (c *StoreClient) GetObjectCustomMetadata(bucket string, objectKey string) *model.Metadata {
	el := cerror.NewErrorList()
	req := &proto.GetObjectCustomMetadataRequest{
		Bucket:    bucket,
		ObjectKey: objectKey,
	}

	resp, err := c.client.GetObjectCustomMetadata(context.Background(), req)
	if err != nil {
		el.Add(cerror.InternalServerError, err.Error())
		return &model.Metadata{
			Errors: el,
		}
	}

	// convert the response to a model.Metadata object
	plugs, cerr := deserializePlugString(resp.Plugs)
	if cerr != nil {
		el.AddCustomError(cerr)
	}
	slots, cerr := deserializeSlotString(resp.Slots)
	if cerr != nil {
		el.AddCustomError(cerr)
	}
	metadata := &model.Metadata{
		Sha3_384Encoded: resp.Sha3_384Encoded,
		Name:            resp.Name,
		Version:         resp.Version,
		Type:            resp.Type,
		Summary:         resp.Summary,
		Description:     resp.Description,
		Confinement:     resp.Confinement,
		Base:            resp.Base,
		Grade:           resp.Grade,
		Architectures:   resp.Architectures,
		Plugs:           plugs,
		Slots:           slots,
		RefreshControl:  resp.RefreshControl,
		Errors:          el,
	}

	return metadata
}

func (c *StoreClient) UpdateSnapEntryWithMetadata(snapEntryId uuid.UUID, metadata *model.Metadata) *proto.UpdateEntryResponse {
	req := &proto.UpdateSnapEntryWithMetadataRequest{
		EntryId:       snapEntryId.String(),
		Name:          metadata.Name,
		Confinement:   metadata.Confinement,
		Base:          metadata.Base,
		Architectures: metadata.Architectures,
		Grade:         metadata.Grade,
		Version:       metadata.Version,
		Summary:       metadata.Summary,
		Description:   metadata.Description,
		Errors:        metadata.Errors.ConvertToProtoErrorList(),
	}

	resp, err := c.client.UpdateSnapEntryWithMetadata(context.Background(), req)
	if err != nil {
		return &proto.UpdateEntryResponse{
			Errors: []*cerrorpb.Error{{
				Code:    cerror.InternalServerError,
				Message: err.Error()},
			},
		}
	}
	return resp
}

// =============== HELPER FUNCTIONS ===================
func checkValidName(name string) bool {
	if name == "" {
		return false
	}

	hasLetter := false
	for i := range len(name) {
		c := name[i]
		switch {
		case c >= 'a' && c <= 'z':
			hasLetter = true
		case c >= '0' && c <= '9', c == '-':
			// allowed
		default:
			return false
		}
	}
	return hasLetter
}

func deserializePlugString(data string) (model.Plugs, *cerror.CustomError) {
	var plugs model.Plugs
	err := json.Unmarshal([]byte(data), &plugs)
	if err != nil {
		cerr := cerror.NewCustomError(cerror.BadRequest, fmt.Sprintf("failed to deserialize plugs: %s", err.Error()))
		logrus.Errorf(cerr.GetMessage())
		return nil, cerr
	}
	return plugs, nil
}

func deserializeSlotString(data string) (model.Slots, *cerror.CustomError) {
	var slots model.Slots
	err := json.Unmarshal([]byte(data), &slots)
	if err != nil {
		cerr := cerror.NewCustomError(cerror.BadRequest, fmt.Sprintf("failed to deserialize slots: %s", err.Error()))
		logrus.Errorf(cerr.GetMessage())
		return nil, cerr
	}
	return slots, nil
}
