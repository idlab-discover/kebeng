package logic

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/idlab-discover/kebeng/services/assertion/internal/config"
	"github.com/idlab-discover/kebeng/services/assertion/internal/errors"
	"github.com/idlab-discover/kebeng/services/assertion/internal/models"
	"github.com/idlab-discover/kebeng/services/assertion/internal/repositories"
	proto "github.com/idlab-discover/kebeng/services/assertion/proto"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestProcessSnapBuildAssertion(t *testing.T) {

	cfg := &config.Config{}

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	// Manually create the assertions table
	sql := `CREATE TABLE assertions (
		id TEXT PRIMARY KEY,
		assertion TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		deleted_at TIMESTAMP
	);`

	// Execute the SQL to create the table
	err = db.Exec(sql).Error
	if err != nil {
		panic("Failed to create table")
	}

	// Manually generate and set the UUID for testing
	assertion := models.Assertion{
		ID:        uuid.New(),
		Assertion: "Test assertion",
	}

	err = db.Create(&assertion).Error
	if err != nil {
		t.Fatalf("Failed to create assertion: %v", err)
	}

	// Verify the assertion was inserted correctly
	var retrievedAssertion models.Assertion
	err = db.First(&retrievedAssertion).Error
	if err != nil {
		t.Fatalf("Failed to retrieve assertion: %v", err)
	}

	t.Logf("Created assertion: %+v", retrievedAssertion)

	repo := repositories.NewAssertionRepository(db)
	service := NewAssertionService(repo, cfg)

	tests := []struct {
		name     string
		request  *proto.SnapBuildAssertionRequest
		expected *proto.SnapBuildAssertionResponse
	}{
		{
			name: "Missing Assertion",
			request: &proto.SnapBuildAssertionRequest{
				Assertion: nil,
			},
			expected: &proto.SnapBuildAssertionResponse{
				Errors: []*proto.Error{
					{
						Code:    errors.MissingField,
						Message: "Assertion field is required",
					},
				},
			},
		},
		{
			name: "Invalid Assertion",
			request: &proto.SnapBuildAssertionRequest{
				Assertion: []byte("invalid assertion"),
			},
			expected: &proto.SnapBuildAssertionResponse{
				Errors: []*proto.Error{
					{
						Code:    errors.Invalid,
						Message: "not a valid snap-build assertion: missing required field: type",
					},
				},
			},
		},
		{
			name: "Valid Assertion",
			request: &proto.SnapBuildAssertionRequest{
				Assertion: []byte("type: snap-build\nauthority-id: a54a55z8d24a84\nsnap-sha3-384: IMGfaV_cMB_nAzrhAKt_v5IqMC-OuO4QlSN4mIKWsyBU6VhEhOy2jmA-pRQAnEz-\ndeveloper-id: a54a55z8d24a84\ngrade: stable\nsnap-id: 52ad16a120d\nsnap-size: 16384\ntimestamp: 2025-03-11T18:35:06+01:00\nsign-key-sha3-384: C32BB4oh_vDRDA-AWo_wO3Bi74ayolOO6_YeA1uDp35eL7LVXbpn_85WZoRUGSrO\n\nAcLB123abc"),
			},
			expected: &proto.SnapBuildAssertionResponse{
				Type:            "snap-build",
				AuthorityId:     "a54a55z8d24a84",
				SnapSha3_384:    "IMGfaV_cMB_nAzrhAKt_v5IqMC-OuO4QlSN4mIKWsyBU6VhEhOy2jmA-pRQAnEz-",
				DeveloperId:     "a54a55z8d24a84",
				Grade:           "stable",
				SnapId:          "52ad16a120d",
				SnapSize:        "16384",
				Timestamp:       "2025-03-11T18:35:06+01:00",
				SignKeySha3_384: "C32BB4oh_vDRDA-AWo_wO3Bi74ayolOO6_YeA1uDp35eL7LVXbpn_85WZoRUGSrO",
				Errors:          []*proto.Error{},
			},
		},
		{
			name: "Wrong type",
			request: &proto.SnapBuildAssertionRequest{
				Assertion: []byte("type: sign-build\nauthority-id: a54a55z8d24a84\nsnap-sha3-384: IMGfaV_cMB_nAzrhAKt_v5IqMC-OuO4QlSN4mIKWsyBU6VhEhOy2jmA-pRQAnEz-\ndeveloper-id: a54a55z8d24a84\ngrade: stable\nsnap-id: 52ad16a120d\nsnap-size: 16384\ntimestamp: 2025-03-11T18:35:06+01:00\nsign-key-sha3-384: BEVVB4oh_vDRDA-AWo_wO3Bi74ayolOO6_YeA1uDp35eL7LVXbpn_85WZoRUGSrO"),
			},
			expected: &proto.SnapBuildAssertionResponse{
				Errors: []*proto.Error{
					{
						Code:    errors.Invalid,
						Message: "not a valid snap-build assertion: invalid type: sign-build",
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response, _ := service.ProcessSnapBuildAssertion(context.Background(), tt.request)
			assert.Equal(t, tt.expected, response)
		})
	}
}
