package server

import "testing"

func TestValidateSandboxIdleTimeout(t *testing.T) {
	got, err := validateSandboxIdleTimeout("45m")
	if err != nil {
		t.Fatalf("validateSandboxIdleTimeout failed: %v", err)
	}
	if got != "45m0s" {
		t.Fatalf("expected normalized duration %q, got %q", "45m0s", got)
	}
}

func TestValidateSandboxIdleTimeoutRejectsOutOfBounds(t *testing.T) {
	for _, value := range []string{"30s", "25h"} {
		t.Run(value, func(t *testing.T) {
			if _, err := validateSandboxIdleTimeout(value); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestValidateSandboxTTL(t *testing.T) {
	got, err := validateSandboxTTL("336h")
	if err != nil {
		t.Fatalf("validateSandboxTTL failed: %v", err)
	}
	if got != "336h0m0s" {
		t.Fatalf("expected normalized duration %q, got %q", "336h0m0s", got)
	}
}

func TestValidateSandboxTTLRejectsOutOfBounds(t *testing.T) {
	for _, value := range []string{"0s", "337h"} {
		t.Run(value, func(t *testing.T) {
			if _, err := validateSandboxTTL(value); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}
