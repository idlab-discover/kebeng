package logic

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"time"

	"github.com/google/uuid"
	cerror "github.com/idlab-discover/kebeng/common/cerror"
	cerrorpb "github.com/idlab-discover/kebeng/common/cerror/proto"
	"github.com/idlab-discover/kebeng/services/store/internal/config"
	"github.com/idlab-discover/kebeng/services/store/internal/models"
	"github.com/idlab-discover/kebeng/services/store/internal/objectstore"
	"github.com/idlab-discover/kebeng/services/store/internal/repository"
	proto "github.com/idlab-discover/kebeng/services/store/proto"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var _ proto.StoreServiceServer = (*StoreLogic)(nil)

type StoreLogic struct {
	config *config.Config
	proto.UnimplementedStoreServiceServer
	repo repository.ISnapsRepository
	obs  objectstore.IObjectStore
}

func NewStoreLogic(repo repository.ISnapsRepository, config *config.Config, obj objectstore.IObjectStore) *StoreLogic {
	return &StoreLogic{repo: repo, config: config, obs: obj}
}

func (s *StoreLogic) RegisterSnapName(ctx context.Context, req *proto.RegisterSnapNameRequest) (*proto.RegisterSnapNameResponse, error) {
	el := cerror.NewErrorList()

	if req.SnapName == "" {
		el.Add(cerror.MissingField, "snap name is required")
		return &proto.RegisterSnapNameResponse{Errors: el.ConvertToProtoErrorList()}, nil
	}

	// TODO: check if snap name is valid (it should only have ASCII lowercase letters, numbers, and hyphens, and must have at least one letter)

	// First check if the snap name is already registered
	snapEntry, cerr := s.repo.GetEntryByName(req.SnapName, nil, el)
	if cerr != nil && cerr.GetCode() != cerror.ResourceNotFound {
		el.AddCustomError(cerr)
		return &proto.RegisterSnapNameResponse{Errors: el.ConvertToProtoErrorList()}, nil
	}

	// TODO: if dryRun is true, but snap name is not registered, what should we return? -> right now we return an empty string

	// If dryRun is true, we only check if the snap name is already registered
	if req.DryRun {
		if snapEntry != nil {
			return &proto.RegisterSnapNameResponse{Id: snapEntry.ID.String(), SnapName: req.SnapName}, nil // Id will be set to empty string, docs say it should be null, but nil can't be assigned to string -> TODO: see later if this is a problem
		} else {
			return &proto.RegisterSnapNameResponse{SnapName: ""}, nil // Return an empty string if the snap name is not registered
		}
	}
	// If dryRun is false, but snap name is already registered, return an error
	if snapEntry != nil {
		el.Add(cerror.AlreadyRegistered, "snap name is already registered")
		return &proto.RegisterSnapNameResponse{Errors: el.ConvertToProtoErrorList()}, nil
	}

	// If there is no snap with the same name and dry_run == false, register the snap name
	accountId, err1 := uuid.Parse(req.AccountId)
	if err1 != nil {
		el.Add(cerror.InvalidField, "invalid account id")
		return &proto.RegisterSnapNameResponse{Errors: el.ConvertToProtoErrorList()}, nil
	}
	snapEntry, cerr = s.repo.RegisterSnap(req.SnapName, req.IsPrivate, req.Store, accountId)
	if cerr != nil {
		el.AddCustomError(cerr)
		return &proto.RegisterSnapNameResponse{Errors: el.ConvertToProtoErrorList()}, nil
	}

	// Add "latest" track to the snap entry
	snapTrack, cerr := s.repo.AddTrack(snapEntry.ID, "latest")
	if cerr != nil {
		el.AddCustomError(cerr)
		return &proto.RegisterSnapNameResponse{Errors: el.ConvertToProtoErrorList()}, nil
	}

	// Add default channels for the "latest" track to the snap entry
	cerr = s.repo.AddDefaultChannels(snapEntry.ID, snapTrack.ID)
	if cerr != nil {
		el.AddCustomError(cerr)
		return &proto.RegisterSnapNameResponse{Errors: el.ConvertToProtoErrorList()}, nil
	}

	return &proto.RegisterSnapNameResponse{Id: snapEntry.ID.String(), SnapName: snapEntry.Name}, nil
}

