package logic

import (
	"context"
	"testing"

	"github.com/idlab-discover/kebeng/services/store/internal/errors"
	"github.com/idlab-discover/kebeng/services/store/internal/repositories"
	proto "github.com/idlab-discover/kebeng/services/store/proto"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestRegisterSnapName(t *testing.T) {
	// Connect to an in-memory database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
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
	err = db.Exec(sql).Error
	if err != nil {
		panic("Failed to create table")
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response, _ := service.RegisterSnapName(context.Background(), tt.request)
			assert.Equal(t, tt.expected, response)
		})
	}
}
