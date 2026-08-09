package server

import (
	"context"
	"testing"

	identityv1 "github.com/agynio/organizations/.gen/go/agynio/api/identity/v1"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type stubIdentityClient struct {
	identityv1.IdentityServiceClient
	identityType identityv1.IdentityType
	err          error
}

func (s *stubIdentityClient) GetIdentityType(context.Context, *identityv1.GetIdentityTypeRequest, ...grpc.CallOption) (*identityv1.GetIdentityTypeResponse, error) {
	if s.err != nil {
		return nil, s.err
	}
	return &identityv1.GetIdentityTypeResponse{IdentityType: s.identityType}, nil
}

// Membership is a roster of people, so only a user creating an organization is
// put in it. The platform admin identity holds the owner tuple instead, which is
// what grants access without adding a member the Console cannot name.
func TestCallerIsUser(t *testing.T) {
	cases := []struct {
		name   string
		client *stubIdentityClient
		want   bool
		errors bool
	}{
		{
			name:   "a user is a member",
			client: &stubIdentityClient{identityType: identityv1.IdentityType_IDENTITY_TYPE_USER},
			want:   true,
		},
		{
			name:   "the platform admin identity is not",
			client: &stubIdentityClient{identityType: identityv1.IdentityType_IDENTITY_TYPE_PLATFORM},
			want:   false,
		},
		{
			name:   "nor is a runner",
			client: &stubIdentityClient{identityType: identityv1.IdentityType_IDENTITY_TYPE_RUNNER},
			want:   false,
		},
		{
			// Every user is registered when their account is provisioned, well
			// before they can create anything, so an unregistered identity is
			// not one.
			name:   "an unregistered identity is not",
			client: &stubIdentityClient{err: status.Error(codes.NotFound, "identity not found")},
			want:   false,
		},
		{
			// Anything else is the identity service being unreachable, which
			// must not be read as "not a user" -- that would silently drop a
			// real owner out of their own organization.
			name:   "an unreachable identity service is an error",
			client: &stubIdentityClient{err: status.Error(codes.Unavailable, "connection refused")},
			errors: true,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			server := &Server{identityClient: testCase.client}
			got, err := server.callerIsUser(context.Background(), uuid.New())
			if testCase.errors {
				if err == nil {
					t.Fatal("expected an error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != testCase.want {
				t.Fatalf("expected %v, got %v", testCase.want, got)
			}
		})
	}
}
