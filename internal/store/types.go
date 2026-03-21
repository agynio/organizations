package store

import (
	"time"

	"github.com/google/uuid"
)

type Organization struct {
	ID        uuid.UUID
	Name      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type OrganizationInput struct {
	Name string
}

type OrganizationUpdate struct {
	Name *string
}

type OrganizationFilter struct{}

type PageCursor struct {
	AfterID uuid.UUID
}

type OrganizationListResult struct {
	Organizations []Organization
	NextCursor *PageCursor
}
