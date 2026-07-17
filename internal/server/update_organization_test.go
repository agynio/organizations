package server

import (
	"context"
	"testing"
	"time"

	organizationsv1 "github.com/agynio/organizations/.gen/go/agynio/api/organizations/v1"
	"github.com/agynio/organizations/internal/store"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestUpdateOrganizationUpdatesSandboxSettings(t *testing.T) {
	authClient, authServer, cleanup := setupAuthClient(t, true)
	defer cleanup()

	identityID := uuid.New()
	organizationID := uuid.New()
	idleTimeout := "45m"
	ttl := "120h"
	updated := store.Organization{
		ID:                        organizationID,
		Name:                      "Acme Corp",
		SandboxDefaultIdleTimeout: "45m0s",
		SandboxDefaultTTL:         "120h0m0s",
		CreatedAt:                 time.Now().UTC(),
		UpdatedAt:                 time.Now().UTC(),
	}
	called := false

	server := &Server{
		authorizationClient: authClient,
		updateOrganization: func(ctx context.Context, id uuid.UUID, update store.OrganizationUpdate) (store.Organization, error) {
			called = true
			if id != organizationID {
				t.Fatalf("expected organization id %s, got %s", organizationID, id)
			}
			if update.SandboxDefaultIdleTimeout == nil || *update.SandboxDefaultIdleTimeout != "45m0s" {
				t.Fatalf("unexpected sandbox idle timeout update: %v", update.SandboxDefaultIdleTimeout)
			}
			if update.SandboxDefaultTTL == nil || *update.SandboxDefaultTTL != "120h0m0s" {
				t.Fatalf("unexpected sandbox ttl update: %v", update.SandboxDefaultTTL)
			}
			return updated, nil
		},
	}

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-identity-id", identityID.String()))
	response, err := server.UpdateOrganization(ctx, &organizationsv1.UpdateOrganizationRequest{
		Id:                        organizationID.String(),
		SandboxDefaultIdleTimeout: &idleTimeout,
		SandboxDefaultTtl:         &ttl,
	})
	if err != nil {
		t.Fatalf("UpdateOrganization returned error: %v", err)
	}
	if !called {
		t.Fatal("expected updateOrganization to be called")
	}
	organization := response.GetOrganization()
	if organization.GetSandboxDefaultIdleTimeout() != "45m0s" {
		t.Fatalf("expected idle timeout %q, got %q", "45m0s", organization.GetSandboxDefaultIdleTimeout())
	}
	if organization.GetSandboxDefaultTtl() != "120h0m0s" {
		t.Fatalf("expected ttl %q, got %q", "120h0m0s", organization.GetSandboxDefaultTtl())
	}

	authServer.requestLock.Lock()
	request := authServer.lastRequest
	authServer.requestLock.Unlock()
	if request == nil || request.GetTupleKey() == nil {
		t.Fatal("expected authorization check request")
	}
	if request.GetTupleKey().GetRelation() != "owner" {
		t.Fatalf("expected owner relation, got %s", request.GetTupleKey().GetRelation())
	}
	if request.GetTupleKey().GetObject() != organizationObjectPrefix+organizationID.String() {
		t.Fatalf("expected organization object %s, got %s", organizationObjectPrefix+organizationID.String(), request.GetTupleKey().GetObject())
	}
}

func TestUpdateOrganizationRejectsInvalidSandboxSettings(t *testing.T) {
	authClient, _, cleanup := setupAuthClient(t, true)
	defer cleanup()

	organizationID := uuid.New()
	idleTimeout := "30s"
	server := &Server{
		authorizationClient: authClient,
		updateOrganization: func(ctx context.Context, id uuid.UUID, update store.OrganizationUpdate) (store.Organization, error) {
			t.Fatal("updateOrganization should not be called for invalid sandbox settings")
			return store.Organization{}, nil
		},
	}

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-identity-id", uuid.NewString()))
	_, err := server.UpdateOrganization(ctx, &organizationsv1.UpdateOrganizationRequest{
		Id:                        organizationID.String(),
		SandboxDefaultIdleTimeout: &idleTimeout,
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument, got %v", err)
	}
}

func TestUpdateOrganizationNameOnlyRequiresOwner(t *testing.T) {
	authClient, authServer, cleanup := setupAuthClient(t, true)
	defer cleanup()

	identityID := uuid.New()
	organizationID := uuid.New()
	name := "Acme Updated"
	updated := store.Organization{
		ID:                        organizationID,
		Name:                      name,
		SandboxDefaultIdleTimeout: defaultSandboxIdleTimeout,
		SandboxDefaultTTL:         defaultSandboxTTL,
		CreatedAt:                 time.Now().UTC(),
		UpdatedAt:                 time.Now().UTC(),
	}
	called := false

	server := &Server{
		authorizationClient: authClient,
		updateOrganization: func(ctx context.Context, id uuid.UUID, update store.OrganizationUpdate) (store.Organization, error) {
			called = true
			if id != organizationID {
				t.Fatalf("expected organization id %s, got %s", organizationID, id)
			}
			if update.Name == nil || *update.Name != name {
				t.Fatalf("unexpected name update: %v", update.Name)
			}
			if update.SandboxDefaultIdleTimeout != nil || update.SandboxDefaultTTL != nil {
				t.Fatalf("unexpected sandbox settings update: %#v", update)
			}
			return updated, nil
		},
	}

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-identity-id", identityID.String()))
	response, err := server.UpdateOrganization(ctx, &organizationsv1.UpdateOrganizationRequest{
		Id:   organizationID.String(),
		Name: &name,
	})
	if err != nil {
		t.Fatalf("UpdateOrganization returned error: %v", err)
	}
	if !called {
		t.Fatal("expected updateOrganization to be called")
	}
	if response.GetOrganization().GetName() != name {
		t.Fatalf("expected name %q, got %q", name, response.GetOrganization().GetName())
	}

	authServer.requestLock.Lock()
	request := authServer.lastRequest
	authServer.requestLock.Unlock()
	if request == nil || request.GetTupleKey() == nil {
		t.Fatal("expected authorization check request")
	}
	if request.GetTupleKey().GetRelation() != "owner" {
		t.Fatalf("expected owner relation, got %s", request.GetTupleKey().GetRelation())
	}
}

func TestUpdateOrganizationNameOnlyRequiresIdentity(t *testing.T) {
	organizationID := uuid.New()
	name := "Acme Updated"
	server := &Server{
		updateOrganization: func(ctx context.Context, id uuid.UUID, update store.OrganizationUpdate) (store.Organization, error) {
			t.Fatal("updateOrganization should not be called without identity")
			return store.Organization{}, nil
		},
	}

	_, err := server.UpdateOrganization(context.Background(), &organizationsv1.UpdateOrganizationRequest{
		Id:   organizationID.String(),
		Name: &name,
	})
	if status.Code(err) != codes.Unauthenticated {
		t.Fatalf("expected Unauthenticated, got %v", err)
	}
}

func TestUpdateOrganizationRequiresOwner(t *testing.T) {
	authClient, _, cleanup := setupAuthClient(t, false)
	defer cleanup()

	organizationID := uuid.New()
	idleTimeout := "45m"
	server := &Server{
		authorizationClient: authClient,
		updateOrganization: func(ctx context.Context, id uuid.UUID, update store.OrganizationUpdate) (store.Organization, error) {
			t.Fatal("updateOrganization should not be called without owner permission")
			return store.Organization{}, nil
		},
	}

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-identity-id", uuid.NewString()))
	_, err := server.UpdateOrganization(ctx, &organizationsv1.UpdateOrganizationRequest{
		Id:                        organizationID.String(),
		SandboxDefaultIdleTimeout: &idleTimeout,
	})
	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("expected PermissionDenied, got %v", err)
	}
}
