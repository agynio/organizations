package server

import (
	"strings"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestSlugFromName(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{"Acme", "acme"},
		{"Acme Corp", "acme-corp"},
		{"  Acme   Corp  ", "acme-corp"},
		{"Acme, Inc.", "acme-inc"},
		{"ACME_CORP", "acme-corp"},
		{"...", "org"},
		{"", "org"},
		{strings.Repeat("a", 80), strings.Repeat("a", maxSlugLength)},
	}
	for _, test := range tests {
		if got := slugFromName(test.name); got != test.want {
			t.Errorf("slugFromName(%q) = %q, want %q", test.name, got, test.want)
		}
	}
}

// A truncated slug must not end on the separator the truncation created.
func TestSlugFromNameDoesNotEndOnASeparator(t *testing.T) {
	name := strings.Repeat("a", 63) + " tail"
	got := slugFromName(name)
	if strings.HasSuffix(got, "-") {
		t.Fatalf("slugFromName(%q) = %q, which ends on a separator", name, got)
	}
	if len(got) > maxSlugLength {
		t.Fatalf("slug %q is longer than %d", got, maxSlugLength)
	}
}

func TestValidateSlug(t *testing.T) {
	if _, err := validateSlug("acme-corp"); err != nil {
		t.Fatalf("validateSlug: %v", err)
	}
	for _, invalid := range []string{"", "   ", "Acme", "acme_corp", "acme corp", "acme.corp", strings.Repeat("a", 65)} {
		_, err := validateSlug(invalid)
		if err == nil {
			t.Fatalf("expected %q to be rejected", invalid)
		}
		if status.Code(err) != codes.InvalidArgument {
			t.Fatalf("code for %q = %v, want InvalidArgument", invalid, status.Code(err))
		}
	}
}

// The slug is a hostname label in an exposed port's address, so the values a
// hostname rejects have to be rejected here.
func TestValidateSlugRejectsWhatAHostnameLabelRejects(t *testing.T) {
	for _, slug := range []string{"-acme", "acme-", "-", strings.Repeat("a", 64)} {
		if _, err := validateSlug(slug); status.Code(err) != codes.InvalidArgument {
			t.Errorf("validateSlug(%q) = %v, want InvalidArgument", slug, err)
		}
	}
	for _, slug := range []string{"a", "acme", "acme-corp", "a1", strings.Repeat("a", 63)} {
		if _, err := validateSlug(slug); err != nil {
			t.Errorf("validateSlug(%q) = %v, want no error", slug, err)
		}
	}
}
