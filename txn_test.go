/*
 * Copyright 2019 Dgraph Labs, Inc.
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

package dgo

import (
	"context"
	"testing"

	"github.com/dgraph-io/dgo/protos/api"
	"github.com/dgraph-io/dgo/y"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// mockClient will mock a DgraphClient
type mockClient struct {
	abort bool
}

func (c *mockClient) Login(ctx context.Context, in *api.LoginRequest, opts ...grpc.CallOption) (*api.Response, error) {
	return nil, nil
}
func (c *mockClient) Query(ctx context.Context, in *api.Request, opts ...grpc.CallOption) (*api.Response, error) {
	return nil, nil
}
func (c *mockClient) Mutate(ctx context.Context, in *api.Mutation, opts ...grpc.CallOption) (*api.Assigned, error) {
	if c.abort {
		return nil, status.Errorf(codes.Aborted, `Mutation aborted`)
	}
	return nil, nil
}
func (c *mockClient) Alter(ctx context.Context, in *api.Operation, opts ...grpc.CallOption) (*api.Payload, error) {
	return nil, nil
}
func (c *mockClient) CommitOrAbort(ctx context.Context, in *api.TxnContext, opts ...grpc.CallOption) (*api.TxnContext, error) {
	return nil, nil
}
func (c *mockClient) CheckVersion(ctx context.Context, in *api.Check, opts ...grpc.CallOption) (*api.Version, error) {
	return nil, nil
}

func TestMutationErrAborted(t *testing.T) {
	ctx := context.Background()
	dg := NewDgraphClient(&mockClient{abort: true})
	txn := dg.NewTxn()
	_, err := txn.Mutate(ctx, &api.Mutation{})
	require.EqualError(t, err, y.ErrAborted.Error())
}
