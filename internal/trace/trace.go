package trace

import (
	"context"
	"crypto/rand"
	"encoding/hex"
)

type ctxKey struct{}

// NewContext stores the trace id in the context so every later layer can read it.
func NewContext(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, ctxKey{}, traceID)
}

// FromContext returns the trace id, or an empty string if there is none.
func FromContext(ctx context.Context) string {
	id, _ := ctx.Value(ctxKey{}).(string)
	return id
}

// traceIDBytes is the number of random bytes in a generated trace id.
const traceIDBytes = 16

// IDLength is the length of ids from NewID: each byte encodes to 2 hex characters.
const IDLength = traceIDBytes * 2

// NewID returns a random trace id of IDLength lowercase hex characters.
func NewID() string {
	var b [traceIDBytes]byte
	_, _ = rand.Read(b[:]) // crypto/rand.Read never returns an error
	return hex.EncodeToString(b[:])
}

const maxIDLength = 128

// IsValidID reports whether a trace id received from outside (e.g. a request header) may be reused:
// it must be non-empty, at most 128 characters, and contain only letters, digits, '-', '_' or '.' (UUIDs and hex ids qualify).
// The id is written verbatim to logs and response headers, so restricting it prevents forged log lines (via newlines) and huge values.
func IsValidID(id string) bool {
	if id == "" || len(id) > maxIDLength {
		return false
	}
	for _, c := range id {
		if !('a' <= c && c <= 'z' || 'A' <= c && c <= 'Z' || '0' <= c && c <= '9' || c == '-' || c == '_' || c == '.') {
			return false
		}
	}
	return true
}
