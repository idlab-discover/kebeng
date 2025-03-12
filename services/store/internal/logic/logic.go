package logic

import (
	"bytes"
	"context"
	"io"
	"os"
	"path"

	"fmt"

	"github.com/google/uuid"
	"github.com/idlab-discover/kebeng/services/store/internal/errors"
	"github.com/idlab-discover/kebeng/services/store/internal/objectstore"
	"github.com/idlab-discover/kebeng/services/store/internal/repositories"
	proto "github.com/idlab-discover/kebeng/services/store/proto"
	"github.com/sirupsen/logrus"
)

type StoreLogic struct {
	proto.UnimplementedStoreServiceServer
	repo *repositories.SnapsRepository
}

func NewStoreLogic(repo *repositories.SnapsRepository) *StoreLogic {
	return &StoreLogic{repo: repo}
}

func (s *StoreLogic) UploadSnap(ctx context.Context, req *proto.UploadSnapRequest) (*proto.UploadSnapResponse, error) {
	// TODO: check which fields are required and which are optional in req
	errList := make([]*proto.Error, 0)
	snapFileName, id, err := saveFileToTemp(bytes.NewReader(req.File))
	if err != nil {
		logrus.Errorf("Failed to save file to temp storage: %v", err)
		errList = append(errList, &proto.Error{Code: errors.InternalServerError, Message: "Failed to save file to temp storage"})
		return &proto.UploadSnapResponse{Errors: errList}, err
	}

	objectstore := objectstore.NewObjectStore()
	tmpPath := path.Join(os.TempDir(), snapFileName)

	size, err := objectstore.SaveFileToBucket("unscanned", tmpPath)
	if err != nil {
		logrus.Errorf("Failed to save file to object store: %v", err)
		errList = append(errList, &proto.Error{Code: errors.InternalServerError, Message: "Failed to save file to object store"})
		return &proto.UploadSnapResponse{Errors: errList}, err
	}

	// addSnap() adds snap to snap_entries table
	_, err = s.repo.AddSnap(snapFileName, size, uuid.New()) // uuid.New() is a placeholder for account id that is going to be added later throught the context
	if err != nil {
		logrus.Error(err)
		errList = append(errList, &proto.Error{Code: errors.InternalServerError, Message: "Failed to add snap to database"})
		return &proto.UploadSnapResponse{Errors: errList}, err
	}

	return &proto.UploadSnapResponse{Id: id, DisplayName: snapFileName}, nil
}