// GetEntries retrieves a list of snap entries based on the provided request.
// It supports fetching entries either by their ID or by their snap name.
// If an entry is found, it is added to the response along with its snap name, type, confinement, base, and private fields.
// If an entry is not found or an error occurs during retrieval, an appropriate error is added to the response.
//
// Parameters:
//   - ctx: The context for the request, used for cancellation and deadlines.
//   - req: The request struct containing the list of entries to retrieve.
//
// Returns:
//   - *proto.GetEntriesResponse: The response containing the list of found entries and any cerror encountered.
//   - error: This error is only not nil of proto fails. Errors while retrieving entries are added to the GetEntriesResponse.
func (s *StoreLogic) GetEntries(ctx context.Context, req *proto.GetEntriesRequest) (*proto.GetEntriesResponse, error) {
	el := cerror.NewErrorList()
	foundEntries := make([]*proto.GetEntryResponse, 0)

	for _, entry := range req.Entries {
		// First try to retrieve the entry by its ID
		if entry.Id != nil && *entry.Id != "" {
			id, err := uuid.Parse(*entry.Id)
			if err != nil {
				logrus.Errorf("failed to parse UUID '%s':", *entry.Id)
				el.Add(cerror.InvalidField, fmt.Sprintf("invalid UUID format for id '%s'", *entry.Id))
				continue
			}

			snapEntry, cerr2 := s.repo.GetEntryById(id, nil, el)
			if cerr2 != nil {
				// Already logged in GetEntryById (repository
				el.AddCustomError(cerr2)
				continue
			}

			foundEntries = append(foundEntries, &proto.GetEntryResponse{
				Id:          snapEntry.ID.String(),
				SnapName:    snapEntry.Name,
				Type:        snapEntry.Type,
				Confinement: snapEntry.Confinement,
				Base:        snapEntry.Base,
				Private:     snapEntry.Private,
				PublisherId: snapEntry.AccountID.String(),
				Price:       snapEntry.Price,
				Status:      snapEntry.Status,
				Since:       timestamppb.New(snapEntry.CreatedAt),
				IconUrl:     snapEntry.IconURL,
			})

			// If ID is not given, try to retrieve the entry by its name
		} else if entry.Name != nil && *entry.Name != "" {
			snapEntry, cerr := s.repo.GetEntryByName(*entry.Name, nil, el)
			if cerr != nil {
				// Already logged in GetEntryByName (repository)
				el.AddCustomError(cerr)
				continue
			}

			foundEntries = append(foundEntries, &proto.GetEntryResponse{
				Id:          snapEntry.ID.String(),
				SnapName:    snapEntry.Name,
				Type:        snapEntry.Type,
				Confinement: snapEntry.Confinement,
				Base:        snapEntry.Base,
				Private:     snapEntry.Private,
				PublisherId: snapEntry.AccountID.String(),
				Price:       snapEntry.Price,
				Status:      snapEntry.Status,
				Since:       timestamppb.New(snapEntry.CreatedAt),
				IconUrl:     snapEntry.IconURL,
			})

		} else {
			el.Add(cerror.MissingField, "id or name is required")
		}
	}
	return &proto.GetEntriesResponse{Entries: foundEntries, Errors: el.ConvertToProtoErrorList()}, nil
}

// GetEntryById retrieves a single snap entry by its ID.
// If the snap entry is found, it is added to the response along with its snap name, type, confinement, base, and private fields.
// If the snap entry is not found or an error occurs during retrieval, an appropriate error is added to the response.
//
// Parameters:
//   - ctx: The context for the request, used for cancellation and deadlines.
//   - req: The request containing the ID of the snap entry to retrieve.
//
// Returns:
//   - *proto.GetEntryResponse: The response containing the found snap entry and any cerror encountered.
//   - error: This error is only not nil of proto fails. Errors while retrieving entry are added to the GetEntriesResponse.
func (s *StoreLogic) GetEntryById(ctx context.Context, req *proto.GetEntryRequest) (*proto.GetEntryResponse, error) {
	el := cerror.NewErrorList()
	if req.Id == nil || *req.Id == "" {
		el.Add(cerror.MissingField, "id is required")
		return &proto.GetEntryResponse{Errors: el.ConvertToProtoErrorList()}, nil
	}

	id, err := uuid.Parse(*req.Id)
	if err != nil {
		logrus.Error(err)
		el.Add(cerror.InvalidField, "invalid UUID format")
		return &proto.GetEntryResponse{Errors: el.ConvertToProtoErrorList()}, nil
	}

	snapEntry, cerr2 := s.repo.GetEntryById(id, nil, el)
	if cerr2 != nil {
		// Already logged in GetEntryById (repository)
		el.AddCustomError(cerr2)
		return &proto.GetEntryResponse{Errors: el.ConvertToProtoErrorList()}, nil
	}

	return &proto.GetEntryResponse{
		Id:          snapEntry.ID.String(),
		SnapName:    snapEntry.Name,
		Type:        snapEntry.Type,
		Confinement: snapEntry.Confinement,
		Base:        snapEntry.Base,
		Private:     snapEntry.Private,
		PublisherId: snapEntry.AccountID.String(),
		Price:       snapEntry.Price,
		Status:      snapEntry.Status,
		Since:       timestamppb.New(snapEntry.CreatedAt),
		IconUrl:     snapEntry.IconURL,
	}, nil
}

