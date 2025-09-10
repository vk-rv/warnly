//nolint:recvcheck // receiver name is consistent with the sql.Scanner interface
package warnly

import (
	"database/sql/driver"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/google/uuid"
)

// UUID represents a 16-byte universally unique identifier
// this type is a wrapper around google/uuid with the following differences
//   - type is a byte slice instead of [16]byte
//   - db serialization converts uuid to bytes as opposed to string
type UUID []byte

// ParseUUID returns a UUID parsed from the given string representation.
func ParseUUID(s string) (UUID, error) {
	if s == "" {
		return nil, errors.New("empty uuid string")
	}
	u, err := uuid.Parse(s)
	if err != nil {
		return nil, fmt.Errorf("parse uuid: %w", err)
	}
	return u[:], nil
}

// NewUUID generates a new UUID.
func NewUUID() UUID {
	u := uuid.New()
	return UUID(u[:])
}

// UUIDPtr simply returns a pointer for the given value type.
func UUIDPtr(u UUID) *UUID {
	return &u
}

// String returns the 36 byet hexstring representation of this uuid
// return empty string if this uuid is nil.
func (u UUID) String() string {
	if len(u) != 16 {
		return ""
	}
	var buf [36]byte
	u.encodeHex(buf[:])
	return string(buf[:])
}

// Scan implements sql.Scanner interface to allow this type to be
// parsed transparently by database drivers.
func (u *UUID) Scan(src any) error {
	if src == nil {
		return nil
	}
	*u = make([]byte, 16)
	switch src := src.(type) {
	case []byte:
		copy(*u, src)
	default:
		return fmt.Errorf("unsupported uuid type: %T", src)
	}
	return nil
}

// Value implements sql.Valuer so that UUIDs can be written to databases
// transparently. This method returns a byte slice representation of uuid.
func (u UUID) Value() (driver.Value, error) {
	return []byte(u), nil
}

// encodeHex encodes u into dst, which must be a byte slice of length 36.
func (u UUID) encodeHex(dst []byte) {
	hex.Encode(dst, u[:4])
	dst[8] = '-'
	hex.Encode(dst[9:13], u[4:6])
	dst[13] = '-'
	hex.Encode(dst[14:18], u[6:8])
	dst[18] = '-'
	hex.Encode(dst[19:23], u[8:10])
	dst[23] = '-'
	hex.Encode(dst[24:], u[10:])
}
