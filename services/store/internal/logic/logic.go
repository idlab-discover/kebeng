package logic

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path"
	"strings"
	"time"

	"github.com/google/uuid"
	cerror "github.com/idlab-discover/kebeng/common/cerror"
	"github.com/idlab-discover/kebeng/services/store/internal/config"
	"github.com/idlab-discover/kebeng/services/store/internal/models"
	"github.com/idlab-discover/kebeng/services/store/internal/objectstore"
	"github.com/idlab-discover/kebeng/services/store/internal/repository"
	proto "github.com/idlab-discover/kebeng/services/store/proto"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/sha3"
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
		// Already logged in GetEntryByName (repository)
		return &proto.RegisterSnapNameResponse{Errors: el.ConvertToProtoErrorList()}, nil
	}

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
	snapEntry, cerr = s.repo.RegisterSnap(req.SnapName, req.IsPrivate, req.Store, accountId, el)
	if cerr != nil {
		// Already logged in RegisterSnap (repository)
		return &proto.RegisterSnapNameResponse{Errors: el.ConvertToProtoErrorList()}, nil
	}

	// Add "latest" track to the snap entry
	snapTrack, cerr := s.repo.AddTrack(snapEntry.ID, "latest", el)
	if cerr != nil {
		// Already logged in AddTrack (repository)
		return &proto.RegisterSnapNameResponse{Errors: el.ConvertToProtoErrorList()}, nil
	}

	// Add default channels for the "latest" track to the snap entry
	cerr = s.repo.AddDefaultChannels(snapEntry.ID, snapTrack.ID, el)
	if cerr != nil {
		// Already logged in AddDefaultChannels (repository)
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
				cerr := cerror.NewCustomError(cerror.InvalidField, fmt.Sprintf("invalid UUID format: %s", *entry.Id))
				logrus.Error(cerr)
				el.AddCustomError(cerr)
				continue
			}

			snapEntry, cerr := s.repo.GetEntryById(id, nil, el)
			if cerr != nil {
				// Already logged in GetEntryById (repository
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
			cerr := cerror.NewCustomError(cerror.MissingField, "id or name is required")
			logrus.Error(cerr)
			el.AddCustomError(cerr)
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
		cerr := cerror.NewCustomError(cerror.MissingField, "id is required")
		logrus.Error(cerr)
		el.AddCustomError(cerr)
		return &proto.GetEntryResponse{Errors: el.ConvertToProtoErrorList()}, nil
	}

	id, err := uuid.Parse(*req.Id)
	if err != nil {
		cerr := cerror.NewCustomError(cerror.InvalidField, fmt.Sprintf("invalid UUID format: %s", *req.Id))
		logrus.Error(cerr)
		el.AddCustomError(cerr)
		return &proto.GetEntryResponse{Errors: el.ConvertToProtoErrorList()}, nil
	}

	snapEntry, cerr := s.repo.GetEntryById(id, nil, el)
	if cerr != nil {
		// Already logged in GetEntryById (repository)
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
		cerr := cerror.NewCustomError(cerror.MissingField, "name is required")
		logrus.Error(cerr)
		el.AddCustomError(cerr)
		return &proto.GetEntryResponse{Errors: el.ConvertToProtoErrorList()}, nil
	}

	snapEntry, cerr := s.repo.GetEntryByName(*req.Name, nil, el)
	if cerr != nil {
		// Already logged in GetEntryByName (repository)
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
			id, err := uuid.Parse(revision.Id)
			if err != nil {
				cerr := cerror.NewCustomError(cerror.InvalidField, fmt.Sprintf("invalid UUID format: %s", revision.Id))
				logrus.Error(cerr)
				el.AddCustomError(cerr)
				continue
			}
			rev, cerr := s.repo.GetRevisionById(id, el)
			if cerr != nil {
				// Already logged in GetRevisionById (repository)
				continue
			}

			entry, cerr := s.repo.GetEntryById(rev.SnapEntryID, nil, el)
			if cerr != nil {
				// Already logged in GetEntryById (repository)
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
				EntryId:                rev.SnapEntryID.String(),
				TrackId:                rev.SnapTrackID.String(),
				ChannelId:              rev.SnapChannelID.String(),
				SnapName:               entry.Name,
			})

			// If id is not provided, check if snapName and sequence are provided
		} else if revision.SnapName != "" && revision.Sequence != 0 {
			rev, cerr := s.repo.GetRevisionByNameAndSequence(revision.SnapName, uint(revision.Sequence), el)
			if cerr != nil {
				// Already logged in GetRevisionByNameAndSequence (repository)
				continue
			}

			entry, cerr := s.repo.GetEntryById(rev.SnapEntryID, nil, el)
			if cerr != nil {
				// Already logged in GetEntryById (repository)
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
				EntryId:                rev.SnapEntryID.String(),
				TrackId:                rev.SnapTrackID.String(),
				ChannelId:              rev.SnapChannelID.String(),
				SnapName:               entry.Name,
			})

		} else {
			if revision.Id == "" && (revision.SnapName == "" || revision.Sequence == 0) {
				cerr := cerror.NewCustomError(cerror.MissingField, "id or snap name and sequence are required")
				logrus.Error(cerr)
				el.AddCustomError(cerr)
			}
			if revision.SnapName == "" && revision.Id == "" {
				cerr := cerror.NewCustomError(cerror.MissingField, "snap name is required")
				logrus.Error(cerr)
				el.AddCustomError(cerr)
			}
			if revision.Sequence == 0 && revision.Id == "" {
				cerr := cerror.NewCustomError(cerror.MissingField, "sequence is required")
				logrus.Error(cerr)
				el.AddCustomError(cerr)
			}
		}
	}
	return &proto.GetRevisionsResponse{Revisions: foundRevisions, Errors: el.ConvertToProtoErrorList()}, nil
}

