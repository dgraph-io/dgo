/*
 * SPDX-FileCopyrightText: © Hypermode Inc. <hello@hypermode.com>
 * SPDX-License-Identifier: Apache-2.0
 */

package dgo_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/dgraph-io/dgo/v240"
)

func TestDialCLoud(t *testing.T) {
	cases := []struct {
		endpoint string
		err      string
	}{
		{endpoint: "godly.grpc.region.aws.cloud.dgraph.io"},
		{endpoint: "godly.grpc.region.aws.cloud.dgraph.io:443"},
		{endpoint: "https://godly.region.aws.cloud.dgraph.io/graphql"},
		{endpoint: "godly.region.aws.cloud.dgraph.io"},
		{endpoint: "https://godly.region.aws.cloud.dgraph.io"},
		{endpoint: "random:url", err: "invalid port"},
		{endpoint: "google", err: "invalid URL"},
	}

	for _, tc := range cases {
		t.Run(tc.endpoint, func(t *testing.T) {
			_, err := dgo.DialCloud(tc.endpoint, "abc123")
			if tc.err == "" {
				require.NoError(t, err)
			} else {
				require.Contains(t, err.Error(), tc.err)
			}
		})
	}
}
