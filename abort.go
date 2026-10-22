/*
 * SPDX-FileCopyrightText: © 2017-2025 Istari Digital, Inc.
 * SPDX-License-Identifier: Apache-2.0
 */

package dgo

import (
	"strings"
)

// AbortReason is the category of a transaction abort, as reported by the Dgraph server.
//
// The server encodes the category as a "<code>: <detail>" prefix on the gRPC Aborted status
// message. Categories are matched on that prefix, and an absent or unrecognized prefix yields
// AbortReasonUnknown, so a newer server may introduce categories this client does not name without
// breaking it.
type AbortReason string

const (
	// AbortReasonConflict is a write-write conflict with another concurrent transaction; retry with
	// a fresh transaction. The server cannot say which key collided: conflict keys are one-way
	// fingerprints by the time they are compared, so the culprit may be the data key written
	// directly, or an index or count key derived from it. On an @upsert predicate the uid is
	// excluded from the comparison, so any two transactions writing the same value conflict.
	AbortReasonConflict AbortReason = "conflict"
	// AbortReasonPredicateMove means a predicate the transaction touched is being moved between
	// groups and commits on it are blocked, or it finished moving while the transaction was open.
	// Back off, then retry with a fresh transaction once the move completes.
	AbortReasonPredicateMove AbortReason = "predicate-move"
	// AbortReasonStaleStartTs means the transaction's start timestamp is older than the oldest
	// timestamp the server can still validate against. That happens after a Zero leader change, and
	// also when Zero trims its conflict map at a snapshot. Retry with a fresh transaction.
	AbortReasonStaleStartTs AbortReason = "stale-startts"
	// AbortReasonUnknown means no category was reported. This covers aborts from older servers,
	// which do not categorize at all, and aborts a current server declines to categorize because no
	// published category fits: for example a transaction already aborted by a schema change or the
	// idle-transaction reaper, a cancelled request, or a predicate no group currently serves. The
	// error message still explains what happened.
	AbortReasonUnknown AbortReason = "unknown"
)

// AbortedError is returned when the server aborts a transaction. It wraps ErrAborted, so
// errors.Is(err, ErrAborted) reports true; use errors.As to read the reason:
//
//	var ae *dgo.AbortedError
//	if errors.As(err, &ae) && ae.Reason == dgo.AbortReasonPredicateMove {
//		// back off before retrying
//	}
//
// Comparing with == (err == dgo.ErrAborted) does not match an AbortedError; use errors.Is.
type AbortedError struct {
	// Reason is the parsed abort category.
	Reason AbortReason
	// Message is the server's description of the abort, including any "<code>: " prefix.
	Message string
}

func (e *AbortedError) Error() string {
	if e.Message == "" {
		return ErrAborted.Error()
	}
	return e.Message
}

// Unwrap returns ErrAborted, so that errors.Is(err, ErrAborted) holds for every abort.
func (e *AbortedError) Unwrap() error {
	return ErrAborted
}

// newAbortedError builds the error returned for a server-side abort from the gRPC status message.
func newAbortedError(msg string) *AbortedError {
	return &AbortedError{Reason: ParseAbortReason(msg), Message: msg}
}

// ParseAbortReason returns the abort category encoded in a server abort message. Only the first
// colon delimits the category, since several server messages contain colons in the detail that
// follows. Matching is case-insensitive and ignores surrounding whitespace. A message with no
// recognized prefix returns AbortReasonUnknown.
func ParseAbortReason(msg string) AbortReason {
	code, _, _ := strings.Cut(msg, ":")
	switch r := AbortReason(strings.ToLower(strings.TrimSpace(code))); r {
	case AbortReasonConflict, AbortReasonPredicateMove, AbortReasonStaleStartTs:
		return r
	default:
		return AbortReasonUnknown
	}
}
