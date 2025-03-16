package logic

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/idlab-discover/kebeng/services/store/internal/errors"
	"github.com/idlab-discover/kebeng/services/store/internal/repositories"
	proto "github.com/idlab-discover/kebeng/services/store/proto"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
)

func TestRegisterSnapName(t *testing.T) {
	// Connect to an in-memory database
	db, err := sqlx.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	// Manually create the snap_entries table
	sql := `CREATE TABLE snap_entries (
		id TEXT PRIMARY KEY,
		name TEXT,
		type TEXT,
		confinement TEXT,
		base TEXT,
		private BOOLEAN,
		account_id TEXT,
		status TEXT,
		price REAL,
		store TEXT,
		icon_url TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		deleted_at TIMESTAMP
	);`

	// Execute the SQL to create the table
	_, err = db.Exec(sql)
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	repo := repositories.NewSnapsRepository(db)
	service := NewStoreLogic(repo)

	tests := []struct {
		name     string
		request  *proto.RegisterSnapNameRequest
		expected *proto.RegisterSnapNameResponse
	}{
		{
			name: "Missing field snap_name",
			request: &proto.RegisterSnapNameRequest{
				SnapName: "",
			},
			expected: &proto.RegisterSnapNameResponse{
				Errors: []*proto.Error{
					{
						Code:    errors.MissingField,
						Message: "snap_name is required",
					},
				},
			},
		},
		{
			name: "Success registering snap name",
			request: &proto.RegisterSnapNameRequest{
				SnapName: "test-snap",
			},
			expected: &proto.RegisterSnapNameResponse{
				Id:       "", // The ID will be checked separately
				SnapName: "test-snap",
				Errors:   nil,
			},
		},
		{
			name: "Duplicate snap name but dry_run == false",
			request: &proto.RegisterSnapNameRequest{
				SnapName: "test-snap",
				DryRun:   false,
			},
			expected: &proto.RegisterSnapNameResponse{
				Errors: []*proto.Error{
					{
						Code:    errors.AlreadyRegistered,
						Message: "The snap name 'test-snap' is already registered.",
					},
				},
			},
		},
		{
			name: "Duplicate snap name but dry_run == true",
			request: &proto.RegisterSnapNameRequest{
				SnapName: "test-snap",
				DryRun:   true,
			},
			expected: &proto.RegisterSnapNameResponse{
				Id:       "", // ID is set to "" when dry_run == true
				SnapName: "test-snap",
				Errors:   nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response, _ := service.RegisterSnapName(context.Background(), tt.request)
			assert.Equal(t, tt.expected.SnapName, response.SnapName)
			if tt.expected.Errors == nil && response.Id != "" {
				_, err := uuid.Parse(response.Id)
				assert.NoError(t, err) // Check that the ID is a valid UUID
			} else {
				assert.Equal(t, tt.expected.Errors, response.Errors)
			}
		})
	}
}

