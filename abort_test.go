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
)

func TestParseAbortReason(t *testing.T) {
	cases := map[string]AbortReason{
		"conflict: Transaction has been aborted. Please retry":                        AbortReasonConflict,
		"predicate-move: Commits on predicate name are blocked due to predicate move": AbortReasonPredicateMove,
		// The detail can contain colons of its own; only the first one delimits the category.
		"predicate-move: Mutation done in group: 1. Predicate name assigned to 2": AbortReasonPredicateMove,
		"stale-startts: Transaction start timestamp is older than ...":            AbortReasonStaleStartTs,
		"  CONFLICT : detail": AbortReasonConflict,
		"conflict":            AbortReasonConflict,

		// A server without abort reasons, or a cause the server leaves uncategorized.
		"Transaction has been aborted. Please retry": AbortReasonUnknown,
		"Tablet for name is nil":                     AbortReasonUnknown,
		"context deadline exceeded":                  AbortReasonUnknown,
		"":                                           AbortReasonUnknown,
		// A category this client does not know about yet.
		"predicate-unavailable: something": AbortReasonUnknown,
	}
	for msg, want := range cases {
		require.Equal(t, want, ParseAbortReason(msg), "ParseAbortReason(%q)", msg)
	}
}

func TestAbortedError(t *testing.T) {
	const msg = "predicate-move: Commits on predicate name are blocked due to predicate move"
	var err error = fmt.Errorf("committing: %w", newAbortedError(msg))

	require.ErrorIs(t, err, ErrAborted)
	require.False(t, errors.Is(err, ErrFinished))

	var ae *AbortedError
	require.ErrorAs(t, err, &ae)
	require.Equal(t, AbortReasonPredicateMove, ae.Reason)
	require.Equal(t, msg, ae.Message)
	require.Equal(t, msg, ae.Error(), "Error returns the server's message verbatim")

	// No message from the server falls back to the legacy text.
	require.Equal(t, ErrAborted.Error(), newAbortedError("").Error())
	require.Equal(t, AbortReasonUnknown, newAbortedError("").Reason)
}
