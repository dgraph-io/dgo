package dgo

import (
	"strings"
	"time"
)

// IsRetryable determines if an error is retryable. Replace this with your own logic.
func IsRetryable(err error) bool {
	return strings.Contains(err.Error(), "504 (Gateway Timeout)")
}

const (
	MaxAttempts   = 5
	InitialBackoff = 100 * time.Millisecond
)

// RetryWithExponentialBackoff retries the given operation with exponential backoff.
// Uses package-level defaults for isRetryable, maxAttempts, and initialBackoff.
// The operation should return a value of type T and an error.
func RetryWithExponentialBackoff[T any](op func() (T, error)) (T, error) {
	var zero T
	backoff := InitialBackoff
	for attempt := 1; attempt <= MaxAttempts; attempt++ {
		result, err := op()
		if err == nil {
			return result, nil
		}
		if !IsRetryable(err) {
			return zero, err
		}
		if attempt < MaxAttempts {
			time.Sleep(backoff)
			backoff *= 2
		}
	}
	return op() // last attempt
}