func (s *StoreLogic) RegisterSnapName(ctx context.Context, req *proto.RegisterSnapNameRequest) (*proto.RegisterSnapNameResponse, error) {
	errList := make([]*proto.Error, 0)

	if req.SnapName == "" {
		errList = append(errList, &proto.Error{Code: errors.MissingField, Message: "Snap name is required"})
		return &proto.RegisterSnapNameResponse{Errors: errList}, nil
	}

	// First check if the snap name is already registered
	snapEntry, err := s.repo.GetEntryByName(req.SnapName, false)
	if err != nil {
		logrus.Error(err)
		errList = append(errList, &proto.Error{Code: errors.InternalServerError, Message: "Failed to get snap from database"})
		return &proto.RegisterSnapNameResponse{Errors: errList}, err
	}
	if snapEntry != nil { // if dryRun is true, we only check if the snap name is already registered -> snapEntry != nil
		errList = append(errList, &proto.Error{Code: errors.AlreadyRegistered, Message: "The snap name '" + req.SnapName + "' is already registered."})
		return &proto.RegisterSnapNameResponse{Errors: errList}, err
	}

	if req.DryRun {
		return &proto.RegisterSnapNameResponse{SnapName: req.SnapName}, nil // Id will be set to empty string, docs say it should be null, but nil can't be assigned to string -> see later if this is a problem
	}

	// If there is no snap with the same name and dry_run == false, register the snap name
	snapEntry, err = s.repo.RegisterSnap(req.SnapName, req.IsPrivate)
	if err != nil {
		logrus.Error(err)
		errList = append(errList, &proto.Error{Code: errors.InternalServerError, Message: "Failed to register snap name"})
		return &proto.RegisterSnapNameResponse{Errors: errList}, err
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
//   - *proto.GetEntriesResponse: The response containing the list of found entries and any errors encountered.
//   - error: An error if the operation fails.
func (s *StoreLogic) GetEntries(ctx context.Context, req *proto.GetEntriesRequest) (*proto.GetEntriesResponse, error) {
	errList := make([]*proto.Error, 0)
	foundEntries := make([]*proto.GetEntryResponse, 0)

	for _, entry := range req.Entries {
		if entry.Id != "" {
			id, err := uuid.Parse(entry.Id)
			if err != nil {
				logrus.Error(err)
				errList = append(errList, &proto.Error{Code: errors.InvalidField, Message: "Invalid UUID format"})
				continue
			}
			snapEntry, err := s.repo.GetEntryById(id, false)
			if err != nil {
				logrus.Error(err)
				errList = append(errList, &proto.Error{Code: errors.InternalServerError, Message: "Failed to get entry from database"})
				continue
			}
			if snapEntry != nil {
				foundEntries = append(foundEntries, &proto.GetEntryResponse{Id: snapEntry.ID.String(), SnapName: snapEntry.Name, Type: snapEntry.Type, Confinement: snapEntry.Confinement, Base: snapEntry.Base, Private: snapEntry.Private})
			} else {
				errList = append(errList, &proto.Error{Code: errors.ResourceNotFound, Message: "Entry with id '" + entry.Id + "' not found"})
			}
		} else if entry.Name != "" {
			snapEntry, err := s.repo.GetEntryByName(entry.Name, false)
			if err != nil {
				logrus.Error(err)
				errList = append(errList, &proto.Error{Code: errors.InternalServerError, Message: "Failed to get entry from database"})
				continue
			}
			if snapEntry != nil {
				foundEntries = append(foundEntries, &proto.GetEntryResponse{Id: snapEntry.ID.String(), SnapName: snapEntry.Name, Type: snapEntry.Type, Confinement: snapEntry.Confinement, Base: snapEntry.Base, Private: snapEntry.Private})
			} else {
				errList = append(errList, &proto.Error{Code: errors.ResourceNotFound, Message: "Entry with name '" + entry.Name + "' not found"})
			}
		} else {
			if entry.Id == "" && entry.Name == "" {
				errList = append(errList, &proto.Error{Code: errors.MissingField, Message: "Id or name is required"})
			}
		}
	}
	return &proto.GetEntriesResponse{Entries: foundEntries, Errors: errList}, nil
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
//   - *proto.GetEntryResponse: The response containing the found snap entry and any errors encountered.
//   - error: An error if the operation fails.
func (s *StoreLogic) GetEntryById(ctx context.Context, req *proto.GetEntryRequest) (*proto.GetEntryResponse, error) {
	errList := make([]*proto.Error, 0)
	if req.Id == "" {
		errList = append(errList, &proto.Error{Code: errors.MissingField, Message: "Id is required"})
		return &proto.GetEntryResponse{Errors: errList}, fmt.Errorf("id is required")
	}

	id, err := uuid.Parse(req.Id)
	if err != nil {
		logrus.Error(err)
		errList = append(errList, &proto.Error{Code: errors.InvalidField, Message: "Invalid UUID format"})
		return &proto.GetEntryResponse{Errors: errList}, nil
	}
	snapEntry, err := s.repo.GetEntryById(id, false)
	if err != nil {
		logrus.Error(err)
		errList = append(errList, &proto.Error{Code: errors.InternalServerError, Message: "Failed to get snap from database"})
		return &proto.GetEntryResponse{Errors: errList}, err
	}
	if snapEntry == nil {
		errList = append(errList, &proto.Error{Code: errors.ResourceNotFound, Message: "Snap with id '" + req.Id + "' not found"})
		return &proto.GetEntryResponse{Errors: errList}, nil
	}

	return &proto.GetEntryResponse{Id: snapEntry.ID.String(), SnapName: snapEntry.Name, Type: snapEntry.Type, Confinement: snapEntry.Confinement, Base: snapEntry.Base, Private: snapEntry.Private}, nil
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
//   - *proto.GetEntryResponse: The response containing the found snap entry and any errors encountered.
//   - error: An error if the operation fails.
func (s *StoreLogic) GetEntryByName(ctx context.Context, req *proto.GetEntryRequest) (*proto.GetEntryResponse, error) {
	errList := make([]*proto.Error, 0)
	if req.Name == "" {
		errList = append(errList, &proto.Error{Code: errors.MissingField, Message: "Name is required"})
		return &proto.GetEntryResponse{Errors: errList}, fmt.Errorf("name is required")
	}

	snapEntry, err := s.repo.GetEntryByName(req.Name, false)
	if err != nil {
		logrus.Error(err)
		errList = append(errList, &proto.Error{Code: errors.InternalServerError, Message: "Failed to get snap from database"})
		return &proto.GetEntryResponse{Errors: errList}, err
	}
	if snapEntry == nil {
		errList = append(errList, &proto.Error{Code: errors.ResourceNotFound, Message: "Snap with name '" + req.Name + "' not found"})
		return &proto.GetEntryResponse{Errors: errList}, nil
	}

	return &proto.GetEntryResponse{Id: snapEntry.ID.String(), SnapName: snapEntry.Name, Type: snapEntry.Type, Confinement: snapEntry.Confinement, Base: snapEntry.Base, Private: snapEntry.Private}, nil
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
//   - *proto.GetRevisionsResponse: The response containing the list of found revisions and any errors encountered.
//   - error: An error if the operation fails.
func (s *StoreLogic) GetRevisions(ctx context.Context, req *proto.GetRevisionsRequest) (*proto.GetRevisionsResponse, error) {
	errList := make([]*proto.Error, 0)
	foundRevisions := make([]*proto.GetRevisionResponse, 0)

	for _, revision := range req.Revisions {
		// Each GetRevisionRequest will contain either id or snapName and sequence
		// First check if id is provided
		if revision.Id != "" {
			rev, err := s.repo.GetRevisionById(revision.Id)
			if err != nil {
				logrus.Error(err)
				errList = append(errList, &proto.Error{Code: errors.InternalServerError, Message: "Failed to get revision from database"})
				continue
			}
			if rev != nil {
				// Extra check if snap entry exists -> need this to get snap name
				entry, err := s.repo.GetEntryById(rev.SnapEntryID, false)
				if err != nil {
					logrus.Error(err)
					errList = append(errList, &proto.Error{Code: errors.InternalServerError, Message: "Failed to get entry from database"})
					continue
				}
				if entry != nil {
					foundRevisions = append(foundRevisions, &proto.GetRevisionResponse{Id: rev.SnapEntryID.String(), SnapName: entry.Name, Sequence: uint64(rev.SequenceNumber)})
				}
			} else {
				errList = append(errList, &proto.Error{Code: errors.ResourceNotFound, Message: "Revision with id '" + revision.Id + "' not found"})
			}
			// If id is not provided, check if snapName and sequence are provided
		} else if revision.SnapName != "" && revision.Sequence != 0 {
			rev, err := s.repo.GetRevisionByNameAndSequence(revision.SnapName, uint(revision.Sequence))
			if err != nil {
				logrus.Error(err)
				errList = append(errList, &proto.Error{Code: errors.InternalServerError, Message: "Failed to get revision from database"})
				continue
			}
			if rev != nil {
				foundRevisions = append(foundRevisions, &proto.GetRevisionResponse{Id: rev.ID, SnapName: revision.SnapName, Sequence: uint64(rev.SequenceNumber)})
			} else {
				errList = append(errList, &proto.Error{Code: errors.ResourceNotFound, Message: "Revision with name " + revision.SnapName + " and revision " + fmt.Sprint(revision.Sequence) + " not found"})
			}
		} else {
			if revision.Id == "" && (revision.SnapName == "" || revision.Sequence == 0) {
				errList = append(errList, &proto.Error{Code: errors.MissingField, Message: "Id is required"})
			}
			if revision.SnapName == "" && revision.Id == "" {
				errList = append(errList, &proto.Error{Code: errors.MissingField, Message: "Snap name is required"})
			}
			if revision.Sequence == 0 && revision.Id == "" {
				errList = append(errList, &proto.Error{Code: errors.MissingField, Message: "Sequence is required"})
			}
		}
	}
	return &proto.GetRevisionsResponse{Revisions: foundRevisions, Errors: errList}, nil
}

// GetRevisionByNameAndSequence returns a single revision by snap name and sequence number
func (s *StoreLogic) GetRevisionByNameAndSequence(ctx context.Context, req *proto.GetRevisionRequest) (*proto.GetRevisionResponse, error) {
	errList := make([]*proto.Error, 0)

	if req.SnapName == "" {
		errList = append(errList, &proto.Error{Code: errors.MissingField, Message: "Snap name is required"})
		return &proto.GetRevisionResponse{Errors: errList}, nil
	}

	if req.Sequence == 0 {
		errList = append(errList, &proto.Error{Code: errors.MissingField, Message: "Sequence is required"})
		return &proto.GetRevisionResponse{Errors: errList}, nil
	}

	snapEntry, err := s.repo.GetEntryByName(req.SnapName, true)
	if err != nil {
		logrus.Error(err)
		errList = append(errList, &proto.Error{Code: errors.InternalServerError, Message: "Failed to get snap from database"})
		return &proto.GetRevisionResponse{Errors: errList}, err
	}
	if snapEntry == nil {
		errList = append(errList, &proto.Error{Code: errors.ResourceNotFound, Message: "Snap name '" + req.SnapName + "' not found"})
		return &proto.GetRevisionResponse{Errors: errList}, err
	}

	revision, err := s.repo.GetRevisionByNameAndSequence(snapEntry.Name, uint(req.Sequence))
	if err != nil {
		logrus.Error(err)
		errList = append(errList, &proto.Error{Code: errors.InternalServerError, Message: "Failed to get revision from database"})
		return &proto.GetRevisionResponse{Errors: errList}, err
	}
	if revision == nil {
		errList = append(errList, &proto.Error{Code: errors.ResourceNotFound, Message: "Revision with sequence " + fmt.Sprint(req.Sequence) + " not found"})
		return &proto.GetRevisionResponse{Errors: errList}, err
	}

	return &proto.GetRevisionResponse{Id: revision.ID, SnapName: snapEntry.Name, Sequence: uint64(revision.SequenceNumber)}, nil
}

func (s *StoreLogic) GetEntriesByAccountId(req *proto.GetEntriesByAccountIdRequest) (*proto.GetEntriesResponse, error) {
    errList := make([]*proto.Error, 0)
    if req.AccountId == "" {
        errList = append(errList, &proto.Error{Code: errors.MissingField, Message: "Account id is required"})
        return &proto.GetEntriesResponse{Errors: errList}, nil
    }
    accId, err := uuid.Parse(req.AccountId)
    if err != nil {
        logrus.Error(err)
        errList = append(errList, &proto.Error{Code: errors.InvalidField, Message: "Invalid UUID format"})
        return &proto.GetEntriesResponse{Errors: errList}, nil
    }

    entries, err := s.repo.GetEntriesByAccountId(accId,true)
    if err != nil {
        logrus.Debugf("Failed to get entries from database: %v", err)
        errList = append(errList, &proto.Error{Code: errors.InternalServerError, Message: "Failed to get entries from database"})
        return &proto.GetEntriesResponse{Errors: errList}, err
    }

    foundEntries := make([]*proto.GetEntryResponse, len(entries))
    for i, entry := range entries {
        foundEntries[i] = &proto.GetEntryResponse{
                Id: entry.ID.String(), 
                SnapName: entry.Name, 
                Type: entry.Type, 
                Confinement: entry.Confinement, 
                Base: entry.Base, 
                Private: entry.Private,
            }
    }

    return &proto.GetEntriesResponse{Entries: foundEntries}, nil
}

func saveFileToTemp(snapFile io.Reader) (string, string, error) {
	// Generate random file name for the new uploaded file so it doesn't override the old file with same name
	snapFileId := uuid.New().String()
	newFileName := snapFileId + ".snap"

	out, err := os.Create(path.Join("/tmp", newFileName))
	if err != nil {
		return "", "", err
	}
	defer out.Close()

	_, err = io.Copy(out, snapFile)
	if err != nil {
		return "", "", err
	}

	return newFileName, snapFileId, nil
}