// GetEntryByName retrieves a single snap entry by its name.
// If the snap entry is found, it is added to the response along with its snap name, type, confinement, base, and private fields.
// If the snap entry is not found or an error occurs during retrieval, an appropriate error is added to the response.
//
// Parameters:
//   - ctx: The context for the request, used for cancellation and deadlines.
//   - req: The request containing the name of the snap entry to retrieve.
//
// Returns:
//   - *proto.GetEntryResponse: The response containing the found snap entry and any cerror encountered.
//   - error: An error if the operation fails.
func (s *StoreLogic) GetEntryByName(ctx context.Context, req *proto.GetEntryRequest) (*proto.GetEntryResponse, error) {
	el := cerror.NewErrorList()
	if req.Name == nil || *req.Name == "" {
		el.Add(cerror.MissingField, "name is required")
		return &proto.GetEntryResponse{Errors: el.ConvertToProtoErrorList()}, nil
	}

	snapEntry, cerr := s.repo.GetEntryByName(*req.Name, nil, el)
	if cerr != nil {
		el.AddCustomError(cerr)
		return &proto.GetEntryResponse{Errors: el.ConvertToProtoErrorList()}, nil
	}

	return &proto.GetEntryResponse{
		Id:          snapEntry.ID.String(),
		SnapName:    snapEntry.Name,
		Type:        snapEntry.Type,
		Confinement: snapEntry.Confinement,
		Base:        snapEntry.Base,
		Private:     snapEntry.Private,
		PublisherId: snapEntry.AccountID.String(),
		Price:       snapEntry.Price,
		Status:      snapEntry.Status,
		Since:       timestamppb.New(snapEntry.CreatedAt),
		IconUrl:     snapEntry.IconURL,
	}, nil
}

