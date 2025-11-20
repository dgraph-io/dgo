/*
 * SPDX-FileCopyrightText: © 2017-2025 Istari Digital, Inc.
 * SPDX-License-Identifier: Apache-2.0
 */

package dgo_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/dgraph-io/dgo/v250"
	"github.com/dgraph-io/dgo/v250/protos/api"
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

	mu := &api.Mutation{
		SetNquads: []byte(`_:user1 <email> "user1@company1.io" .`),
		CommitNow: true,
	}
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
	_, err = txn1.Do(ctx1, req)
	require.NoError(t, err)
	_, err = txn2.Do(ctx2, req)
	require.NoError(t, err)

	err = txn1.Commit(ctx1)
	require.NoError(t, err)
	require.Error(t, txn2.Commit(ctx2), dgo.ErrAborted, "2nd transaction should have aborted")
}