// GetRevisionByNameAndSequence returns a single revision by snap name and sequence number
func (s *StoreLogic) GetRevisionByNameAndSequence(ctx context.Context, req *proto.GetRevisionRequest) (*proto.GetRevisionResponse, *cerror.CustomError) {
	el := cerror.NewErrorList()

	if req.SnapName == "" {
		cerr := cerror.NewCustomError(cerror.MissingField, "snap name is required")
		logrus.Error(cerr)
		el.AddCustomError(cerr)
		return &proto.GetRevisionResponse{Errors: el.ConvertToProtoErrorList()}, cerr
	}

	if req.Sequence == 0 {
		cerr := cerror.NewCustomError(cerror.MissingField, "sequence is required")
		logrus.Error(cerr)
		el.AddCustomError(cerr)
		return &proto.GetRevisionResponse{Errors: el.ConvertToProtoErrorList()}, cerr
	}

	entry, cerr := s.repo.GetEntryByName(req.SnapName, nil, el)
	if cerr != nil {
		// Already logged in GetEntryByName (repository)
		return &proto.GetRevisionResponse{Errors: el.ConvertToProtoErrorList()}, cerr
	}

	rev, cerr := s.repo.GetRevisionByNameAndSequence(entry.Name, uint(req.Sequence), el)
	if cerr != nil {
		// Already logged in GetRevisionByNameAndSequence (repository)
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
		EntryId:                rev.SnapEntryID.String(),
		TrackId:                rev.SnapTrackID.String(),
		ChannelId:              rev.SnapChannelID.String(),
		SnapName:               entry.Name,
	}, nil
}