// GetRevisions retrieves a list of revisions based on the provided request.
// It supports fetching revisions either by their ID or by their snap name and sequence number.
// If a revision is found, it is added to the response along with its snap name and sequence number.
// If a revision is not found or an error occurs during retrieval, an appropriate error is added to the response.
//
// Parameters:
//   - ctx: The context for the request, used for cancellation and deadlines.
//   - req: The request containing the list of revisions to retrieve.
//
// Returns:
//   - *proto.GetRevisionsResponse: The response containing the list of found revisions and any cerror encountered.
//   - error: This error is only not nil of proto fails. Errors while retrieving revisions are added to the GetEntriesResponse.
func (s *StoreLogic) GetRevisions(ctx context.Context, req *proto.GetRevisionsRequest) (*proto.GetRevisionsResponse, error) {
	el := cerror.NewErrorList()
	foundRevisions := make([]*proto.GetRevisionResponse, 0)

	// Each GetRevisionRequest will contain either id or snapName and sequence
	for _, revision := range req.Revisions {
		// First check if id is provided
		if revision.Id != "" {
			id, err1 := uuid.Parse(revision.Id)
			if err1 != nil {
				logrus.Errorf("failed to parse UUID '%s':", revision.Id)
				el.Add(cerror.InvalidField, fmt.Sprintf("invalid UUID format for id '%s'", revision.Id))
				continue
			}
			rev, cerr := s.repo.GetRevisionById(id)
			if cerr != nil {
				// Already logged in GetRevisionById (repository)
				el.AddCustomError(cerr)
				continue
			}

			entry, cerr := s.repo.GetEntryById(rev.SnapEntryID, nil, el)
			if cerr != nil {
				// Already logged in GetEntryById (repository)
				el.AddCustomError(cerr)
				continue
			}

			foundRevisions = append(foundRevisions, &proto.GetRevisionResponse{
				Id:                     rev.ID.String(),
				CreatedAt:              timestamppb.New(rev.CreatedAt),
				UpdatedAt:              timestamppb.New(rev.UpdatedAt),
				DeletedAt:              timePointerToTimestamp(rev.DeletedAt),
				BuildAssertionFilename: pointerToString(rev.BuildAssertionFileName),
				Sha3_384:               pointerToString(rev.SHA3_384),
				Sha3_384Encoded:        pointerToString(rev.SHA3_384_Encoded),
				Size:                   uint64(*rev.Size),
				SequenceNumber:         uint64(*rev.SequenceNumber),
				Architectures:          rev.Architectures,
				Status:                 pointerToString(rev.Status),
				Version:                pointerToString(rev.Version),
				EntryId:                rev.SnapEntryID.String(),
				TrackId:                rev.SnapTrackID.String(),
				ChannelId:              rev.SnapChannelID.String(),
				SnapName:               entry.Name,
			})

			// If id is not provided, check if snapName and sequence are provided
		} else if revision.SnapName != "" && revision.Sequence != 0 {
			rev, cerr := s.repo.GetRevisionByNameAndSequence(revision.SnapName, uint(revision.Sequence))
			if cerr != nil {
				el.AddCustomError(cerr)
				continue
			}

			entry, cerr := s.repo.GetEntryById(rev.SnapEntryID, nil, el)
			if cerr != nil {
				// Already logged in GetEntryById (repository)
				el.AddCustomError(cerr)
				continue
			}

			foundRevisions = append(foundRevisions, &proto.GetRevisionResponse{
				Id:                     rev.ID.String(),
				CreatedAt:              timestamppb.New(rev.CreatedAt),
				UpdatedAt:              timestamppb.New(rev.UpdatedAt),
				DeletedAt:              timePointerToTimestamp(rev.DeletedAt),
				BuildAssertionFilename: pointerToString(rev.BuildAssertionFileName),
				Sha3_384:               pointerToString(rev.SHA3_384),
				Sha3_384Encoded:        pointerToString(rev.SHA3_384_Encoded),
				Size:                   uint64(*rev.Size),
				SequenceNumber:         uint64(*rev.SequenceNumber),
				Architectures:          rev.Architectures,
				Status:                 pointerToString(rev.Status),
				Version:                pointerToString(rev.Version),
				EntryId:                rev.SnapEntryID.String(),
				TrackId:                rev.SnapTrackID.String(),
				ChannelId:              rev.SnapChannelID.String(),
				SnapName:               entry.Name,
			})

		} else {
			if revision.Id == "" && (revision.SnapName == "" || revision.Sequence == 0) {
				el.Add(cerror.MissingField, "id or snap name and sequence are required")
			}
			if revision.SnapName == "" && revision.Id == "" {
				el.Add(cerror.MissingField, "snap name is required")
			}
			if revision.Sequence == 0 && revision.Id == "" {
				el.Add(cerror.MissingField, "sequence is required")
			}
		}
	}
	return &proto.GetRevisionsResponse{Revisions: foundRevisions, Errors: el.ConvertToProtoErrorList()}, nil
}

