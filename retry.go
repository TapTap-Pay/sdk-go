package taptap

import (
	"context"
	"math/rand/v2"
	"time"

	"connectrpc.com/connect"
)

// retryInterceptor retries unary calls on transient errors with
// exponential backoff + jitter. Streaming calls are passed through
// unretried — Connect bidi/server streams aren't safely replayable
// without application-level checkpointing.
//
// Retried Connect codes: Unavailable, DeadlineExceeded,
// ResourceExhausted. Network-level errors (no code surfaced) are also
// retried since they're effectively the same class.
//
// Idempotency: state-changing RPCs in the TapTap-Pay API take
// idempotency_key as a message field; the same proto request is sent
// on every attempt, so the server dedupes naturally as long as the
// caller set the key. Read-only RPCs are always safe to retry.
type retryInterceptor struct {
	max       int
	baseDelay time.Duration
}

func newRetryInterceptor(max int, baseDelay time.Duration) connect.Interceptor {
	return &retryInterceptor{max: max, baseDelay: baseDelay}
}

func (i *retryInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		var lastErr error
		for attempt := 0; attempt <= i.max; attempt++ {
			resp, err := next(ctx, req)
			if err == nil {
				return resp, nil
			}
			if !isRetryable(err) {
				return nil, err
			}
			lastErr = err
			if attempt == i.max {
				break
			}
			delay := backoff(i.baseDelay, attempt)
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
			}
		}
		return nil, lastErr
	}
}

func (i *retryInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return next
}

func (i *retryInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return next
}

func isRetryable(err error) bool {
	var connectErr *connect.Error
	if !errorsAs(err, &connectErr) {
		// Non-Connect error (likely a transport-level failure) — retry.
		return true
	}
	switch connectErr.Code() {
	case connect.CodeUnavailable, connect.CodeDeadlineExceeded, connect.CodeResourceExhausted:
		return true
	default:
		return false
	}
}

// backoff returns baseDelay * 2^attempt with full jitter (random in
// [0, computed]).
func backoff(base time.Duration, attempt int) time.Duration {
	max := base << attempt
	if max <= 0 {
		max = base
	}
	return time.Duration(rand.Int64N(int64(max)))
}