func (s *StoreLogic) GetEntriesByAccountId(ctx context.Context, req *proto.GetEntriesByAccountIdRequest) (*proto.GetEntriesResponse, error) {
	el := cerror.NewErrorList()
	if req.AccountId == "" {
		cerr := cerror.NewCustomError(cerror.MissingField, "account id is required")
		logrus.Error(cerr)
		el.AddCustomError(cerr)
		return &proto.GetEntriesResponse{Errors: el.ConvertToProtoErrorList()}, nil
	}
	accId, err := uuid.Parse(req.AccountId)
	if err != nil {
		cerr := cerror.NewCustomError(cerror.InvalidField, fmt.Sprintf("invalid UUID format: %s", req.AccountId))
		logrus.Error(cerr)
		el.AddCustomError(cerr)
		return &proto.GetEntriesResponse{Errors: el.ConvertToProtoErrorList()}, nil
	}

	entries, cerr := s.repo.GetEntriesByAccountId(accId, nil, el)
	if cerr != nil {
		// Already logged in GetEntriesByAccountId (repository)
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
			cerr := cerror.NewCustomError(cerror.MissingField, "entry id is required")
			logrus.Error(cerr)
			el.AddCustomError(cerr)
			continue
		}

		entryId, err := uuid.Parse(entryIdReq.EntryId)
		if err != nil {
			cerr := cerror.NewCustomError(cerror.InvalidField, fmt.Sprintf("invalid UUID format: %s", entryIdReq.EntryId))
			logrus.Error(cerr)
			el.AddCustomError(cerr)
			continue
		}
		entry, cerr := s.repo.GetEntryById(entryId, nil, el)
		if cerr != nil {
			// Already logged in GetEntryById (repository)
			responses = append(responses, &proto.GetRevisionsByEntryIdResponse{
				EntryId: entryId.String(),
				Errors:  el.ConvertToProtoErrorList(),
			})
			continue
		}
		revisions, cerr := s.repo.GetRevisionsByEntryId(entryId, el)
		if cerr != nil {
			// Already logged in GetRevisionsByEntryId (repository)
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
		cerr := cerror.NewCustomError(cerror.MissingField, "snap name is required")
		logrus.Error(cerr)
		el.AddCustomError(cerr)
		return &proto.GetRevisionResponse{Errors: el.ConvertToProtoErrorList()}, nil
	}

	revision, cerr := s.repo.GetLatestRevisionByTrackAndChannel(req.SnapName, req.Track, req.Channel, el)
	if cerr != nil {
		// Already logged in GetLatestRevisionByTrackAndChannel (repository)
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
		EntryId:                revision.SnapEntryID.String(),
		TrackId:                revision.SnapTrackID.String(),
		ChannelId:              revision.SnapChannelID.String(),
		SnapName:               req.SnapName,
	}, nil
}

