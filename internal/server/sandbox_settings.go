package server

import (
	"fmt"
	"time"
)

const (
	defaultSandboxIdleTimeout = "30m"
	defaultSandboxTTL         = "72h"
	minSandboxIdleTimeout     = time.Minute
	maxSandboxIdleTimeout     = 24 * time.Hour
	maxSandboxTTL             = 14 * 24 * time.Hour
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