// GetRevisionByNameAndSequence returns a single revision by snap name and sequence number
func (s *StoreLogic) GetRevisionByNameAndSequence(ctx context.Context, req *proto.GetRevisionRequest) (*proto.GetRevisionResponse, *cerror.CustomError) {
	el := cerror.NewErrorList()

	if req.SnapName == "" {
		el.Add(cerror.MissingField, "snap name is required")
		return &proto.GetRevisionResponse{Errors: el.ConvertToProtoErrorList()}, nil
	}

	if req.Sequence == 0 {
		el.Add(cerror.MissingField, "sequence is required")
		return &proto.GetRevisionResponse{Errors: el.ConvertToProtoErrorList()}, nil
	}

	entry, cerr := s.repo.GetEntryByName(req.SnapName, nil, el)
	if cerr != nil {
		// Already logged in GetEntryByName (repository)
		el.AddCustomError(cerr)
		return &proto.GetRevisionResponse{Errors: el.ConvertToProtoErrorList()}, cerr
	}

	rev, cerr := s.repo.GetRevisionByNameAndSequence(entry.Name, uint(req.Sequence))
	if cerr != nil {
		// Already logged in GetRevisionByNameAndSequence (repository)
		el.AddCustomError(cerr)
		return &proto.GetRevisionResponse{Errors: el.ConvertToProtoErrorList()}, cerr
	}

	return &proto.GetRevisionResponse{
		Id:                     rev.ID.String(),
		CreatedAt:              timestamppb.New(rev.CreatedAt),
		UpdatedAt:              timestamppb.New(rev.UpdatedAt),
		DeletedAt:              timePointerToTimestamp(rev.DeletedAt),
		BuildAssertionFilename: pointerToString(rev.BuildAssertionFileName),
		Sha3_384:               pointerToString(rev.SHA3_384),
		Sha3_384Encoded:        pointerToString(rev.SHA3_384_Encoded),
		Size:                   uint64(*rev.Size),
		SequenceNumber:         uint64(*rev.SequenceNumber),
		Architectures:          rev.Architectures,
		Status:                 pointerToString(rev.Status),
		Version:                pointerToString(rev.Version),
		EntryId:                rev.SnapEntryID.String(),
		TrackId:                rev.SnapTrackID.String(),
		ChannelId:              rev.SnapChannelID.String(),
		SnapName:               entry.Name,
	}, nil
}

func (s *StoreLogic) GetEntriesByAccountId(ctx context.Context, req *proto.GetEntriesByAccountIdRequest) (*proto.GetEntriesResponse, error) {
	el := cerror.NewErrorList()
	if req.AccountId == "" {
		el.Add(cerror.MissingField, "account id is required")
		return &proto.GetEntriesResponse{Errors: el.ConvertToProtoErrorList()}, nil
	}
	accId, err := uuid.Parse(req.AccountId)
	if err != nil {
		logrus.Error(err)
		el.Add(cerror.InvalidField, "invalid UUID format")
		return &proto.GetEntriesResponse{Errors: el.ConvertToProtoErrorList()}, nil
	}

	entries, cerr := s.repo.GetEntriesByAccountId(accId, nil, el)
	if cerr != nil {
		el.AddCustomError(cerr)
		return &proto.GetEntriesResponse{Errors: el.ConvertToProtoErrorList()}, nil
	}

	foundEntries := make([]*proto.GetEntryResponse, len(entries))
	for i, entry := range entries {
		foundEntries[i] = &proto.GetEntryResponse{
			Id:          entry.ID.String(),
			SnapName:    entry.Name,
			Type:        entry.Type,
			Confinement: entry.Confinement,
			Base:        entry.Base,
			Private:     entry.Private,
			PublisherId: entry.AccountID.String(),
			Price:       entry.Price,
			Status:      entry.Status,
			Since:       timestamppb.New(entry.CreatedAt),
			IconUrl:     entry.IconURL,
		}
	}
	return &proto.GetEntriesResponse{Entries: foundEntries}, nil
}

