/*
 * SPDX-FileCopyrightText: © Hypermode Inc. <hello@hypermode.com>
 * SPDX-License-Identifier: Apache-2.0
 */

package dgo

import (
	"context"
	"errors"
	"math/rand"

	apiv2 "github.com/dgraph-io/dgo/v250/protos/api.v2"
)

const (
	RootNamespace = "root"
)

var (
	ErrUnsupportedAPI = errors.New("API is not supported by the version of dgraph cluster")
)

type txnOptions struct {
	readOnly   bool
	bestEffort bool
	respFormat apiv2.RespFormat
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
func (d *Dgraph) RunDQL(ctx context.Context, nsName string, q string, opts ...TxnOption) (
	*apiv2.RunDQLResponse, error) {

	return d.RunDQLWithVars(ctx, nsName, q, nil, opts...)
}

// RunDQLWithVars is like RunDQL with variables.
func (d *Dgraph) RunDQLWithVars(ctx context.Context, nsName string, q string,
	vars map[string]string, opts ...TxnOption) (*apiv2.RunDQLResponse, error) {

	topts, err := buildTxnOptions(opts...)
	if err != nil {
		return nil, err
	}

	req := &apiv2.RunDQLRequest{NsName: nsName, DqlQuery: q, Vars: vars,
		ReadOnly: topts.readOnly, BestEffort: topts.bestEffort, RespFormat: topts.respFormat}
	return doWithRetryLogin(ctx, d, func(dc apiv2.DgraphClient) (*apiv2.RunDQLResponse, error) {
		return dc.RunDQL(d.getContext(ctx), req)
	})
}

// CreateNamespace creates a new namespace with the given name and password for groot user.
func (d *Dgraph) CreateNamespace(ctx context.Context, name string) error {
	req := &apiv2.CreateNamespaceRequest{NsName: name}
	_, err := doWithRetryLogin(ctx, d, func(dc apiv2.DgraphClient) (*apiv2.CreateNamespaceResponse, error) {
		return dc.CreateNamespace(d.getContext(ctx), req)
	})
	return err
}

// DropNamespace deletes the namespace with the given name.
func (d *Dgraph) DropNamespace(ctx context.Context, name string) error {
	req := &apiv2.DropNamespaceRequest{NsName: name}
	_, err := doWithRetryLogin(ctx, d, func(dc apiv2.DgraphClient) (*apiv2.DropNamespaceResponse, error) {
		return dc.DropNamespace(d.getContext(ctx), req)
	})
	return err
}

// RenameNamespace renames the namespace from the given name to the new name.
func (d *Dgraph) RenameNamespace(ctx context.Context, from string, to string) error {
	req := &apiv2.UpdateNamespaceRequest{NsName: from, RenameToNs: to}
	_, err := doWithRetryLogin(ctx, d, func(dc apiv2.DgraphClient) (*apiv2.UpdateNamespaceResponse, error) {
		return dc.UpdateNamespace(d.getContext(ctx), req)
	})
	return err
}

// ListNamespaces returns a map of namespace names to their details.
func (d *Dgraph) ListNamespaces(ctx context.Context) (map[string]*apiv2.Namespace, error) {
	resp, err := doWithRetryLogin(ctx, d, func(dc apiv2.DgraphClient) (*apiv2.ListNamespacesResponse, error) {
		return dc.ListNamespaces(d.getContext(ctx), &apiv2.ListNamespacesRequest{})
	})
	if err != nil {
		return nil, err
	}

	return resp.NsList, nil
}

func (d *Dgraph) anyClientv2() apiv2.DgraphClient {
	//nolint:gosec
	return d.dcv2[rand.Intn(len(d.dcv2))]
}

func doWithRetryLogin[T any](ctx context.Context, d *Dgraph,
	f func(dc apiv2.DgraphClient) (*T, error)) (*T, error) {

	if d.useV1 {
		return nil, ErrUnsupportedAPI
	}

	dc := d.anyClientv2()
	resp, err := f(dc)
	if isJwtExpired(err) {
		if err := d.retryLogin(ctx); err != nil {
			return nil, err
		}
		return f(dc)
	}
	return resp, err
}
