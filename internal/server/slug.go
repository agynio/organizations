package server

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/agynio/organizations/internal/store"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const maxSlugLength = 64

var (
	slugPattern      = regexp.MustCompile(`^[a-z0-9-]+$`)
	slugSeparators   = regexp.MustCompile(`[^a-z0-9]+`)
	slugTrimmedDashs = regexp.MustCompile(`^-+|-+$`)
)

func validateSlug(value string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", status.Error(codes.InvalidArgument, "slug: value is empty")
	}
	if len(trimmed) > maxSlugLength {
		return "", status.Errorf(codes.InvalidArgument, "slug: must be at most %d characters", maxSlugLength)
	}
	if !slugPattern.MatchString(trimmed) {
		return "", status.Error(codes.InvalidArgument, "slug: must match ^[a-z0-9-]+$")
	}
	return trimmed, nil
}

// slugFromName derives a slug from a display name. Creating an organization
// without naming a slug has to keep working: the field is new, and a client
// built against the previous API sends none.
func slugFromName(name string) string {
	derived := slugTrimmedDashs.ReplaceAllString(slugSeparators.ReplaceAllString(strings.ToLower(name), "-"), "")
	if len(derived) > maxSlugLength {
		derived = strings.TrimRight(derived[:maxSlugLength], "-")
	}
	if derived == "" {
		return "org"
	}
	return derived
}

// resolveSlug settles the slug a new organization gets. A slug the caller named
// is taken as written and collides loudly; a derived one is disambiguated,
// because the caller did not choose it and has nothing to correct.
func (s *Server) resolveSlug(ctx context.Context, requested, name string) (string, error) {
	if strings.TrimSpace(requested) != "" {
		slug, err := validateSlug(requested)
		if err != nil {
			return "", err
		}
		return slug, nil
	}

	base := slugFromName(name)
	candidate := base
	for attempt := 2; attempt < 100; attempt++ {
		taken, err := s.slugTaken(ctx, candidate)
		if err != nil {
			return "", err
		}
		if !taken {
			return candidate, nil
		}
		suffix := fmt.Sprintf("-%d", attempt)
		trimmed := base
		if len(trimmed)+len(suffix) > maxSlugLength {
			trimmed = strings.TrimRight(base[:maxSlugLength-len(suffix)], "-")
		}
		candidate = trimmed + suffix
	}
	return "", status.Error(codes.AlreadyExists, "slug: could not derive an unused slug from the name")
}

func (s *Server) slugTaken(ctx context.Context, slug string) (bool, error) {
	_, err := s.store.GetOrganizationBySlug(ctx, slug)
	if err == nil {
		return true, nil
	}
	var notFound *store.NotFoundError
	if errors.As(err, &notFound) {
		return false, nil
	}
	return false, toStatusError(err)
}
