package server

import (
	"fmt"
	"time"
)

const (
	defaultSandboxIdleTimeout = "30m"
	// The ceiling starts at the platform maximum; an organization narrows it.
	defaultSandboxMaxIdleTimeout = "24h"
	defaultSandboxTTL            = "72h"
	minSandboxIdleTimeout        = time.Minute
	maxSandboxIdleTimeout        = 24 * time.Hour
	maxSandboxTTL                = 14 * 24 * time.Hour
)

func validateSandboxIdleTimeout(value string) (string, error) {
	duration, err := time.ParseDuration(value)
	if err != nil {
		return "", err
	}
	if duration < minSandboxIdleTimeout || duration > maxSandboxIdleTimeout {
		return "", fmt.Errorf("must be between %s and %s", minSandboxIdleTimeout, maxSandboxIdleTimeout)
	}
	return duration.String(), nil
}

// resolveSandboxIdleBounds settles the pair after an update: whichever of the
// two the request names, over what is already stored. A default above the
// ceiling would hand every sandbox that names nothing a value the organization
// refuses to anyone who asks for it.
func resolveSandboxIdleBounds(currentDefault, currentMax string, requestedDefault, requestedMax *string) error {
	effectiveDefault := currentDefault
	if requestedDefault != nil {
		effectiveDefault = *requestedDefault
	}
	effectiveMax := currentMax
	if requestedMax != nil {
		effectiveMax = *requestedMax
	}
	defaultDuration, err := time.ParseDuration(effectiveDefault)
	if err != nil {
		return fmt.Errorf("sandbox_default_idle_timeout: %w", err)
	}
	maxDuration, err := time.ParseDuration(effectiveMax)
	if err != nil {
		return fmt.Errorf("sandbox_max_idle_timeout: %w", err)
	}
	if defaultDuration > maxDuration {
		return fmt.Errorf("sandbox_default_idle_timeout %s exceeds sandbox_max_idle_timeout %s", defaultDuration, maxDuration)
	}
	return nil
}

func validateSandboxTTL(value string) (string, error) {
	duration, err := time.ParseDuration(value)
	if err != nil {
		return "", err
	}
	if duration <= 0 || duration > maxSandboxTTL {
		return "", fmt.Errorf("must be greater than 0s and at most %s", maxSandboxTTL)
	}
	return duration.String(), nil
}
