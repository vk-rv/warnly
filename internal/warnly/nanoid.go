package warnly

import (
	"fmt"
	"strings"

	nanoid "github.com/matoous/go-nanoid/v2"
)

const (
	alphabet = "0123456789abcdefghijklmnopqrstuvwxyz"
	length   = 7
)

// NewNanoID generates a new random ID.
// It returns an error if the ID generation fails.
func NewNanoID() (string, error) { return nanoid.Generate(alphabet, length) }

// MustNanoID is the same as New, but panics on error.
func MustNanoID() string { return nanoid.MustGenerate(alphabet, length) }

// ValidateNanoID validates a given ID.
func ValidateNanoID(fieldName, id string) error {
	if id == "" {
		return fmt.Errorf("%s cannot be blank", fieldName)
	}

	if len(id) != length {
		return fmt.Errorf("%s should be %d characters long", fieldName, length)
	}

	if strings.Trim(id, alphabet) != "" {
		return fmt.Errorf("%s has invalid characters", fieldName)
	}

	return nil
}