func TestGetEntries(t *testing.T) {
	// Connect to an in-memory database
	db, err := sqlx.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	// Create the snap_entries table
	sql := `CREATE TABLE snap_entries (
		id TEXT PRIMARY KEY,
		name TEXT,
		type TEXT,
		confinement TEXT,
		base TEXT,
		private BOOLEAN,
		account_id TEXT,
		status TEXT,
		price REAL,
		store TEXT,
		icon_url TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		deleted_at TIMESTAMP
	);`

	_, err = db.Exec(sql)
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	repo := repositories.NewSnapsRepository(db)
	service := NewStoreLogic(repo)

	// Seed database with test entry
	testID := uuid.New().String()
	db.Exec("INSERT INTO snap_entries (id, name, type, confinement, base, private) VALUES (?, ?, ?, ?, ?, ?)",
		testID, "test-snap", "app", "strict", "core20", false)

	tests := []struct {
		name     string
		request  *proto.GetEntriesRequest
		expected *proto.GetEntriesResponse
	}{
		{
			name: "Retrieve existing entry by ID",
			request: &proto.GetEntriesRequest{
				Entries: []*proto.GetEntryRequest{{Id: testID}},
			},
			expected: &proto.GetEntriesResponse{
				Entries: []*proto.GetEntryResponse{
					{
						Id:          testID,
						SnapName:    "test-snap",
						Type:        "app",
						Confinement: "strict",
						Base:        "core20",
						Private:     false,
					},
				},
				Errors: nil,
			},
		},
		{
			name: "Invalid UUID format",
			request: &proto.GetEntriesRequest{
				Entries: []*proto.GetEntryRequest{{Id: "invalid-uuid"}},
			},
			expected: &proto.GetEntriesResponse{
				Entries: nil,
				Errors: []*proto.Error{
					{
						Code:    errors.InvalidField,
						Message: "Invalid UUID format",
					},
				},
			},
		},
		{
			name: "Retrieve non-existing entry by ID",
			request: &proto.GetEntriesRequest{
				Entries: []*proto.GetEntryRequest{{Id: uuid.New().String()}},
			},
			expected: &proto.GetEntriesResponse{
				Entries: nil,
				Errors: []*proto.Error{
					{
						Code:    errors.ResourceNotFound,
						Message: "Entry with id '" + testID + "' not found",
					},
				},
			},
		},
		{
			name: "Retrieve non-existing entry by name",
			request: &proto.GetEntriesRequest{
				Entries: []*proto.GetEntryRequest{{Name: "non-existent"}},
			},
			expected: &proto.GetEntriesResponse{
				Entries: nil,
				Errors: []*proto.Error{
					{
						Code:    errors.ResourceNotFound,
						Message: "Entry with name 'non-existent' not found",
					},
				},
			},
		},
		{
			name: "Retrieve existing entry by name",
			request: &proto.GetEntriesRequest{
				Entries: []*proto.GetEntryRequest{{Name: "test-snap"}},
			},
			expected: &proto.GetEntriesResponse{
				Entries: []*proto.GetEntryResponse{
					{
						Id:          testID,
						SnapName:    "test-snap",
						Type:        "app",
						Confinement: "strict",
						Base:        "core20",
						Private:     false,
					},
				},
				Errors: nil,
			},
		},
		{
			name: "Missing ID and name in request",
			request: &proto.GetEntriesRequest{
				Entries: []*proto.GetEntryRequest{{}},
			},
			expected: &proto.GetEntriesResponse{
				Entries: nil,
				Errors: []*proto.Error{
					{
						Code:    errors.MissingField,
						Message: "Id or name is required",
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response, _ := service.GetEntries(context.Background(), tt.request)
			assert.Equal(t, len(tt.expected.Entries), len(response.Entries))
			assert.Equal(t, len(tt.expected.Errors), len(response.Errors))
			if len(tt.expected.Entries) > 0 {
				assert.Equal(t, tt.expected.Entries[0].SnapName, response.Entries[0].SnapName)
			}
		})
	}
}

func TestGetEntryById(t *testing.T) {
	// Connect to an in-memory database
	db, err := sqlx.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	// Create the snap_entries table
	sql := `CREATE TABLE snap_entries (
		id TEXT PRIMARY KEY,
		name TEXT,
		type TEXT,
		confinement TEXT,
		base TEXT,
		private BOOLEAN,
		account_id TEXT,
		status TEXT,
		price REAL,
		store TEXT,
		icon_url TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		deleted_at TIMESTAMP
	);`

	_, err = db.Exec(sql)
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	repo := repositories.NewSnapsRepository(db)
	service := NewStoreLogic(repo)

	// Seed database with test entry
	testID := uuid.New().String()
	db.Exec("INSERT INTO snap_entries (id, name, type, confinement, base, private) VALUES (?, ?, ?, ?, ?, ?)",
		testID, "test-snap", "app", "strict", "core20", false)

	tests := []struct {
		name     string
		request  *proto.GetEntryRequest
		expected *proto.GetEntryResponse
	}{
		{
			name: "Retrieve existing entry by ID",
			request: &proto.GetEntryRequest{
				Id: testID,
			},
			expected: &proto.GetEntryResponse{
				Id:          testID,
				SnapName:    "test-snap",
				Type:        "app",
				Confinement: "strict",
				Base:        "core20",
				Private:     false,
			},
		},
		{
			name: "Retrieve non-existing entry by ID",
			request: &proto.GetEntryRequest{
				Id: "550e8400-e29b-41d4-a716-446655440000",
			},
			expected: &proto.GetEntryResponse{
				Errors: []*proto.Error{
					{
						Code:    errors.ResourceNotFound,
						Message: "Snap with id '" + "550e8400-e29b-41d4-a716-446655440000" + "' not found",
					},
				},
			},
		},
		{
			name: "Missing ID in request",
			request: &proto.GetEntryRequest{
				Id: "",
			},
			expected: &proto.GetEntryResponse{
				Errors: []*proto.Error{
					{
						Code:    errors.MissingField,
						Message: "Id is required",
					},
				},
			},
		},
		{
			name: "Invalid UUID format",
			request: &proto.GetEntryRequest{
				Id: "invalid-uuid",
			},
			expected: &proto.GetEntryResponse{
				Errors: []*proto.Error{
					{
						Code:    errors.InvalidField,
						Message: "Invalid UUID format",
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response, _ := service.GetEntryById(context.Background(), tt.request)
			assert.Equal(t, tt.expected.Id, response.Id)
			assert.Equal(t, len(tt.expected.Errors), len(response.Errors))
			if len(tt.expected.Errors) > 0 {
				assert.Equal(t, tt.expected.Errors[0].Message, response.Errors[0].Message)
			}
		})
	}
}

func TestGetEntryByName(t *testing.T) {
	// Connect to an in-memory database
	db, err := sqlx.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	// Create the snap_entries table
	sql := `CREATE TABLE snap_entries (
		id TEXT PRIMARY KEY,
		name TEXT,
		type TEXT,
		confinement TEXT,
		base TEXT,
		private BOOLEAN,
		account_id TEXT,
		status TEXT,
		price REAL,
		store TEXT,
		icon_url TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		deleted_at TIMESTAMP
	);`

	_, err = db.Exec(sql)
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	repo := repositories.NewSnapsRepository(db)
	service := NewStoreLogic(repo)

	// Seed database with test entry
	testID := uuid.New().String()
	testName := "test-snap"
	db.Exec("INSERT INTO snap_entries (id, name, type, confinement, base, private) VALUES (?, ?, ?, ?, ?, ?)",
		testID, testName, "app", "strict", "core20", false)

	tests := []struct {
		name     string
		request  *proto.GetEntryRequest
		expected *proto.GetEntryResponse
	}{
		{
			name: "Retrieve existing entry by name",
			request: &proto.GetEntryRequest{
				Name: testName,
			},
			expected: &proto.GetEntryResponse{
				Id:          testID,
				SnapName:    testName,
				Type:        "app",
				Confinement: "strict",
				Base:        "core20",
				Private:     false,
			},
		},
		{
			name: "Retrieve non-existing entry by name",
			request: &proto.GetEntryRequest{
				Name: "non-existent",
			},
			expected: &proto.GetEntryResponse{
				Errors: []*proto.Error{
					{
						Code:    errors.ResourceNotFound,
						Message: "Snap with name 'non-existent' not found",
					},
				},
			},
		},
		{
			name: "Missing name in request",
			request: &proto.GetEntryRequest{
				Name: "",
			},
			expected: &proto.GetEntryResponse{
				Errors: []*proto.Error{
					{
						Code:    errors.MissingField,
						Message: "Name is required",
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response, _ := service.GetEntryByName(context.Background(), tt.request)
			assert.Equal(t, tt.expected.Id, response.Id)
			assert.Equal(t, len(tt.expected.Errors), len(response.Errors))
			if len(tt.expected.Errors) > 0 {
				assert.Equal(t, tt.expected.Errors[0].Message, response.Errors[0].Message)
			}
		})
	}
}

func TestGetRevisions(t *testing.T) {
	// Connect to an in-memory database
	db, err := sqlx.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	// Create the snap_revisions and snap_entries tables
	sql := `CREATE TABLE snap_entries (
		id TEXT PRIMARY KEY,
		name TEXT,
		type TEXT,
		confinement TEXT,
		base TEXT,
		private BOOLEAN,
		account_id TEXT,
		status TEXT,
		price REAL,
		store TEXT,
		icon_url TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		deleted_at TIMESTAMP
	);
	
	CREATE TABLE snap_revisions (
		id TEXT PRIMARY KEY,
		snap_entry_id TEXT,
		sequence_number bigint,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		deleted_at TIMESTAMP,
		FOREIGN KEY(snap_entry_id) REFERENCES snap_entries(id)
	);`

	_, err = db.Exec(sql)
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	repo := repositories.NewSnapsRepository(db)
	service := NewStoreLogic(repo)

	// Seed database with test entry and revision
	testSnapID := uuid.New().String()
	testRevisionID := uuid.New().String()
	testSnapName := "test-snap"
	testSequence := uint64(1)
	db.Exec("INSERT INTO snap_entries (id, name) VALUES (?, ?)", testSnapID, testSnapName)
	db.Exec("INSERT INTO snap_revisions (id, snap_entry_id, sequence_number) VALUES (?, ?, ?)", testRevisionID, testSnapID, testSequence)

	tests := []struct {
		name     string
		request  *proto.GetRevisionsRequest
		expected *proto.GetRevisionsResponse
	}{
		{
			name: "Retrieve existing revision by ID",
			request: &proto.GetRevisionsRequest{
				Revisions: []*proto.GetRevisionRequest{{Id: testRevisionID}},
			},
			expected: &proto.GetRevisionsResponse{
				Revisions: []*proto.GetRevisionResponse{{Id: testRevisionID, SnapName: testSnapName, Sequence: testSequence}},
				Errors:    make([]*proto.Error, 0),
			},
		},
		{
			name: "Revision with ID not found",
			request: &proto.GetRevisionsRequest{
				Revisions: []*proto.GetRevisionRequest{{Id: "550e8400-e29b-41d4-a716-446655440000"}},
			},
			expected: &proto.GetRevisionsResponse{
				Revisions: nil,
				Errors: []*proto.Error{
					{Code: errors.ResourceNotFound, Message: "Revision with id '" + "550e8400-e29b-41d4-a716-446655440000" + "' not found"},
				},
			},
		},
		{
			name: "Retrieve existing revision by name and sequence",
			request: &proto.GetRevisionsRequest{
				Revisions: []*proto.GetRevisionRequest{{SnapName: testSnapName, Sequence: testSequence}},
			},
			expected: &proto.GetRevisionsResponse{
				Revisions: []*proto.GetRevisionResponse{{Id: testRevisionID, SnapName: testSnapName, Sequence: testSequence}},
				Errors:    make([]*proto.Error, 0),
			},
		},
		{
			name: "Retrieve non-existing revision by name and sequence",
			request: &proto.GetRevisionsRequest{
				Revisions: []*proto.GetRevisionRequest{{SnapName: "non-existing-snap", Sequence: testSequence}},
			},
			expected: &proto.GetRevisionsResponse{
				Revisions: nil,
				Errors: []*proto.Error{
					{Code: errors.ResourceNotFound, Message: "Revision with name '" + "non-existing-snap" + "' and revision '" + fmt.Sprint(testSequence) + "' not found"},
				},
			},
		},
		{
			name: "Missing ID and name/sequence in request",
			request: &proto.GetRevisionsRequest{
				Revisions: []*proto.GetRevisionRequest{{}},
			},
			expected: &proto.GetRevisionsResponse{
				Revisions: nil,
				Errors: []*proto.Error{
					{Code: errors.MissingField, Message: "Id is required"},
					{Code: errors.MissingField, Message: "Snap name is required"},
					{Code: errors.MissingField, Message: "Sequence is required"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response, _ := service.GetRevisions(context.Background(), tt.request)
			assert.Equal(t, len(tt.expected.Revisions), len(response.Revisions))
			assert.Equal(t, len(tt.expected.Errors), len(response.Errors))
			if len(tt.expected.Revisions) > 0 {
				assert.Equal(t, tt.expected.Revisions[0].Id, response.Revisions[0].Id)
				assert.Equal(t, tt.expected.Revisions[0].SnapName, response.Revisions[0].SnapName)
				assert.Equal(t, tt.expected.Revisions[0].Sequence, response.Revisions[0].Sequence)
			}
		})
	}
}

func TestGetRevisionByNameAndSequence(t *testing.T) {
	// Connect to an in-memory database
	db, err := sqlx.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	// Create the snap_revisions and snap_entries tables
	sql := `CREATE TABLE snap_entries (
		id TEXT PRIMARY KEY,
		name TEXT,
		type TEXT,
		confinement TEXT,
		base TEXT,
		private BOOLEAN,
		account_id TEXT,
		status TEXT,
		price REAL,
		store TEXT,
		icon_url TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		deleted_at TIMESTAMP
	);
	
	CREATE TABLE snap_revisions (
		id TEXT PRIMARY KEY,
		snap_entry_id TEXT,
		sequence_number bigint,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		deleted_at TIMESTAMP,
		FOREIGN KEY(snap_entry_id) REFERENCES snap_entries(id)
	);`

	_, err = db.Exec(sql)
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	repo := repositories.NewSnapsRepository(db)
	service := NewStoreLogic(repo)

	// Seed test data
	testSnapID := uuid.New().String()
	testRevisionID := uuid.New().String()
	testSnapName := "test-snap"
	testSequence := uint64(1)
	db.Exec("INSERT INTO snap_entries (id, name) VALUES (?, ?)", testSnapID, testSnapName)
	db.Exec("INSERT INTO snap_revisions (id, snap_entry_id, sequence_number) VALUES (?, ?, ?)", testRevisionID, testSnapID, testSequence)

	tests := []struct {
		name     string
		request  *proto.GetRevisionRequest
		expected *proto.GetRevisionResponse
	}{
		{
			name: "Valid revision retrieval",
			request: &proto.GetRevisionRequest{
				SnapName: testSnapName,
				Sequence: testSequence,
			},
			expected: &proto.GetRevisionResponse{
				Id:       testRevisionID,
				SnapName: testSnapName,
				Sequence: testSequence,
			},
		},
		{
			name: "Missing snap name",
			request: &proto.GetRevisionRequest{
				SnapName: "",
				Sequence: testSequence,
			},
			expected: &proto.GetRevisionResponse{
				Errors: []*proto.Error{{Code: errors.MissingField, Message: "Snap name is required"}},
			},
		},
		{
			name: "Missing sequence number",
			request: &proto.GetRevisionRequest{
				SnapName: testSnapName,
				Sequence: 0,
			},
			expected: &proto.GetRevisionResponse{
				Errors: []*proto.Error{{Code: errors.MissingField, Message: "Sequence is required"}},
			},
		},
		{
			name: "Non-existing snap name",
			request: &proto.GetRevisionRequest{
				SnapName: "unknown-snap",
				Sequence: testSequence,
			},
			expected: &proto.GetRevisionResponse{
				Errors: []*proto.Error{{Code: errors.ResourceNotFound, Message: "Snap name 'unknown-snap' not found"}},
			},
		},
		{
			name: "Non-existing sequence number",
			request: &proto.GetRevisionRequest{
				SnapName: testSnapName,
				Sequence: 999,
			},
			expected: &proto.GetRevisionResponse{
				Errors: []*proto.Error{{Code: errors.ResourceNotFound, Message: "Revision with sequence 999 not found"}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response, _ := service.GetRevisionByNameAndSequence(context.Background(), tt.request)
			assert.Equal(t, len(tt.expected.Errors), len(response.Errors))
			if len(tt.expected.Errors) == 0 {
				assert.Equal(t, tt.expected.Id, response.Id)
				assert.Equal(t, tt.expected.SnapName, response.SnapName)
				assert.Equal(t, tt.expected.Sequence, response.Sequence)
			}
		})
	}
}
