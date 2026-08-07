package server

import (
	"context"
	"testing"

	organizationsv1 "github.com/agynio/organizations/.gen/go/agynio/api/organizations/v1"
	"github.com/agynio/organizations/internal/store"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// The default is what a creator who has not thought about it gets; the ceiling
// is what the organization will pay for when someone has. A default above the
// ceiling would hand every unthinking creator a value the organization refuses
// to anyone who asks for it, so the pair is checked together whichever half the
// request names.
func boundsServer(t *testing.T, current store.Organization, onUpdate func(store.OrganizationUpdate)) *Server {
	t.Helper()
	authClient, _, cleanup := setupAuthClient(t, true)
	t.Cleanup(cleanup)
	return &Server{
		authorizationClient: authClient,
		getOrganization: func(context.Context, uuid.UUID) (store.Organization, error) {
			return current, nil
		},
		updateOrganization: func(_ context.Context, _ uuid.UUID, update store.OrganizationUpdate) (store.Organization, error) {
			if onUpdate != nil {
				onUpdate(update)
			}
			return current, nil
		},
	}
}

func storedBounds() store.Organization {
	return store.Organization{
		SandboxDefaultIdleTimeout: defaultSandboxIdleTimeout,
		SandboxMaxIdleTimeout:     defaultSandboxMaxIdleTimeout,
		SandboxDefaultTTL:         defaultSandboxTTL,
	}
}

func ownerContext() context.Context {
	return metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-identity-id", uuid.NewString()))
}

func TestUpdateOrganizationSetsTheIdleTimeoutCeiling(t *testing.T) {
	var seen store.OrganizationUpdate
	server := boundsServer(t, storedBounds(), func(update store.OrganizationUpdate) { seen = update })

	value := "4h"
	if _, err := server.UpdateOrganization(ownerContext(), &organizationsv1.UpdateOrganizationRequest{
		Id:                    uuid.NewString(),
		SandboxMaxIdleTimeout: &value,
	}); err != nil {
		t.Fatalf("UpdateOrganization returned error: %v", err)
	}
	if seen.SandboxMaxIdleTimeout == nil || *seen.SandboxMaxIdleTimeout != "4h0m0s" {
		t.Fatalf("unexpected ceiling update: %v", seen.SandboxMaxIdleTimeout)
	}
}

// Lowering the ceiling under a default that is already stored is the case a
// single field would have hidden.
func TestUpdateOrganizationRejectsACeilingBelowTheStoredDefault(t *testing.T) {
	current := storedBounds()
	current.SandboxDefaultIdleTimeout = "2h0m0s"
	server := boundsServer(t, current, func(store.OrganizationUpdate) {
		t.Fatal("updateOrganization should not be called when the pair is inconsistent")
	})

	value := "1h"
	_, err := server.UpdateOrganization(ownerContext(), &organizationsv1.UpdateOrganizationRequest{
		Id:                    uuid.NewString(),
		SandboxMaxIdleTimeout: &value,
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument, got %v", err)
	}
}

func TestUpdateOrganizationRejectsADefaultAboveTheStoredCeiling(t *testing.T) {
	current := storedBounds()
	current.SandboxMaxIdleTimeout = "2h0m0s"
	server := boundsServer(t, current, func(store.OrganizationUpdate) {
		t.Fatal("updateOrganization should not be called when the pair is inconsistent")
	})

	value := "3h"
	_, err := server.UpdateOrganization(ownerContext(), &organizationsv1.UpdateOrganizationRequest{
		Id:                        uuid.NewString(),
		SandboxDefaultIdleTimeout: &value,
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument, got %v", err)
	}
}

// Both halves moving together is consistent even though each alone would not be.
func TestUpdateOrganizationAcceptsBothBoundsRaisedTogether(t *testing.T) {
	current := storedBounds()
	current.SandboxMaxIdleTimeout = "1h0m0s"
	var seen store.OrganizationUpdate
	server := boundsServer(t, current, func(update store.OrganizationUpdate) { seen = update })

	newDefault := "3h"
	newMax := "6h"
	if _, err := server.UpdateOrganization(ownerContext(), &organizationsv1.UpdateOrganizationRequest{
		Id:                        uuid.NewString(),
		SandboxDefaultIdleTimeout: &newDefault,
		SandboxMaxIdleTimeout:     &newMax,
	}); err != nil {
		t.Fatalf("UpdateOrganization returned error: %v", err)
	}
	if seen.SandboxDefaultIdleTimeout == nil || *seen.SandboxDefaultIdleTimeout != "3h0m0s" {
		t.Fatalf("unexpected default update: %v", seen.SandboxDefaultIdleTimeout)
	}
	if seen.SandboxMaxIdleTimeout == nil || *seen.SandboxMaxIdleTimeout != "6h0m0s" {
		t.Fatalf("unexpected ceiling update: %v", seen.SandboxMaxIdleTimeout)
	}
}
