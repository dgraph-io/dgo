/*
 * SPDX-FileCopyrightText: © 2017-2025 Istari Digital, Inc.
 * SPDX-License-Identifier: Apache-2.0
 */

package dgo

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestParseAbortReason(t *testing.T) {
	cases := map[string]AbortReason{
		"conflict: Transaction has been aborted. Please retry": AbortReasonConflict,
		"predicate-move: Transaction has been aborted. Please retry. Commits on predicate name are " +
			"blocked due to predicate move": AbortReasonPredicateMove,
		// The detail can contain colons of its own; only the first one delimits the category.
		"predicate-move: Transaction has been aborted. Please retry. Mutation done in group: 1. " +
			"Predicate name assigned to 2": AbortReasonPredicateMove,
		"stale-startts: Transaction has been aborted. Please retry. Its start timestamp is older " +
			"than ...": AbortReasonStaleStartTs,
		"  CONFLICT : detail": AbortReasonConflict,
		"conflict":            AbortReasonConflict,

		// A server without abort reasons, or a cause the server leaves uncategorized.
		"Transaction has been aborted. Please retry":                            AbortReasonUnknown,
		"Transaction has been aborted. Please retry. Tablet for name is nil":    AbortReasonUnknown,
		"Transaction has been aborted. Please retry. context deadline exceeded": AbortReasonUnknown,
		"": AbortReasonUnknown,
		// A category this client does not know about yet.
		"predicate-unavailable: something": AbortReasonUnknown,
	}
	for msg, want := range cases {
		require.Equal(t, want, ParseAbortReason(msg), "ParseAbortReason(%q)", msg)
	}
}

func TestAbortedError(t *testing.T) {
	const msg = "predicate-move: Transaction has been aborted. Please retry. Commits on predicate name " +
		"are blocked due to predicate move"
	err := fmt.Errorf("committing: %w", newAbortedError(msg))

	require.ErrorIs(t, err, ErrAborted)
	require.False(t, errors.Is(err, ErrFinished))

	var ae *AbortedError
	require.ErrorAs(t, err, &ae)
	require.Equal(t, AbortReasonPredicateMove, ae.Reason)
	require.Equal(t, msg, ae.Message)
	require.Equal(t, msg, ae.Error(), "Error returns the server's message verbatim")

	require.Equal(t, codes.Aborted, status.Code(err), "status.Code sees through the wrapping")
	require.Equal(t, msg, status.Convert(errors.Unwrap(err)).Message())

	// No message from the server falls back to the legacy text.
	require.Equal(t, ErrAborted.Error(), newAbortedError("").Error())
	require.Equal(t, AbortReasonUnknown, newAbortedError("").Reason)
}
