package logic

import (
	"bytes"
	"context"
	"io"
	"os"
	"path"

	"fmt"

	"github.com/google/uuid"
	cerrors "github.com/idlab-discover/kebeng/services/store/internal/errors"
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
	el := make([]*proto.Error, 0)
	snapFileName, id, err := saveFileToTemp(bytes.NewReader(req.File))
	if err != nil {
		logrus.Errorf("Failed to save file to temp storage: %v", err)
		el = append(el, &proto.Error{Code: cerrors.InternalServerError, Message: "Failed to save file to temp storage"})
		return &proto.UploadSnapResponse{Errors: el}, err
	}

	objectstore := objectstore.NewObjectStore()
	tmpPath := path.Join(os.TempDir(), snapFileName)

	size, err := objectstore.SaveFileToBucket("unscanned", tmpPath)
	if err != nil {
		logrus.Errorf("Failed to save file to object store: %v", err)
		el = append(el, &proto.Error{Code: cerrors.InternalServerError, Message: "Failed to save file to object store"})
		return &proto.UploadSnapResponse{Errors: el}, err
	}

	// addSnap() adds snap to snap_entries table
	_, err = s.repo.AddSnap(snapFileName, size, uuid.New()) // uuid.New() is a placeholder for account id that is going to be added later throught the context
	if err != nil {
		logrus.Error(err)
		el = append(el, &proto.Error{Code: cerrors.InternalServerError, Message: "Failed to add snap to database"})
		return &proto.UploadSnapResponse{Errors: el}, err
	}

	return &proto.UploadSnapResponse{Id: id, DisplayName: snapFileName}, nil
}

