/*
 * SPDX-FileCopyrightText: © Hypermode Inc. <hello@hypermode.com>
 * SPDX-License-Identifier: Apache-2.0
 */

package dgo_test

import (
	"testing"

	"github.com/dgraph-io/dgo/v240"

	"github.com/stretchr/testify/require"
)

func TestOpen(t *testing.T) {
	var err error

	_, err = dgo.Open("127.0.0.1:9180")
	require.ErrorContains(t, err, "first path segment in URL cannot contain colon")

	_, err = dgo.Open("localhost:9180")
	require.ErrorContains(t, err, "invalid scheme: must start with dgraph://")

	_, err = dgo.Open("dgraph://localhost:9180")
	require.NoError(t, err)

	_, err = dgo.Open("dgraph://localhost")
	require.ErrorContains(t, err, "invalid connection string: host url must have both host and port")

	_, err = dgo.Open("dgraph://localhost:")
	require.NoError(t, err)

	_, err = dgo.Open("dgraph://localhost:9180?sslmode=verify-ca")
	require.NoError(t, err)

	_, err = dgo.Open("dgraph://localhost:9180?sslmode=prefer")
	require.ErrorContains(t, err, "invalid SSL mode: prefer (must be one of disable, require, verify-ca)")

	_, err = dgo.Open("dgraph://localhost:9180?sslmode=disable&bearertoken=abc")
	require.ErrorContains(t, err, "grpc: the credentials require transport level security")

	_, err = dgo.Open("dgraph://localhost:9180?sslmode=disable&apikey=abc")
	require.ErrorContains(t, err, "grpc: the credentials require transport level security")

	_, err = dgo.Open("dgraph://localhost:9180?sslmode=disable&apikey=abc&bearertoken=bgf")
	require.ErrorContains(t, err, "invalid connection string: both apikey and bearertoken cannot be provided")

	_, err = dgo.Open("dgraph://localhost:9180?sslmode=verify-ca&bearertoken=hfs")
	require.NoError(t, err)

	_, err = dgo.Open("dgraph://localhost:9180?sslmode=verify-ca&apikey=hfs")
	require.NoError(t, err)

	_, err = dgo.Open("dgraph://localhost:9180?sslmode=require&bearertoken=hfs")
	require.NoError(t, err)

	_, err = dgo.Open("dgraph://localhost:9180?sslmode=require&apikey=hfs")
	require.NoError(t, err)

	_, err = dgo.Open("dgraph://localhost:9180?sslm")
	require.NoError(t, err)
}
