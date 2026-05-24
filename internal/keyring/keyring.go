// Package keyring wraps the OS keyring (macOS Keychain / Linux SecretService)
// so that API keys are never stored in plaintext config files.
package keyring

import (
	"errors"
	"fmt"
	"strings"

	gokeyring "github.com/zalando/go-keyring"
)

const (
	// ServicePrefix is the keyring service name prefix.
	ServicePrefix = "talon-sandbox"
	// RefPrefix is the prefix in config YAML api-key-ref values.
	RefPrefix = "keyring:"
)

// ErrNotFound is returned when a key is not present in the keyring.
var ErrNotFound = errors.New("keyring: key not found")

// Store is a thin wrapper around go-keyring.
type Store struct{}

// New creates a new keyring Store.
func New() *Store { return &Store{} }

// Set stores the API key under the given context name.
func (s *Store) Set(contextName, apiKey string) error {
	service := serviceName(contextName)
	if err := gokeyring.Set(service, "api-key", apiKey); err != nil {
		return fmt.Errorf("keyring: set %q: %w", service, err)
	}
	return nil
}

// Get retrieves the API key for the given context name.
func (s *Store) Get(contextName string) (string, error) {
	service := serviceName(contextName)
	val, err := gokeyring.Get(service, "api-key")
	if err != nil {
		if errors.Is(err, gokeyring.ErrNotFound) {
			return "", ErrNotFound
		}
		return "", fmt.Errorf("keyring: get %q: %w", service, err)
	}
	return val, nil
}

// Delete removes the API key for the given context name.
func (s *Store) Delete(contextName string) error {
	service := serviceName(contextName)
	if err := gokeyring.Delete(service, "api-key"); err != nil {
		if errors.Is(err, gokeyring.ErrNotFound) {
			return nil // idempotent
		}
		return fmt.Errorf("keyring: delete %q: %w", service, err)
	}
	return nil
}

// RefForContext returns the api-key-ref string for a context.
func RefForContext(contextName string) string {
	return RefPrefix + serviceName(contextName)
}

// ContextFromRef extracts the context name from an api-key-ref value.
// Returns "", false if the ref is not a keyring ref.
func ContextFromRef(ref string) (string, bool) {
	if !strings.HasPrefix(ref, RefPrefix) {
		return "", false
	}
	svc := strings.TrimPrefix(ref, RefPrefix)
	name := strings.TrimPrefix(svc, ServicePrefix+"-")
	return name, true
}

func serviceName(contextName string) string {
	return ServicePrefix + "-" + contextName
}
