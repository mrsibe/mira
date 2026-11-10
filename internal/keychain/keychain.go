// Package keychain adapts MyGo's cross-platform credential store to the
// configuration package.
package keychain

import (
	"errors"

	"github.com/egoist/mygo"
)

// Store keeps secrets in the login Keychain, Secret Service, or Windows
// Credential Manager.
type Store struct{}

// Get returns a secret and whether it exists.
func (Store) Get(key string) (string, bool, error) {
	value, err := mygo.Secrets.Get(key)
	switch {
	case errors.Is(err, mygo.ErrSecretNotFound):
		return "", false, nil
	case err != nil:
		return "", false, err
	default:
		return string(value), true, nil
	}
}

// Set stores a secret.
func (Store) Set(key, value string) error {
	return mygo.Secrets.Set(key, []byte(value))
}

// Delete removes a secret. Deleting a missing secret is not an error.
func (Store) Delete(key string) error {
	return mygo.Secrets.Delete(key)
}
