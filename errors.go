package taptap

import (
	"errors"

	"connectrpc.com/connect"
	"github.com/google/uuid"
)

// errorsAs is a thin wrapper so the rest of the package doesn't need
// to import "errors" just for the standard As helper.
func errorsAs(err error, target any) bool {
	return errors.As(err, target)
}

// IsNotFound reports whether err is a Connect NotFound error.
func IsNotFound(err error) bool {
	return codeOf(err) == connect.CodeNotFound
}

// IsAlreadyExists reports whether err is a Connect AlreadyExists
// error — typically returned when an idempotency_key collides with a
// prior request whose payload differed.
func IsAlreadyExists(err error) bool {
	return codeOf(err) == connect.CodeAlreadyExists
}

// IsInvalidArgument reports whether err is a Connect InvalidArgument
// error — usually a validation failure surfaced from protovalidate.
func IsInvalidArgument(err error) bool {
	return codeOf(err) == connect.CodeInvalidArgument
}

// IsPermissionDenied reports whether err is a Connect PermissionDenied
// error — e.g. an API key that lacks scope for the requested resource.
func IsPermissionDenied(err error) bool {
	return codeOf(err) == connect.CodePermissionDenied
}

// IsUnauthenticated reports whether err is a Connect Unauthenticated
// error — typically a missing or invalid API key.
func IsUnauthenticated(err error) bool {
	return codeOf(err) == connect.CodeUnauthenticated
}

// IsRateLimited reports whether err is a Connect ResourceExhausted
// error — the SDK's retry layer will already have tried, so a surfaced
// error here means the limit held across the retry window.
func IsRateLimited(err error) bool {
	return codeOf(err) == connect.CodeResourceExhausted
}

// IsFailedPrecondition reports whether err is a Connect
// FailedPrecondition error — the request was well-formed but the
// resource is in the wrong state (e.g. refunding an already-refunded
// payment).
func IsFailedPrecondition(err error) bool {
	return codeOf(err) == connect.CodeFailedPrecondition
}

func codeOf(err error) connect.Code {
	var connectErr *connect.Error
	if !errors.As(err, &connectErr) {
		return 0
	}
	return connectErr.Code()
}

// NewIdempotencyKey returns a fresh UUID v4 suitable for use as an
// idempotency_key on state-changing requests. Persist this client-side
// before sending the request so a crash-restart can resend the same
// key and reach the same result.
func NewIdempotencyKey() string {
	return uuid.NewString()
}