func (s *StoreLogic) SnapDownload(req *proto.SnapDownloadRequest, stream proto.StoreService_SnapDownloadServer) error {
	el := cerror.NewErrorList()
	if req.RevisionId == "" {
		cerr := cerror.NewCustomError(cerror.MissingField, "revision id is required")
		logrus.Error(cerr)
		el.AddCustomError(cerr)
		return stream.Send(&proto.SnapDownloadResponse{
			Errors: el.ConvertToProtoErrorList(),
		})
	}

	parsedRevisionId, err := uuid.Parse(req.RevisionId)
	if err != nil {
		cerr := cerror.NewCustomError(cerror.InvalidField, fmt.Sprintf("invalid UUID format: %s", req.RevisionId))
		logrus.Error(cerr)
		el.AddCustomError(cerr)
		err := stream.Send(&proto.SnapDownloadResponse{
			Errors: el.ConvertToProtoErrorList(),
		})
		if err != nil {
			logrus.Error("failed to send error response: ", err)
			return err
		}
		return err
	}
	revision, cerr := s.repo.GetRevisionById(parsedRevisionId, el)
	if cerr != nil {
		// Already logged in GetRevisionById (repository)
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
		// Already logged in retrieveObjectStoreFilePath (repository)
		err := stream.Send(&proto.SnapDownloadResponse{
			Errors: el.ConvertToProtoErrorList(),
		})
		if err != nil {
			logrus.Error("failed to send error response: ", err)
			return err
		}
		return fmt.Errorf("failed to retrieve snap from objectstore with filePath: %v", filePath)
	}

	snapFileReader, err := s.obs.GetSnapFileReader(filePath)
	if err != nil {
		cerr := cerror.NewCustomError(cerror.InternalServerError, "failed to get snap file reader")
		logrus.Error(cerr)
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

func saveFileToTemp(snapFile io.Reader, el *cerror.ErrorList) (string, *cerror.CustomError) {
	// Generate random file name for the new uploaded file so it doesn't override an old file with same name
	snapFileId := uuid.New().String()
	newFileName := snapFileId + ".snap"

	out, err := os.Create(path.Join("/tmp", newFileName))
	if err != nil {
		cerr := cerror.NewCustomError(cerror.InternalServerError, "Failed to create file")
		logrus.Error(cerr)
		el.AddCustomError(cerr)
		return "", cerr
	}
	defer func(out *os.File) {
		err := out.Close()
		if err != nil {
			logrus.Error("failed to close file: ", err)
		}

	}(out)

	_, err = io.Copy(out, snapFile)
	if err != nil {
		cerr := cerror.NewCustomError(cerror.InternalServerError, "Failed to copy file")
		logrus.Error(cerr)
		el.AddCustomError(cerr)
		return "", cerr
	}

	return newFileName, nil
}

func (s *StoreLogic) AddUpload(ctx context.Context, req *proto.AddUploadRequest) (*proto.AddUploadResponse, error) {
	el := cerror.NewErrorList()

	// Check on empty fields
	if req.SnapName == "" {
		cerr := cerror.NewCustomError(cerror.MissingField, "snap name is required")
		logrus.Error(cerr)
		el.AddCustomError(cerr)
	}
	if req.EntryId == "" {
		cerr := cerror.NewCustomError(cerror.MissingField, "entry id is required")
		logrus.Error(cerr)
		el.AddCustomError(cerr)
	}
	if req.Status == "" {
		cerr := cerror.NewCustomError(cerror.MissingField, "status is required")
		logrus.Error(cerr)
		el.AddCustomError(cerr)
	}
	if req.AccountId == "" {
		cerr := cerror.NewCustomError(cerror.MissingField, "account id is required")
		logrus.Error(cerr)
		el.AddCustomError(cerr)
	}
	if req.UnscannedFileName == "" {
		cerr := cerror.NewCustomError(cerror.MissingField, "unscanned file name is required")
		logrus.Error(cerr)
		el.AddCustomError(cerr)
	}
	if len(*el) > 0 {
		return &proto.AddUploadResponse{Errors: el.ConvertToProtoErrorList()}, nil
	}

	// Parse UUIDs
	entryId, err := uuid.Parse(req.EntryId)
	if err != nil {
		cerr := cerror.NewCustomError(cerror.InvalidField, err.Error())
		logrus.Error(cerr)
		el.AddCustomError(cerr)
		return &proto.AddUploadResponse{Errors: el.ConvertToProtoErrorList()}, nil
	}
	accountId, err := uuid.Parse(req.AccountId)
	if err != nil {
		cerr := cerror.NewCustomError(cerror.InvalidField, err.Error())
		logrus.Error(cerr)
		el.AddCustomError(cerr)
		return &proto.AddUploadResponse{Errors: el.ConvertToProtoErrorList()}, nil
	}

	// Add upload to the database
	snapUpload, cerr := s.repo.AddUpload(req.SnapName, entryId, req.Status, accountId, req.UnscannedFileName, el)
	if cerr != nil {
		// Already logged in AddUpload (repository)
		return &proto.AddUploadResponse{Errors: el.ConvertToProtoErrorList()}, nil
	}

	return &proto.AddUploadResponse{
		Id:       snapUpload.ID.String(),
		SnapName: snapUpload.SnapName,
		Status:   snapUpload.Status,
	}, nil
}

// LOGIC: UnscannedUpload receives a stream of data chunks from the client and saves them to a temporary file.
// After receiving all chunks, it saves the file to an object store bucket named "unscanned".
func (s *StoreLogic) UnscannedUpload(stream proto.StoreService_UnscannedUploadServer) error {
	el := cerror.NewErrorList()
	var snapFileBuffer bytes.Buffer
	sha := sha3.New384()

	for {
		req, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			cerr := cerror.NewCustomError(cerror.InternalServerError, fmt.Sprintf("failed to receive chunk: %v", err))
			logrus.Error(cerr)
			el.AddCustomError(cerr)
			return fmt.Errorf("failed to receive chunk: %v", err)
		}

		dataChunk := req.GetData()
		if dataChunk == nil {
			cerr := cerror.NewCustomError(cerror.InvalidField, "received nil data chunk")
			logrus.Error(cerr)
			el.AddCustomError(cerr)
			return fmt.Errorf("received empty data chunk")
		}

		// Write chunk to buffer
		_, err = snapFileBuffer.Write(dataChunk.Chunk)
		if err != nil {
			cerr := cerror.NewCustomError(cerror.InternalServerError, fmt.Sprintf("failed to write chunk to buffer: %v", err))
			logrus.Error(cerr)
			el.AddCustomError(cerr)
			return fmt.Errorf("failed to write chunk to buffer: %v", err)
		}

		// Update hash with the chunk
		_, err = sha.Write(dataChunk.Chunk)
		if err != nil {
			cerr := cerror.NewCustomError(cerror.InternalServerError, fmt.Sprintf("failed to update hash: %v", err))
			logrus.Error(cerr)
			el.AddCustomError(cerr)
			return fmt.Errorf("failed to update hash: %v", err)
		}
	}

	snapFileName, cerr := saveFileToTemp(&snapFileBuffer, el)
	if cerr != nil {
		// Already logged in saveFileToTemp (repository)
		return fmt.Errorf("failed to save file to temp storage: %v", cerr)
	}

	// Calculate sha3_384 hash of the file
	digest := sha.Sum(nil)                                                 // returns [48]byte
	sha3_384HashEncoded := base64.RawURLEncoding.EncodeToString(digest[:]) // just raw hash

	tmpPath := path.Join(os.TempDir(), snapFileName)

	metadata, err := s.obs.SaveFileToBucket("unscanned", tmpPath, sha3_384HashEncoded)
	if err != nil {
		cerr := cerror.NewCustomError(cerror.InternalServerError, fmt.Sprintf("failed to save file to object store: %v", err))
		logrus.Error(cerr)
		el.AddCustomError(cerr)
		return fmt.Errorf("failed to save file to object store: %v", cerr)
	}

	err = stream.SendAndClose(&proto.UnscannedUploadCompleteResponse{
		TempFileName: metadata.Key,
		Size:         uint64(metadata.Size),
		Errors:       el.ConvertToProtoErrorList(),
	})
	if err != nil {
		cerr := cerror.NewCustomError(cerror.InternalServerError, fmt.Sprintf("failed to send response: %v", err))
		logrus.Error(cerr)
		el.AddCustomError(cerr)
		return fmt.Errorf("failed to send response: %v", cerr)
	}

	return nil
}

func (s *StoreLogic) GetUploadStatus(ctx context.Context, req *proto.GetUploadStatusRequest) (*proto.GetUploadStatusResponse, error) {
	el := cerror.NewErrorList()
	if req.GetUploadId() == "" {
		cerr := cerror.NewCustomError(cerror.MissingField, "id is required")
		logrus.Error(cerr)
		el.AddCustomError(cerr)
		return &proto.GetUploadStatusResponse{Processed: true, Errors: el.ConvertToProtoErrorList()}, nil
	}

	id, err := uuid.Parse(req.GetUploadId())
	if err != nil {
		cerr := cerror.NewCustomError(cerror.InvalidField, fmt.Sprintf("invalid UUID format: %s", req.GetUploadId()))
		logrus.Error(cerr)
		el.AddCustomError(cerr)
		return &proto.GetUploadStatusResponse{Processed: true, Errors: el.ConvertToProtoErrorList()}, nil
	}

	snapUpload, cerr := s.repo.GetUploadById(id, el)
	if cerr != nil {
		// Already logged in GetUploadById (repository)
		return &proto.GetUploadStatusResponse{Processed: true, Errors: el.ConvertToProtoErrorList()}, nil
	}

	processed := false
	if snapUpload.Status == "processed" {
		processed = true
	}

	resp := &proto.GetUploadStatusResponse{
		UploadId:  snapUpload.ID.String(),
		Processed: processed,
		Revision:  snapUpload.Revision,
		Errors:    el.ConvertToProtoErrorList(),
	}

	return resp, nil
}

func (s *StoreLogic) UpdateUploadStatus(ctx context.Context, req *proto.UpdateUploadStatusRequest) (*proto.UpdateUploadStatusResponse, error) {
	el := cerror.NewErrorList()
	if req.GetUploadId() == "" {
		el.Add(cerror.MissingField, "id is required")
		return &proto.UpdateUploadStatusResponse{Errors: el.ConvertToProtoErrorList()}, nil
	}

	id, err := uuid.Parse(req.GetUploadId())
	if err != nil {
		logrus.Error(err)
		el.Add(cerror.InvalidField, fmt.Sprintf("invalid UUID format: %s", req.GetUploadId()))
		return &proto.UpdateUploadStatusResponse{Errors: el.ConvertToProtoErrorList()}, nil
	}

	if req.Errors != nil {
		for _, err := range req.Errors {
			el.Add(err.GetCode(), err.GetMessage())
		}
	}

	cerr := s.repo.UpdateUploadStatus(id, req.Status, req.Revision, el)
	if cerr != nil {
		// Already logged in UpdateUploadStatus (repository)
		return &proto.UpdateUploadStatusResponse{Errors: el.ConvertToProtoErrorList()}, nil
	}

	return &proto.UpdateUploadStatusResponse{
		Status: req.Status,
		Errors: el.ConvertToProtoErrorList(),
	}, nil
}

func (s *StoreLogic) AddRevision(ctx context.Context, req *proto.AddRevisionRequest) (*proto.AddRevisionResponse, error) {
	el := cerror.NewErrorList()

	if req.SnapName == "" {
		cerr := cerror.NewCustomError(cerror.MissingField, "snap name is required")
		logrus.Error(cerr)
		el.AddCustomError(cerr)
		return &proto.AddRevisionResponse{Errors: el.ConvertToProtoErrorList()}, nil
	}

	// Get entry by snapName
	entry, cerr := s.repo.GetEntryByName(req.SnapName, nil, el)
	if cerr != nil {
		// Already logged in GetEntryByName (repository)
		return &proto.AddRevisionResponse{Errors: el.ConvertToProtoErrorList()}, nil
	}

	// Check if a revision with the same SHA3_384 already exists -> this means that the uploaded file is the same as one already in the database
	existingRevision, cerr := s.repo.GetRevisionBySHA(req.Sha3_384, false, el)
	if cerr != nil && cerr.GetCode() != cerror.ResourceNotFound {
		// Already logged in GetRevisionBySHA (repository))
		return &proto.AddRevisionResponse{Errors: el.ConvertToProtoErrorList()}, nil
	}
	if existingRevision != nil {
		el.Add(cerror.AlreadyRegistered, fmt.Sprintf("revision with the same SHA3_384 already exists for %s", req.SnapName))
		// If revision already exists, we don't need to move the file to the `snaps` bucket and we can delete the file from the `unscanned` bucket
		cerr := s.obs.DeleteFileFromBucket("unscanned", req.UnscannedFileName)
		if cerr != nil {
			// Already logged in DeleteFileFromBucket (object store)
			el.AddCustomError(cerr)
		}
		return &proto.AddRevisionResponse{Errors: el.ConvertToProtoErrorList()}, nil
	}

	// Get the sequence number of the last revision (if there is any)
	lastRevision, cerr := s.repo.GetLatestRevisionByEntryId(entry.ID, el)
	if cerr != nil && cerr.GetCode() != cerror.ResourceNotFound {
		// Already logged in GetLatestRevisionByEntryId (repository)
		return &proto.AddRevisionResponse{Errors: el.ConvertToProtoErrorList()}, nil
	}
	sequenceNumber := uint(1) // Initialize to 1
	// If there is a last revision, increment the sequence number
	if lastRevision != nil && lastRevision.SequenceNumber != nil {
		sequenceNumber = *lastRevision.SequenceNumber + 1
	}

	// Get the file path in the object store
	minioFilePath := s.createObjectStoreFilePath(entry.Name, sequenceNumber)

	// Parse the tracks and channels
	tracksAndChannels := parseTracksAndChannels(req.TracksAndChannels)

	// Create the revisions for the tracks and channels
	for track, channels := range tracksAndChannels {
		for _, channel := range channels {
			// Get the track to which the revision will be associated
			trackResp, cerr := s.repo.GetTrackByEntryIdAndName(entry.ID, track, el)
			if cerr != nil {
				// Already logged in GetTracksByEntryIdAndName (repository)
				return &proto.AddRevisionResponse{Errors: el.ConvertToProtoErrorList()}, nil
			}

			if trackResp == nil {
				// If the track is not found, create a new one
				trackResp, cerr = s.repo.AddTrack(entry.ID, track, el)
				if cerr != nil {
					// Already logged in AddTrack (repository)
					return &proto.AddRevisionResponse{Errors: el.ConvertToProtoErrorList()}, nil
				}
				// Add default channels to the new track
				cerr = s.repo.AddDefaultChannels(trackResp.SnapEntryID, trackResp.ID, el)
				if cerr != nil {
					// Already logged in AddDefaultChannels (repository)
					return &proto.AddRevisionResponse{Errors: el.ConvertToProtoErrorList()}, nil
				}
			}

			// Get the channel to which the revision will be associated
			channelResp, cerr := s.repo.GetChannelByTrackIdAndName(trackResp.ID, channel, el)
			if cerr != nil && cerr.GetCode() != cerror.ResourceNotFound {
				// Already logged in GetChannelByTrackIdAndName (repository)
				return &proto.AddRevisionResponse{Errors: el.ConvertToProtoErrorList()}, nil
			}
			// If the channel is not found, create a new one
			if channelResp == nil {
				channelResp, cerr = s.repo.AddChannel(entry.ID, trackResp.ID, channel, el)
				if cerr != nil {
					// Already logged in AddChannel (repository)
					return &proto.AddRevisionResponse{Errors: el.ConvertToProtoErrorList()}, nil
				}
			}

			// Create the revision
			_, cerr = s.repo.AddRevision(entry.ID, trackResp.ID, channelResp.ID, req.SnapName, req.Size, sequenceNumber, req.Architectures, req.Sha3_384, minioFilePath, el)
			if cerr != nil {
				// Already logged in AddRevision (repository)
				return &proto.AddRevisionResponse{Errors: el.ConvertToProtoErrorList()}, nil
			}
		}
	}

	// Move the snap file to the `snap` bucket
	err := s.obs.Move("unscanned", "snaps", req.UnscannedFileName, minioFilePath)
	if err != nil {
		cerr := cerror.NewCustomError(cerror.InternalServerError, fmt.Sprintf("failed to move snap file from `unscanned` to `snaps` bucket: %+v", err))
		logrus.Error(cerr)
		el.AddCustomError(cerr)
		return &proto.AddRevisionResponse{Errors: el.ConvertToProtoErrorList()}, nil
	}

	return &proto.AddRevisionResponse{
		SnapName: req.SnapName,
		Status:   "success",
		Revision: uint64(sequenceNumber),
	}, nil
}

func (s *StoreLogic) GetObjectCustomMetadata(ctx context.Context, req *proto.GetObjectCustomMetadataRequest) (*proto.GetObjectCustomMetadataResponse, error) {
	el := cerror.NewErrorList()
	if req.Bucket == "" {
		cerr := cerror.NewCustomError(cerror.MissingField, "bucket is required")
		logrus.Error(cerr)
		el.AddCustomError(cerr)
		return &proto.GetObjectCustomMetadataResponse{Errors: el.ConvertToProtoErrorList()}, nil
	}
	if req.ObjectKey == "" {
		cerr := cerror.NewCustomError(cerror.MissingField, "object key is required")
		logrus.Error(cerr)
		el.AddCustomError(cerr)
		return &proto.GetObjectCustomMetadataResponse{Errors: el.ConvertToProtoErrorList()}, nil
	}
	metadata, err := s.obs.GetObjectCustomMetadata(req.Bucket, req.ObjectKey)
	if err != nil {
		cerr := cerror.NewCustomError(cerror.InternalServerError, fmt.Sprintf("failed to get object custom metadata: %v", err))
		logrus.Error(cerr)
		el.AddCustomError(cerr)
		return &proto.GetObjectCustomMetadataResponse{Errors: el.ConvertToProtoErrorList()}, nil
	}

	return &proto.GetObjectCustomMetadataResponse{
		Sha3_384: *metadata.Sha3_384,
	}, nil
}

// ################# HELPERS #################

func (s *StoreLogic) retrieveObjectStoreFilePath(revision *models.SnapRevision, el *cerror.ErrorList) (string, *cerror.CustomError) {
	entry, cerr := s.repo.GetEntryById(revision.SnapEntryID, nil, el)
	if cerr != nil {
		// Already logged in GetEntryById (repository)
		return "", cerr
	}
	track, cerr := s.repo.GetTrackById(revision.SnapTrackID, el)
	if cerr != nil {
		// Already logged in GetTrackById (repository)
		return "", cerr
	}
	channel, cerr := s.repo.GetChannelById(revision.SnapChannelID, el)
	if cerr != nil {
		// Already logged in GetChannelById (repository)
		return "", cerr
	}
	return fmt.Sprintf("%s/%s/%s/%s_%d.snap", entry.Name, track.Name, channel.Name, entry.Name, uintPointerToUint(revision.SequenceNumber)), nil
}

func (s *StoreLogic) createObjectStoreFilePath(entryName string, sequenceNumber uint) string {
	return fmt.Sprintf("%s/%s_%d.snap", entryName, entryName, sequenceNumber)
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
	}
}

func parseTracksAndChannels(tracksAndChannels []string) map[string][]string {
	// Initialize a map to hold the parsed tracks and channels.
	parsed := make(map[string][]string)

	// If no tracks and channels are provided, default to "latest/stable".
	if len(tracksAndChannels) == 0 {
		parsed["latest"] = []string{"stable"}
		return parsed
	}

	// Iterate over the provided tracks and channels.
	for _, tc := range tracksAndChannels {
		var track, channel string

		// Split the input into track and channel.
		parts := strings.Split(tc, "/")
		if len(parts) == 2 {
			track, channel = parts[0], parts[1]
		} else if len(parts) == 1 {
			track, channel = "latest", parts[0]
		} else {
			logrus.Warnf("Invalid format for track/channel: %s", tc)
			continue
		}

		// Add the channel to the list for the track.
		parsed[track] = append(parsed[track], channel)
	}

	return parsed
}
