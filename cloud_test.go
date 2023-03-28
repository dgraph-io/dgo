/*
 * Copyright (C) 2023 Dgraph Labs, Inc. and Contributors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package dgo_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/dgraph-io/dgo/v230"
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