func (s *StoreLogic) RegisterSnapName(ctx context.Context, req *proto.RegisterSnapNameRequest) (*proto.RegisterSnapNameResponse, error) {
	el := make([]*proto.Error, 0)

	if req.SnapName == "" {
		el = append(el, &proto.Error{Code: cerrors.MissingField, Message: "snap_name is required"})
		return &proto.RegisterSnapNameResponse{Errors: el}, nil
	}

	// TODO: check if snap name is valid (it should only have ASCII lowercase letters, numbers, and hyphens, and must have at least one letter)

	// First check if the snap name is already registered
	snapEntry, err := s.repo.GetEntryByName(req.SnapName, false)
	if err != nil {
		switch {
		case cerrors.Is(err, cerrors.DatabaseError):
			el = append(el, &proto.Error{Code: cerrors.DatabaseError, Message: "Failed to get snap from database"})

		case cerrors.Is(err, cerrors.ResourceNotFound):
			// If snap name is not found, do nothing -> this is what we want
			break
		default:
			logrus.Error(err)
			el = append(el, &proto.Error{Code: cerrors.InternalServerError, Message: "something went wrong while getting snap"})
		}
	}

	// If dryRun is true, we only check if the snap name is already registered
	if req.DryRun {
		return &proto.RegisterSnapNameResponse{SnapName: req.SnapName}, nil // Id will be set to empty string, docs say it should be null, but nil can't be assigned to string -> see later if this is a problem
	}

	// If dryRun is false, but snap name is already registered, return an error
	if snapEntry != nil {
		el = append(el, &proto.Error{Code: cerrors.AlreadyRegistered, Message: "The snap name '" + req.SnapName + "' is already registered."})
		return &proto.RegisterSnapNameResponse{Errors: el}, err
	}

	// If there is no snap with the same name and dry_run == false, register the snap name
	snapEntry, err = s.repo.RegisterSnap(req.SnapName, req.IsPrivate)
	if err != nil {
		logrus.Error(err)
		el = append(el, &proto.Error{Code: cerrors.InternalServerError, Message: "Failed to register snap name"})
		return &proto.RegisterSnapNameResponse{Errors: el}, err
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
//   - *proto.GetEntriesResponse: The response containing the list of found entries and any cerrors encountered.
//   - error: This error is only not nil of proto fails. Errors while retrieving entries are added to the GetEntriesResponse.
func (s *StoreLogic) GetEntries(ctx context.Context, req *proto.GetEntriesRequest) (*proto.GetEntriesResponse, error) {
	el := make([]*proto.Error, 0)
	foundEntries := make([]*proto.GetEntryResponse, 0)

	for _, entry := range req.Entries {
		// First try to retrieve the entry by its ID
		if entry.Id != "" {
			id, err := uuid.Parse(entry.Id)
			if err != nil {
				logrus.Errorf("Failed to parse UUID '%s':", entry.Id)
				el = append(el, &proto.Error{Code: cerrors.InvalidField, Message: fmt.Sprintf("Invalid UUID format for id '%s'", entry.Id)})
				continue
			}

			snapEntry, err := s.repo.GetEntryById(id, false)
			if err != nil {
				switch {
				case cerrors.Is(err, cerrors.ResourceNotFound):
					el = append(el, &proto.Error{Code: cerrors.ResourceNotFound, Message: "Entry with id '" + entry.Id + "' not found"})
				case cerrors.Is(err, cerrors.DatabaseError):
					el = append(el, &proto.Error{Code: cerrors.DatabaseError, Message: "Failed to get entry from database"})
				default:
					logrus.Error(err)
					el = append(el, &proto.Error{Code: cerrors.InternalServerError, Message: "Failed to get entry from database"})
				}
				continue
			}

			foundEntries = append(foundEntries, &proto.GetEntryResponse{
				Id:          snapEntry.ID.String(),
				SnapName:    snapEntry.Name,
				Type:        snapEntry.Type,
				Confinement: snapEntry.Confinement,
				Base:        snapEntry.Base,
				Private:     snapEntry.Private,
			})

			// If ID is not given, try to retrieve the entry by its name
		} else if entry.Name != "" {
			snapEntry, err := s.repo.GetEntryByName(entry.Name, false)
			if err != nil {
				switch {
				case cerrors.Is(err, cerrors.ResourceNotFound):
					el = append(el, &proto.Error{Code: cerrors.ResourceNotFound, Message: "Entry with name '" + entry.Name + "' not found"})
				case cerrors.Is(err, cerrors.DatabaseError):
					el = append(el, &proto.Error{Code: cerrors.DatabaseError, Message: "Failed to get entry from database"})
				default:
					logrus.Error(err)
					el = append(el, &proto.Error{Code: cerrors.InternalServerError, Message: "Failed to get entry from database"})
				}
				continue
			}

			foundEntries = append(foundEntries, &proto.GetEntryResponse{
				Id:          snapEntry.ID.String(),
				SnapName:    snapEntry.Name,
				Type:        snapEntry.Type,
				Confinement: snapEntry.Confinement,
				Base:        snapEntry.Base,
				Private:     snapEntry.Private,
			})

		} else {
			el = append(el, &proto.Error{
				Code:    cerrors.MissingField,
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
//   - *proto.GetEntryResponse: The response containing the found snap entry and any cerrors encountered.
//   - error: This error is only not nil of proto fails. Errors while retrieving entry are added to the GetEntriesResponse.
func (s *StoreLogic) GetEntryById(ctx context.Context, req *proto.GetEntryRequest) (*proto.GetEntryResponse, error) {
	el := make([]*proto.Error, 0)
	if req.Id == "" {
		el = append(el, &proto.Error{Code: cerrors.MissingField, Message: "Id is required"})
		return &proto.GetEntryResponse{Errors: el}, fmt.Errorf("id is required")
	}

	id, err := uuid.Parse(req.Id)
	if err != nil {
		logrus.Error(err)
		el = append(el, &proto.Error{Code: cerrors.InvalidField, Message: "Invalid UUID format"})
		return &proto.GetEntryResponse{Errors: el}, nil
	}

	snapEntry, err := s.repo.GetEntryById(id, false)
	if err != nil {
		switch {
		case cerrors.Is(err, cerrors.ResourceNotFound):
			el = append(el, &proto.Error{Code: cerrors.ResourceNotFound, Message: "Snap with id '" + req.Id + "' not found"})

		case cerrors.Is(err, cerrors.DatabaseError):
			el = append(el, &proto.Error{Code: cerrors.DatabaseError, Message: "Failed to get snap from database"})

		default:
			logrus.Error(err)
			el = append(el, &proto.Error{Code: cerrors.InternalServerError, Message: "something went wrong while getting snap"})
		}
		return &proto.GetEntryResponse{Errors: el}, err
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
//   - *proto.GetEntryResponse: The response containing the found snap entry and any cerrors encountered.
//   - error: An error if the operation fails.
func (s *StoreLogic) GetEntryByName(ctx context.Context, req *proto.GetEntryRequest) (*proto.GetEntryResponse, error) {
	el := make([]*proto.Error, 0)
	if req.Name == "" {
		el = append(el, &proto.Error{Code: cerrors.MissingField, Message: "Name is required"})
		return &proto.GetEntryResponse{Errors: el}, fmt.Errorf("name is required")
	}

	snapEntry, err := s.repo.GetEntryByName(req.Name, false)
	if err != nil {
		switch {
		case cerrors.Is(err, cerrors.ResourceNotFound):
			el = append(el, &proto.Error{Code: cerrors.ResourceNotFound, Message: "Snap with name '" + req.Name + "' not found"})

		case cerrors.Is(err, cerrors.DatabaseError):
			el = append(el, &proto.Error{Code: cerrors.DatabaseError, Message: "Failed to get snap from database"})

		default:
			logrus.Error(err)
			el = append(el, &proto.Error{Code: cerrors.InternalServerError, Message: "something went wrong while getting snap"})
		}
		return &proto.GetEntryResponse{Errors: el}, err
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
//   - *proto.GetRevisionsResponse: The response containing the list of found revisions and any cerrors encountered.
//   - error: This error is only not nil of proto fails. Errors while retrieving revisions are added to the GetEntriesResponse.
func (s *StoreLogic) GetRevisions(ctx context.Context, req *proto.GetRevisionsRequest) (*proto.GetRevisionsResponse, error) {
	el := make([]*proto.Error, 0)
	foundRevisions := make([]*proto.GetRevisionResponse, 0)

	// Each GetRevisionRequest will contain either id or snapName and sequence
	for _, revision := range req.Revisions {
		// First check if id is provided
		if revision.Id != "" {
			rev, err := s.repo.GetRevisionById(revision.Id)
			if err != nil {
				switch {
				case cerrors.Is(err, cerrors.ResourceNotFound):
					el = append(el, &proto.Error{Code: cerrors.ResourceNotFound, Message: "Revision with id '" + revision.Id + "' not found"})

				case cerrors.Is(err, cerrors.DatabaseError):
					el = append(el, &proto.Error{Code: cerrors.DatabaseError, Message: "Failed to get revision from database"})

				default:
					logrus.Error(err)
					el = append(el, &proto.Error{Code: cerrors.InternalServerError, Message: "Failed to get revision from database"})
				}
				continue
			}

			entry, err := s.repo.GetEntryById(rev.SnapEntryID, false)
			if err != nil {
				switch {
				case cerrors.Is(err, cerrors.ResourceNotFound):
					el = append(el, &proto.Error{Code: cerrors.ResourceNotFound, Message: "Snap entry with id '" + rev.SnapEntryID.String() + "' not found"})

				case cerrors.Is(err, cerrors.DatabaseError):
					el = append(el, &proto.Error{Code: cerrors.DatabaseError, Message: "Failed to get snap entry from database"})

				default:
					logrus.Error(err)
					el = append(el, &proto.Error{Code: cerrors.InternalServerError, Message: "Failed to get snap entry from database"})
				}
				continue
			}

			foundRevisions = append(foundRevisions, &proto.GetRevisionResponse{Id: rev.ID, SnapName: entry.Name, Sequence: uint64(rev.SequenceNumber)})

			// If id is not provided, check if snapName and sequence are provided
		} else if revision.SnapName != "" && revision.Sequence != 0 {
			// err is a database error
			rev, err := s.repo.GetRevisionByNameAndSequence(revision.SnapName, uint(revision.Sequence))
			if err != nil {
				switch {
				case cerrors.Is(err, cerrors.ResourceNotFound):
					el = append(el, &proto.Error{Code: cerrors.ResourceNotFound, Message: "Revision with sequence " + fmt.Sprint(revision.Sequence) + " not found"})

				case cerrors.Is(err, cerrors.DatabaseError):
					el = append(el, &proto.Error{Code: cerrors.DatabaseError, Message: "Failed to get revision from database"})

				default:
					logrus.Error(err)
					el = append(el, &proto.Error{Code: cerrors.InternalServerError, Message: "something went wrong while getting revision"})
				}
				continue
			}

			foundRevisions = append(foundRevisions, &proto.GetRevisionResponse{Id: rev.ID, SnapName: revision.SnapName, Sequence: uint64(rev.SequenceNumber)})

		} else {
			if revision.Id == "" && (revision.SnapName == "" || revision.Sequence == 0) {
				el = append(el, &proto.Error{Code: cerrors.MissingField, Message: "Id is required"})
			}
			if revision.SnapName == "" && revision.Id == "" {
				el = append(el, &proto.Error{Code: cerrors.MissingField, Message: "Snap name is required"})
			}
			if revision.Sequence == 0 && revision.Id == "" {
				el = append(el, &proto.Error{Code: cerrors.MissingField, Message: "Sequence is required"})
			}
		}
	}
	return &proto.GetRevisionsResponse{Revisions: foundRevisions, Errors: el}, nil
}

// GetRevisionByNameAndSequence returns a single revision by snap name and sequence number
func (s *StoreLogic) GetRevisionByNameAndSequence(ctx context.Context, req *proto.GetRevisionRequest) (*proto.GetRevisionResponse, error) {
	el := make([]*proto.Error, 0)

	if req.SnapName == "" {
		el = append(el, &proto.Error{Code: cerrors.MissingField, Message: "Snap name is required"})
		return &proto.GetRevisionResponse{Errors: el}, nil
	}

	if req.Sequence == 0 {
		el = append(el, &proto.Error{Code: cerrors.MissingField, Message: "Sequence is required"})
		return &proto.GetRevisionResponse{Errors: el}, nil
	}

	snapEntry, err := s.repo.GetEntryByName(req.SnapName, false)
	if err != nil {
		switch {
		case cerrors.Is(err, cerrors.ResourceNotFound):
			el = append(el, &proto.Error{Code: cerrors.ResourceNotFound, Message: "Snap with name '" + req.SnapName + "' not found"})

		case cerrors.Is(err, cerrors.DatabaseError):
			el = append(el, &proto.Error{Code: cerrors.DatabaseError, Message: "Failed to get snap from database"})

		default:
			logrus.Error(err)
			el = append(el, &proto.Error{Code: cerrors.InternalServerError, Message: "something went wrong while getting snap"})
		}
		return &proto.GetRevisionResponse{Errors: el}, err
	}

	revision, err := s.repo.GetRevisionByNameAndSequence(snapEntry.Name, uint(req.Sequence))
	if err != nil {
		switch {
		case cerrors.Is(err, cerrors.ResourceNotFound):
			el = append(el, &proto.Error{Code: cerrors.ResourceNotFound, Message: "Revision with sequence " + fmt.Sprint(req.Sequence) + " not found"})

		case cerrors.Is(err, cerrors.DatabaseError):
			el = append(el, &proto.Error{Code: cerrors.DatabaseError, Message: "Failed to get revision from database"})

		default:
			logrus.Error(err)
			el = append(el, &proto.Error{Code: cerrors.InternalServerError, Message: "something went wrong while getting revision"})
		}
		return &proto.GetRevisionResponse{Errors: el}, err
	}

	return &proto.GetRevisionResponse{Id: revision.ID, SnapName: snapEntry.Name, Sequence: uint64(revision.SequenceNumber)}, nil
}

func (s *StoreLogic) GetEntriesByAccountId(ctx context.Context, req *proto.GetEntriesByAccountIdRequest) (*proto.GetEntriesResponse, error) {
    el := make([]*proto.Error, 0)
    if req.AccountId == "" {
        el = append(el, &proto.Error{Code: cerrors.MissingField, Message: "Account id is required"})
        return &proto.GetEntriesResponse{Errors: el}, nil
    }
    accId, err := uuid.Parse(req.AccountId)
    if err != nil {
        logrus.Error(err)
        el = append(el, &proto.Error{Code: cerrors.InvalidField, Message: "Invalid UUID format"})
        return &proto.GetEntriesResponse{Errors: el}, nil
    }

    entries, err := s.repo.GetEntriesByAccountId(accId,true)
    if err != nil {
        logrus.Debugf("Failed to get entries from database: %v", err)
        el = append(el, &proto.Error{Code: cerrors.InternalServerError, Message: "Failed to get entries from database"})
        return &proto.GetEntriesResponse{Errors: el}, err
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
                PublisherId: entry.AccountID.String(),
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
                Code: cerrors.MissingField, 
                Message: "Entry id is required",
            })
            continue
        }

        entryId, err := uuid.Parse(entryIdReq.EntryId)
        if err != nil {
            logrus.Error(err)
            el = append(el, &proto.Error{
                Code: cerrors.InvalidField, 
                Message: "Invalid UUID format",
            })
            continue
        }

        revisions, err := s.repo.GetRevisionsByEntryId(entryId)
        if err != nil {
            logrus.Debugf("Failed to get revisions from database: %v", err)
            el = append(el, &proto.Error{
                Code: cerrors.InternalServerError, 
                Message: "Failed to get revisions from database",
            })
            // add empty response to keep the order of responses
            responses = append(responses, &proto.GetRevisionsByEntryIdResponse{
                EntryId: entryId.String(),
                Errors: el,
            })
            continue
        }
        // revision were found so convert them in response format
        revisionsProto := make([]*proto.GetRevisionResponse, len(revisions))
        for i, rev := range revisions {
            revisionsProto[i] = &proto.GetRevisionResponse{
                Id: rev.ID,
                SnapName: rev.SnapFilename,
                Sequence: uint64(rev.SequenceNumber),
            }
        }
        // add to response
        responses = append(responses, &proto.GetRevisionsByEntryIdResponse{
            EntryId: entryId.String(),
            Revisions: revisionsProto,
        })
    }
    return &proto.GetRevisionsByEntryIdResponses{Responses: responses}, nil
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