// returns multiple Revisions by their entry ids
// if a revision is not found a response with EntryId filled in and len(Errors) > 0 is put in the response
func (s *StoreLogic) GetRevisionsByEntryIds(ctx context.Context, req *proto.GetRevisionsByEntryIdRequests) (*proto.GetRevisionsByEntryIdResponses, error) {
	el := cerror.NewErrorList()
	responses := make([]*proto.GetRevisionsByEntryIdResponse, 0)
	for _, entryIdReq := range req.GetRequests() {
		if entryIdReq.EntryId == "" {
			el.Add(cerror.MissingField, "entry id is required")
			continue
		}

		entryId, err := uuid.Parse(entryIdReq.EntryId)
		if err != nil {
			logrus.Errorf("failed to parse UUID '%s':", entryIdReq.EntryId)
			el.Add(cerror.InvalidField, "invalid UUID format")
			continue
		}
		entry, cerr := s.repo.GetEntryById(entryId, nil, el)
		if cerr != nil {
			logrus.Errorf("Failed to get entry from database: %v", cerr)
			el.AddCustomError(cerr)
			responses = append(responses, &proto.GetRevisionsByEntryIdResponse{
				EntryId: entryId.String(),
				Errors:  el.ConvertToProtoErrorList(),
			})
			continue
		}
		revisions, cerr := s.repo.GetRevisionsByEntryId(entryId)
		if cerr != nil {
			logrus.Debugf("Failed to get revisions from database: %v", cerr)
			el.AddCustomError(cerr)
			// add empty response to keep the order of responses
			responses = append(responses, &proto.GetRevisionsByEntryIdResponse{
				EntryId: entryId.String(),
				Errors:  el.ConvertToProtoErrorList(),
			})
			continue
		}
		// revision were found so convert them in response format
		revisionsProto := make([]*proto.GetRevisionResponse, len(revisions))
		for i, rev := range revisions {
			revisionsProto[i] = &proto.GetRevisionResponse{
				Id:                     rev.ID.String(),
				CreatedAt:              timestamppb.New(rev.CreatedAt),
				UpdatedAt:              timestamppb.New(rev.UpdatedAt),
				DeletedAt:              timePointerToTimestamp(rev.DeletedAt),
				BuildAssertionFilename: pointerToString(rev.BuildAssertionFileName),
				Sha3_384:               pointerToString(rev.SHA3_384),
				Sha3_384Encoded:        pointerToString(rev.SHA3_384_Encoded),
				Size:                   uint64(*rev.Size),
				SequenceNumber:         uint64(*rev.SequenceNumber),
				Architectures:          rev.Architectures,
				Status:                 pointerToString(rev.Status),
				Version:                pointerToString(rev.Version),
				EntryId:                rev.SnapEntryID.String(),
				TrackId:                rev.SnapTrackID.String(),
				ChannelId:              rev.SnapChannelID.String(),
				SnapName:               entry.Name,
			}
		}
		// add to response
		responses = append(responses, &proto.GetRevisionsByEntryIdResponse{
			EntryId:   entryId.String(),
			Revisions: revisionsProto,
		})
	}
	return &proto.GetRevisionsByEntryIdResponses{Responses: responses}, nil
}

func (s *StoreLogic) GetLatestRevision(ctx context.Context, req *proto.GetLatestRevisionRequest) (*proto.GetRevisionResponse, error) {
	el := cerror.NewErrorList()
	if req.SnapName == "" {
		el.Add(cerror.MissingField, "snap name is required")
		return &proto.GetRevisionResponse{Errors: el.ConvertToProtoErrorList()}, nil
	}

	revision, cerr := s.repo.GetLatestRevision(req.SnapName, req.Track, req.Channel)
	if cerr != nil {
		logrus.Error(cerr)
		el.AddCustomError(cerr)
		return &proto.GetRevisionResponse{Errors: el.ConvertToProtoErrorList()}, nil
	}

	return &proto.GetRevisionResponse{
		Id:                     revision.ID.String(),
		CreatedAt:              timestamppb.New(revision.CreatedAt),
		UpdatedAt:              timestamppb.New(revision.UpdatedAt),
		DeletedAt:              timePointerToTimestamp(revision.DeletedAt),
		BuildAssertionFilename: pointerToString(revision.BuildAssertionFileName),
		Sha3_384:               pointerToString(revision.SHA3_384),
		Sha3_384Encoded:        pointerToString(revision.SHA3_384_Encoded),
		Size:                   uint64(*revision.Size),
		SequenceNumber:         uint64(*revision.SequenceNumber),
		Architectures:          revision.Architectures,
		Status:                 pointerToString(revision.Status),
		Version:                pointerToString(revision.Version),
		EntryId:                revision.SnapEntryID.String(),
		TrackId:                revision.SnapTrackID.String(),
		ChannelId:              revision.SnapChannelID.String(),
		SnapName:               req.SnapName,
	}, nil
}

