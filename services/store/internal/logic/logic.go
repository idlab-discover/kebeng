package logic

import (
	"bytes"
	"context"
	"io"
	"os"
	"path"

	"fmt"

	"github.com/google/uuid"
	cerror "github.com/idlab-discover/kebeng/common/cerror"
	"github.com/idlab-discover/kebeng/services/store/internal/config"
	"github.com/idlab-discover/kebeng/services/store/internal/objectstore"
	"github.com/idlab-discover/kebeng/services/store/internal/repository"
	proto "github.com/idlab-discover/kebeng/services/store/proto"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type StoreLogic struct {
	config *config.Config
	proto.UnimplementedStoreServiceServer
	repo repository.ISnapsRepository
}

func NewStoreLogic(repo repository.ISnapsRepository, config *config.Config) *StoreLogic {
	return &StoreLogic{repo: repo, config: config}
}

func (s *StoreLogic) RegisterSnapName(ctx context.Context, req *proto.RegisterSnapNameRequest) (*proto.RegisterSnapNameResponse, error) {
	el := make([]*proto.Error, 0)

	if req.SnapName == "" {
		el = append(el, &proto.Error{Code: cerror.MissingField, Message: "snap_name is required"})
		return &proto.RegisterSnapNameResponse{Errors: el}, nil
	}

	// TODO: check if snap name is valid (it should only have ASCII lowercase letters, numbers, and hyphens, and must have at least one letter)

	// First check if the snap name is already registered
	snapEntry, err := s.repo.GetEntryByName(req.SnapName, nil)
	if err != nil && err.GetCode() != cerror.ResourceNotFound {
		el = append(el, &proto.Error{Code: err.GetCode(), Message: err.GetMessage()})
		return &proto.RegisterSnapNameResponse{Errors: el}, nil
	}

	// TODO: if dryRun is true, but snap name is not registered, what should we return? -> right now we return an empty string

	// If dryRun is true, we only check if the snap name is already registered
	if req.DryRun {
		if snapEntry != nil {
			return &proto.RegisterSnapNameResponse{SnapName: req.SnapName}, nil // Id will be set to empty string, docs say it should be null, but nil can't be assigned to string -> TODO: see later if this is a problem
		} else {
			return &proto.RegisterSnapNameResponse{SnapName: ""}, nil // Return an empty string if the snap name is not registered
		}
	}
	// If dryRun is false, but snap name is already registered, return an error
	if snapEntry != nil {
		el = append(el, &proto.Error{Code: cerror.AlreadyRegistered, Message: "snap name '" + req.SnapName + "' is already registered."})
		return &proto.RegisterSnapNameResponse{Errors: el}, nil
	}

	// If there is no snap with the same name and dry_run == false, register the snap name
	accountId, err1 := uuid.Parse(req.AccountId)
	if err1 != nil {
		el = append(el, &proto.Error{Code: cerror.InvalidField, Message: "invalid UUID format for AccountId"})
		return &proto.RegisterSnapNameResponse{Errors: el}, nil
	}
	snapEntry, err = s.repo.RegisterSnap(req.SnapName, req.IsPrivate, req.Store, accountId)
	if err != nil {
		el = append(el, &proto.Error{Code: err.GetCode(), Message: err.GetMessage()})
		return &proto.RegisterSnapNameResponse{Errors: el}, nil
	}

	// Add "latest" track to the snap entry
	snapTrack, err := s.repo.AddTrack(snapEntry.ID, "latest")
	if err != nil {
		el = append(el, &proto.Error{Code: err.GetCode(), Message: err.GetMessage()})
		return &proto.RegisterSnapNameResponse{Errors: el}, nil
	}

	// Add default channels for the "latest" track to the snap entry
	err = s.repo.AddDefaultChannels(snapEntry.ID, snapTrack.ID)
	if err != nil {
		el = append(el, &proto.Error{Code: err.GetCode(), Message: err.GetMessage()})
		return &proto.RegisterSnapNameResponse{Errors: el}, nil
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
	el := make([]*proto.Error, 0)
	foundEntries := make([]*proto.GetEntryResponse, 0)

	for _, entry := range req.Entries {
		// First try to retrieve the entry by its ID
		if entry.Id != nil && *entry.Id != "" {
			id, err := uuid.Parse(*entry.Id)
			if err != nil {
				logrus.Errorf("Failed to parse UUID '%s':", *entry.Id)
				el = append(el, &proto.Error{Code: cerror.InvalidField, Message: fmt.Sprintf("Invalid UUID format for id '%s'", *entry.Id)})
				continue
			}

			snapEntry, err2 := s.repo.GetEntryById(id, nil)
			if err2 != nil {
				// Already logged in GetEntryById (repository
				el = append(el, &proto.Error{Code: err2.GetCode(), Message: err2.GetMessage()})
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
			snapEntry, err := s.repo.GetEntryByName(*entry.Name, nil)
			if err != nil {
				// Already logged in GetEntryByName (repository)
				el = append(el, &proto.Error{Code: err.GetCode(), Message: err.GetMessage()})
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
			el = append(el, &proto.Error{
				Code:    cerror.MissingField,
				Message: "Id or name is required"})
		}
	}
	return &proto.GetEntriesResponse{Entries: foundEntries, Errors: el}, nil
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
	el := make([]*proto.Error, 0)
	if req.Id == nil || *req.Id == "" {
		el = append(el, &proto.Error{Code: cerror.MissingField, Message: "Id is required"})
		return &proto.GetEntryResponse{Errors: el}, nil
	}

	id, err := uuid.Parse(*req.Id)
	if err != nil {
		logrus.Error(err)
		el = append(el, &proto.Error{Code: cerror.InvalidField, Message: "Invalid UUID format"})
		return &proto.GetEntryResponse{Errors: el}, nil
	}

	snapEntry, err2 := s.repo.GetEntryById(id, nil)
	if err2 != nil {
		// Already logged in GetEntryById (repository)
		el = append(el, &proto.Error{Code: err2.GetCode(), Message: err2.GetMessage()})
		return &proto.GetEntryResponse{Errors: el}, nil
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
	el := make([]*proto.Error, 0)
	if req.Name == nil || *req.Name == "" {
		el = append(el, &proto.Error{Code: cerror.MissingField, Message: "Name is required"})
		return &proto.GetEntryResponse{Errors: el}, nil
	}

	snapEntry, err := s.repo.GetEntryByName(*req.Name, nil)
	if err != nil {
		el = append(el, &proto.Error{Code: err.GetCode(), Message: err.GetMessage()})
		return &proto.GetEntryResponse{Errors: el}, nil
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
	el := make([]*proto.Error, 0)
	foundRevisions := make([]*proto.GetRevisionResponse, 0)

	// Each GetRevisionRequest will contain either id or snapName and sequence
	for _, revision := range req.Revisions {
		// First check if id is provided
		if revision.Id != "" {
			id, err1 := uuid.Parse(revision.Id)
			if err1 != nil {
				logrus.Errorf("Failed to parse UUID '%s':", revision.Id)
				el = append(el, &proto.Error{Code: cerror.InvalidField, Message: fmt.Sprintf("Invalid UUID format for id '%s'", revision.Id)})
				continue
			}
			rev, err := s.repo.GetRevisionById(id)
			if err != nil {
				// Already logged in GetRevisionById (repository)
				el = append(el, &proto.Error{Code: err.GetCode(), Message: err.GetMessage()})
				continue
			}

			entry, err := s.repo.GetEntryById(rev.SnapEntryID, nil)
			if err != nil {
				// Already logged in GetEntryById (repository)
				el = append(el, &proto.Error{Code: err.GetCode(), Message: err.GetMessage()})
				continue
			}

			foundRevisions = append(foundRevisions, &proto.GetRevisionResponse{
				Id:            rev.ID.String(),
				SnapName:      entry.Name,
				Sequence:      uint64(*rev.SequenceNumber),
				Architectures: rev.Architectures,
				Version:       *rev.Version,
				Status:        *rev.Status,
			})

			// If id is not provided, check if snapName and sequence are provided
		} else if revision.SnapName != "" && revision.Sequence != 0 {
			rev, err := s.repo.GetRevisionByNameAndSequence(revision.SnapName, uint(revision.Sequence))
			if err != nil {
				el = append(el, &proto.Error{Code: err.GetCode(), Message: err.GetMessage()})
				continue
			}

			foundRevisions = append(foundRevisions, &proto.GetRevisionResponse{
				Id:            rev.ID.String(),
				SnapName:      revision.SnapName,
				Sequence:      uint64(*rev.SequenceNumber),
				Architectures: rev.Architectures,
				Version:       *rev.Version,
				Status:        *rev.Status,
			})

		} else {
			if revision.Id == "" && (revision.SnapName == "" || revision.Sequence == 0) {
				el = append(el, &proto.Error{Code: cerror.MissingField, Message: "Id is required"})
			}
			if revision.SnapName == "" && revision.Id == "" {
				el = append(el, &proto.Error{Code: cerror.MissingField, Message: "Snap name is required"})
			}
			if revision.Sequence == 0 && revision.Id == "" {
				el = append(el, &proto.Error{Code: cerror.MissingField, Message: "Sequence is required"})
			}
		}
	}
	return &proto.GetRevisionsResponse{Revisions: foundRevisions, Errors: el}, nil
}

// GetRevisionByNameAndSequence returns a single revision by snap name and sequence number
func (s *StoreLogic) GetRevisionByNameAndSequence(ctx context.Context, req *proto.GetRevisionRequest) (*proto.GetRevisionResponse, *cerror.CustomError) {
	el := make([]*proto.Error, 0)

	if req.SnapName == "" {
		el = append(el, &proto.Error{Code: cerror.MissingField, Message: "Snap name is required"})
		return &proto.GetRevisionResponse{Errors: el}, nil
	}

	if req.Sequence == 0 {
		el = append(el, &proto.Error{Code: cerror.MissingField, Message: "Sequence is required"})
		return &proto.GetRevisionResponse{Errors: el}, nil
	}

	snapEntry, err := s.repo.GetEntryByName(req.SnapName, nil)
	if err != nil {
		// Already logged in GetEntryByName (repository)
		el = append(el, &proto.Error{Code: err.GetCode(), Message: err.GetMessage()})
		return &proto.GetRevisionResponse{Errors: el}, err
	}

	revision, err := s.repo.GetRevisionByNameAndSequence(snapEntry.Name, uint(req.Sequence))
	if err != nil {
		// Already logged in GetRevisionByNameAndSequence (repository)
		el = append(el, &proto.Error{Code: err.GetCode(), Message: err.GetMessage()})
		return &proto.GetRevisionResponse{Errors: el}, err
	}

	return &proto.GetRevisionResponse{
		Id:            revision.ID.String(),
		SnapName:      snapEntry.Name,
		Sequence:      uint64(*revision.SequenceNumber),
		Architectures: revision.Architectures,
		Version:       *revision.Version,
		Status:        *revision.Status,
	}, nil
}

func (s *StoreLogic) GetEntriesByAccountId(ctx context.Context, req *proto.GetEntriesByAccountIdRequest) (*proto.GetEntriesResponse, error) {
	el := make([]*proto.Error, 0)
	if req.AccountId == "" {
		el = append(el, &proto.Error{Code: cerror.MissingField, Message: "Account id is required"})
		return &proto.GetEntriesResponse{Errors: el}, nil
	}
	accId, err := uuid.Parse(req.AccountId)
	if err != nil {
		logrus.Error(err)
		el = append(el, &proto.Error{Code: cerror.InvalidField, Message: "Invalid UUID format"})
		return &proto.GetEntriesResponse{Errors: el}, nil
	}

	entries, err2 := s.repo.GetEntriesByAccountId(accId, nil)
	if err2 != nil {
		el = append(el, &proto.Error{Code: err2.GetCode(), Message: err2.GetMessage()})
		return &proto.GetEntriesResponse{Errors: el}, nil
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
	el := make([]*proto.Error, 0)
	responses := make([]*proto.GetRevisionsByEntryIdResponse, 0)
	for _, entryIdReq := range req.GetRequests() {
		if entryIdReq.EntryId == "" {
			el = append(el, &proto.Error{
				Code:    cerror.MissingField,
				Message: "Entry id is required",
			})
			continue
		}

		entryId, err := uuid.Parse(entryIdReq.EntryId)
		if err != nil {
			logrus.Error(err)
			el = append(el, &proto.Error{
				Code:    cerror.InvalidField,
				Message: "Invalid UUID format",
			})
			continue
		}

		revisions, err2 := s.repo.GetRevisionsByEntryId(entryId)
		if err2 != nil {
			logrus.Debugf("Failed to get revisions from database: %v", err2)
			el = append(el, &proto.Error{
				Code:    err2.GetCode(),
				Message: err2.GetMessage(),
			})
			// add empty response to keep the order of responses
			responses = append(responses, &proto.GetRevisionsByEntryIdResponse{
				EntryId: entryId.String(),
				Errors:  el,
			})
			continue
		}
		// revision were found so convert them in response format
		revisionsProto := make([]*proto.GetRevisionResponse, len(revisions))
		for i, rev := range revisions {
			revisionsProto[i] = &proto.GetRevisionResponse{
				Id:                     rev.ID.String(),
				BuildAssertionFilename: *rev.BuildAssertionFileName,
				Sequence:               uint64(*rev.SequenceNumber),
				Architectures:          rev.Architectures,
				Version:                *rev.Version,
				Status:                 *rev.Status,
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

// func (s *StoreLogic) UploadSnap(ctx context.Context, req *proto.UploadSnapRequest) (*proto.UploadSnapResponse, error) {
// 	// TODO: check which fields are required and which are optional in req
// 	el := make([]*proto.Error, 0)
// 	snapFileName, id, err := saveFileToTemp(bytes.NewReader(req.File))
// 	if err != nil {
// 		logrus.Errorf("Failed to save file to temp storage: %v", err)
// 		el = append(el, &proto.Error{Code: cerror.InternalServerError, Message: "Failed to save file to temp storage"})
// 		return &proto.UploadSnapResponse{Errors: el}, nil
// 	}

// 	objectstore := objectstore.NewObjectStore()
// 	tmpPath := path.Join(os.TempDir(), snapFileName)

// 	size, err2 := objectstore.SaveFileToBucket("unscanned", tmpPath)
// 	if err2 != nil {
// 		logrus.Errorf("Failed to save file to object store: %v", err)
// 		el = append(el, &proto.Error{Code: cerror.InternalServerError, Message: "Failed to save file to object store"})
// 		return &proto.UploadSnapResponse{Errors: el}, nil
// 	}

// 	// addSnap() adds snap to entry table
// 	_, err3 := s.repo.AddSnap(snapFileName, size, uuid.New()) // uuid.New() is a placeholder for account id that is going to be added later throught the context
// 	if err3 != nil {
// 		logrus.Error(err2)
// 		el = append(el, &proto.Error{Code: cerror.InternalServerError, Message: "Failed to add snap to database"})
// 		return &proto.UploadSnapResponse{Errors: el}, nil
// 	}

// 	return &proto.UploadSnapResponse{Id: id, DisplayName: snapFileName}, nil
// }

func (s *StoreLogic) UnscannedUpload(ctx context.Context, req *proto.UnscannedUploadRequest) (*proto.UnscannedUploadResponse, error) {
	el := make([]*proto.Error, 0)
	snapFileName, err := saveFileToTemp(bytes.NewReader(req.SnapFile))
	if err != nil {
		logrus.Errorf("Failed to save file to temp storage: %v", err)
		el = append(el, &proto.Error{Code: cerror.InternalServerError, Message: "Failed to save file to temp storage"})
		return &proto.UnscannedUploadResponse{Errors: el}, nil
	}

	objectstore := objectstore.NewObjectStore()
	tmpPath := path.Join(os.TempDir(), snapFileName)

	size, err2 := objectstore.SaveFileToBucket("unscanned", tmpPath)
	if err2 != nil {
		logrus.Errorf("Failed to save file to object store: %v", err)
		el = append(el, &proto.Error{Code: cerror.InternalServerError, Message: "Failed to save file to object store"})
		return &proto.UnscannedUploadResponse{Errors: el}, nil
	}

	return &proto.UnscannedUploadResponse{TempFileName: tmpPath, FileSize: size}, nil
}

func saveFileToTemp(snapFile io.Reader) (string, *cerror.CustomError) {
	// Generate random file name for the new uploaded file so it doesn't override the old file with same name
	snapFileId := uuid.New().String()
	newFileName := snapFileId + ".snap"

	out, err := os.Create(path.Join("/tmp", newFileName))
	if err != nil {
		return "", cerror.NewCustomError(cerror.InternalServerError, "Failed to create file")
	}
	defer out.Close()

	_, err = io.Copy(out, snapFile)
	if err != nil {
		return "", cerror.NewCustomError(cerror.InternalServerError, "Failed to copy file")
	}

	return newFileName, nil
}
