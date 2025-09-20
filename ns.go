/*
 * SPDX-FileCopyrightText: © Hypermode Inc. <hello@hypermode.com>
 * SPDX-License-Identifier: Apache-2.0
 */

package dgo

import (
	"context"

	"github.com/dgraph-io/dgo/v250/protos/api"
)

type txnOptions struct {
	readOnly   bool
	bestEffort bool
	respFormat api.Request_RespFormat
}

// TxnOption is a function that modifies the txn options.
type TxnOption func(*txnOptions) error

// WithReadOnly sets the txn to be read-only.
func WithReadOnly() TxnOption {
	return func(o *txnOptions) error {
		o.readOnly = true
		return nil
	}
}

// WithBestEffort sets the txn to be best effort.
func WithBestEffort() TxnOption {
	return func(o *txnOptions) error {
		o.readOnly = true
		o.bestEffort = true
		return nil
	}
}

func buildTxnOptions(opts ...TxnOption) (*txnOptions, error) {
	topts := &txnOptions{}
	for _, opt := range opts {
		if err := opt(topts); err != nil {
			return nil, err
		}
	}
	if topts.bestEffort {
		topts.readOnly = true
	}

	return topts, nil
}

// RunDQL runs a DQL query in the given namespace. A DQL query could be a mutation
// or a query or an upsert which is a combination of mutations and queries.
func (d *Dgraph) RunDQL(ctx context.Context, q string, opts ...TxnOption) (
	*api.Response, error) {

	return d.RunDQLWithVars(ctx, q, nil, opts...)
}

// RunDQLWithVars is like RunDQL with variables.
func (d *Dgraph) RunDQLWithVars(ctx context.Context, q string,
	vars map[string]string, opts ...TxnOption) (*api.Response, error) {

	topts, err := buildTxnOptions(opts...)
	if err != nil {
		return nil, err
	}

	req := &api.RunDQLRequest{DqlQuery: q, Vars: vars,
		ReadOnly: topts.readOnly, BestEffort: topts.bestEffort, RespFormat: topts.respFormat}
	return doWithRetryLogin(ctx, d, func(dc api.DgraphClient) (*api.Response, error) {
		return dc.RunDQL(d.getContext(ctx), req)
	})
}

// CreateNamespace creates a new namespace with the given name and password for groot user.
func (d *Dgraph) CreateNamespace(ctx context.Context) (uint64, error) {
	req := &api.CreateNamespaceRequest{}
	resp, err := doWithRetryLogin(ctx, d, func(dc api.DgraphClient) (*api.CreateNamespaceResponse, error) {
		return dc.CreateNamespace(d.getContext(ctx), req)
	})
	if err != nil {
		return 0, err
	}
	return resp.Namespace, nil
}

// DropNamespace deletes the namespace with the given name.
func (d *Dgraph) DropNamespace(ctx context.Context, nsID uint64) error {
	req := &api.DropNamespaceRequest{Namespace: nsID}
	_, err := doWithRetryLogin(ctx, d, func(dc api.DgraphClient) (*api.DropNamespaceResponse, error) {
		return dc.DropNamespace(d.getContext(ctx), req)
	})
	return err
}

// ListNamespaces returns a map of namespace names to their details.
func (d *Dgraph) ListNamespaces(ctx context.Context) (map[uint64]*api.Namespace, error) {
	resp, err := doWithRetryLogin(ctx, d, func(dc api.DgraphClient) (*api.ListNamespacesResponse, error) {
		return dc.ListNamespaces(d.getContext(ctx), &api.ListNamespacesRequest{})
	})
	if err != nil {
		return nil, err
	}
	return resp.Namespaces, nil
}

func doWithRetryLogin[T any](ctx context.Context, d *Dgraph,
	f func(dc api.DgraphClient) (*T, error)) (*T, error) {

	dc := d.anyClient()
	resp, err := f(dc)
	if isJwtExpired(err) {
		if err := d.retryLogin(ctx); err != nil {
			return nil, err
		}
		return f(dc)
	}
	return resp, err
}