func (s *StoreLogic) SnapDownload(req *proto.SnapDownloadRequest, stream proto.StoreService_SnapDownloadServer) error {
	el := cerror.NewErrorList()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if req.RevisionId == "" {
		return stream.Send(&proto.SnapDownloadResponse{
			Errors: []*cerrorpb.Error{{
				Code:    cerror.MissingField,
				Message: "revision id is required",
			}},
		})
	}

	parsedRevisionId, err := uuid.Parse(req.RevisionId)
	if err != nil {
		logrus.Error(err)
		el.Add(cerror.InvalidField, "invalid UUID format")
		err := stream.Send(&proto.SnapDownloadResponse{
			Errors: el.ConvertToProtoErrorList(),
		})
		if err != nil {
			logrus.Error("failed to send error response: ", err)
			return err
		}
		return err
	}
	revision, cerr := s.repo.GetRevisionById(parsedRevisionId)
	if cerr != nil {
		el.AddCustomError(cerr)
		err := stream.Send(&proto.SnapDownloadResponse{
			Errors: el.ConvertToProtoErrorList(),
		})
		if err != nil {
			logrus.Error("failed to send error response: ", err)
			return err
		}
		return fmt.Errorf("failed to get revision by id: %v", cerr)
	}

	// retrieve file path inside minio to find correct snap package
	filePath, cerr := s.retrieveObjectStoreFilePath(revision, el)
	if cerr != nil {
		el.AddCustomError(cerr)
		err := stream.Send(&proto.SnapDownloadResponse{
			Errors: el.ConvertToProtoErrorList(),
		})
		if err != nil {
			logrus.Error("failed to send error response: ", err)
			return err
		}
		return fmt.Errorf("failed to retrieve snap from objectstore with filePath: %v", filePath)
	}

	snapFileReader, err := s.obs.GetSnapFileReader(ctx, filePath)
	if err != nil {
		el.Add(cerror.InternalServerError, "failed to get snap file reader")
		err := stream.Send(&proto.SnapDownloadResponse{
			Errors: el.ConvertToProtoErrorList(),
		})
		if err != nil {
			logrus.Error("failed to send error response: ", err)
			return err
		}
		return fmt.Errorf("failed to get revision by id: %v", cerr)
	}
	defer func(snapfileReader io.Reader) {
		err := snapFileReader.Close()
		if err != nil {
			logrus.Error("failed to close snap file reader: ", err)
		}
	}(snapFileReader)

	// define chunksize for reading the file
	const chunkSize = 64 * 1024
	buffer := make([]byte, chunkSize)

	// TODO: add snapName to Revision
	protoRevision := convertRevisionToProto(revision)

	// send the initial message with revision metadata.
	initialResp := &proto.SnapDownloadResponse{
		Payload: &proto.SnapDownloadResponse_Initial{
			Initial: &proto.InitialDownloadResponse{
				Revision: protoRevision,
			},
		},
		Errors: nil,
	}
	if err := stream.Send(initialResp); err != nil {
		logrus.Error("failed to send initial gRPC stream response: ", err)
		return err
	}

	for {
		n, err := snapFileReader.Read(buffer)
		if n > 0 {
			dataResp := &proto.SnapDownloadResponse{
				Payload: &proto.SnapDownloadResponse_Data{
					Data: &proto.DataChunk{
						Chunk: buffer[:n],
					},
				},
				Errors: nil,
			}
			if err := stream.Send(dataResp); err != nil {
				logrus.Error("failed to send data chunk: ", err)
				return err
			}
		}
		if err == io.EOF {
			break
		}
		// only check error down here since io.EOF is not an error and it can return some bytes and io.EOF at the same time, we would miss the last few bytes
		if err != nil {
			logrus.Error("failed to read snap file: ", err)
			return err
		}
	}
	return nil
}

func saveFileToTemp(snapFile io.Reader) (string, *cerror.CustomError) {
	// Generate random file name for the new uploaded file so it doesn't override the old file with same name
	snapFileId := uuid.New().String()
	newFileName := snapFileId + ".snap"

	out, err := os.Create(path.Join("/tmp", newFileName))
	if err != nil {
		return "", cerror.NewCustomError(cerror.InternalServerError, "Failed to create file")
	}
	defer func(out *os.File) {
		err := out.Close()
		if err != nil {
			logrus.Error("failed to close file: ", err)
		}

	}(out)

	_, err = io.Copy(out, snapFile)
	if err != nil {
		return "", cerror.NewCustomError(cerror.InternalServerError, "Failed to copy file")
	}

	return newFileName, nil
}

