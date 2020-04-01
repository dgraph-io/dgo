/*
 * Copyright (C) 2019 Dgraph Labs, Inc. and Contributors
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
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/dgraph-io/dgo/v200"
	"github.com/dgraph-io/dgo/v200/protos/api"
)

func TestTxnErrFinished(t *testing.T) {
	dg, cancel := getDgraphClient()
	defer cancel()

	ctx := context.Background()
	err := dg.Alter(ctx, &api.Operation{DropAll: true})
	require.NoError(t, err)

	op := &api.Operation{}
	op.Schema = `email: string @index(exact) .`
	err = dg.Alter(ctx, op)
	require.NoError(t, err)

	mu := &api.Mutation{SetNquads: []byte(`_:user1 <email> "user1@company1.io" .`), CommitNow: true}
	txn := dg.NewTxn()
	_, err = txn.Mutate(context.Background(), mu)
	require.NoError(t, err, "first mutation should be successful")

	// Run the mutation again on same transaction.
	_, err = txn.Mutate(context.Background(), mu)
	require.Equal(t, err, dgo.ErrFinished, "should have returned ErrFinished")
}

func TestTxnErrReadOnly(t *testing.T) {
	dg, cancel := getDgraphClient()
	defer cancel()

	ctx := context.Background()
	err := dg.Alter(ctx, &api.Operation{DropAll: true})
	require.NoError(t, err)

	op := &api.Operation{}
	op.Schema = `email: string @index(exact) .`
	err = dg.Alter(ctx, op)
	require.NoError(t, err)

	mu := &api.Mutation{SetNquads: []byte(`_:user1 <email> "user1@company1.io" .`)}

	// Run mutation on ReadOnly transaction.
	_, err = dg.NewReadOnlyTxn().Mutate(context.Background(), mu)
	require.Equal(t, err, dgo.ErrReadOnly)
}

func TestTxnErrAborted(t *testing.T) {
	dg, cancel := getDgraphClient()
	defer cancel()

	ctx := context.Background()
	err := dg.Alter(ctx, &api.Operation{DropAll: true})
	require.NoError(t, err)

	op := &api.Operation{}
	op.Schema = `email: string @index(exact) .`
	err = dg.Alter(ctx, op)
	require.NoError(t, err)

	mu1 := &api.Mutation{
		SetNquads: []byte(`_:user1 <email> "user1@company1.io" .`),
		CommitNow: true,
	}

	// Insert first record.
	_, err = dg.NewTxn().Mutate(context.Background(), mu1)
	require.NoError(t, err, "first mutation failed")

	q := `{
		v as var(func: eq(email, "user1@company1.io"))
	}
	`
	mu2 := &api.Mutation{
		SetNquads: []byte(`uid(v) <email> "updated1@company1.io" .`),
	}

	// Run same mutation using two transactions.
	txn1 := dg.NewTxn()
	txn2 := dg.NewTxn()

	req := &api.Request{Query: q, Mutations: []*api.Mutation{mu2}}
	ctx1, ctx2 := context.Background(), context.Background()
	_, err1 := txn1.Do(ctx1, req)
	_, err2 := txn2.Do(ctx2, req)

	require.NoError(t, err1)
	require.NoError(t, err2)

	err = txn1.Commit(ctx1)
	require.Error(t, txn2.Commit(ctx2), dgo.ErrAborted, "2nd transaction should have aborted")
}