func (s *StoreLogic) AddUpload(ctx context.Context, req *proto.AddUploadRequest) (*proto.AddUploadResponse, error) {
	el := cerror.NewErrorList()

	// Check on empry fields
	if req.SnapName == "" {
		el.Add(cerror.MissingField, "snap name is required")
	}
	if req.EntryId == "" {
		el.Add(cerror.MissingField, "entry id is required")
	}
	if req.Status == "" {
		el.Add(cerror.MissingField, "status is required")
	}
	if req.AccountId == "" {
		el.Add(cerror.MissingField, "account id is required")
	}
	if len(*el) > 0 {
		return &proto.AddUploadResponse{Errors: el.ConvertToProtoErrorList()}, nil
	}

	// Parse UUIDs
	entryId, err := uuid.Parse(req.EntryId)
	if err != nil {
		logrus.Error(err)
		el.Add(cerror.InvalidField, "invalid UUID format")
		return &proto.AddUploadResponse{Errors: el.ConvertToProtoErrorList()}, nil
	}
	accountId, err := uuid.Parse(req.AccountId)
	if err != nil {
		logrus.Error(err)
		el.Add(cerror.InvalidField, "invalid UUID format")
		return &proto.AddUploadResponse{Errors: el.ConvertToProtoErrorList()}, nil
	}

	// Add upload to the database
	snapUpload, cerr := s.repo.AddUpload(req.SnapName, entryId, req.Status, accountId)
	if cerr != nil {
		el.AddCustomError(cerr)
		return &proto.AddUploadResponse{Errors: el.ConvertToProtoErrorList()}, nil
	}

	return &proto.AddUploadResponse{
		Id:       snapUpload.ID.String(),
		SnapName: snapUpload.SnapName,
		Status:   snapUpload.Status,
	}, nil
}

func (s *StoreLogic) UnscannedUpload(ctx context.Context, req *proto.UnscannedUploadRequest) (*proto.UnscannedUploadResponse, error) {
	el := cerror.NewErrorList()
	snapFileName, cerr := saveFileToTemp(bytes.NewReader(req.SnapFile))
	if cerr != nil {
		logrus.Errorf("failed to save file to temp storage: %v", cerr)
		el.AddCustomError(cerr)
		return &proto.UnscannedUploadResponse{Errors: el.ConvertToProtoErrorList()}, nil
	}

	tmpPath := path.Join(os.TempDir(), snapFileName)

	size, err := s.obs.SaveFileToBucket("unscanned", tmpPath)
	if err != nil {
		logrus.Errorf("failed to save file to object store: %v", err)
		el.Add(cerror.InternalServerError, "failed to save file to object store")
		return &proto.UnscannedUploadResponse{Errors: el.ConvertToProtoErrorList()}, nil
	}

	return &proto.UnscannedUploadResponse{TempFileName: tmpPath, FileSize: size}, nil
}

// ################# HELPERS #################

func (s *StoreLogic) retrieveObjectStoreFilePath(revision *models.SnapRevision, el *cerror.ErrorList) (string, *cerror.CustomError) {
	entry, cerr := s.repo.GetEntryById(revision.SnapEntryID, nil, el)
	if cerr != nil {
		return "", cerr
	}
	track, cerr := s.repo.GetTrackById(revision.SnapTrackID)
	if cerr != nil {
		return "", cerr
	}
	channel, cerr := s.repo.GetChannelById(revision.SnapChannelID)
	if cerr != nil {
		return "", cerr
	}
	return fmt.Sprintf("%s/%s/%s/%s_%d.snap", entry.Name, track.Name, channel.Name, entry.Name, uintPointerToUint(revision.SequenceNumber)), nil
}

func pointerToString(s *string) string {
	if s != nil {
		return *s
	}
	return ""
}

func timePointerToTimestamp(s *time.Time) *timestamppb.Timestamp {
	if s != nil {
		return timestamppb.New(*s)
	}
	return nil
}

func uintPointerToUint(s *uint) uint {
	if s != nil {
		return *s
	}
	return 0
}

func convertRevisionToProto(revision *models.SnapRevision) *proto.GetRevisionResponse {
	return &proto.GetRevisionResponse{
		Id:                     revision.ID.String(),
		CreatedAt:              timestamppb.New(revision.CreatedAt),
		UpdatedAt:              timestamppb.New(revision.UpdatedAt),
		DeletedAt:              timePointerToTimestamp(revision.DeletedAt),
		BuildAssertionFilename: pointerToString(revision.BuildAssertionFileName),
		Sha3_384:               pointerToString(revision.SHA3_384),
		Sha3_384Encoded:        pointerToString(revision.SHA3_384_Encoded),
		Size:                   uint64(*revision.Size),
		SequenceNumber:         uint64(*revision.SequenceNumber),
		Architectures:          revision.Architectures,
		Status:                 pointerToString(revision.Status),
		Version:                pointerToString(revision.Version),
	}
}
