/*
 * SPDX-FileCopyrightText: © Hypermode Inc. <hello@hypermode.com>
 * SPDX-License-Identifier: Apache-2.0
 */

package dgo

import (
	"context"

	apiv25 "github.com/dgraph-io/dgo/v240/protos/api.v25"
)

type txnOptions struct {
	readOnly   bool
	bestEffort bool
	respFormat apiv25.RespFormat
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

// WithResponseFormat sets the response format for queries. By default, the
// response format is JSON. We can also specify RDF format.
func WithResponseFormat(respFormat apiv25.RespFormat) TxnOption {
	return func(o *txnOptions) error {
		o.respFormat = respFormat
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
	*apiv25.RunDQLResponse, error) {

	return d.RunDQLWithVars(ctx, nsName, q, nil, opts...)
}

// RunDQLWithVars is like RunDQL with variables.
func (d *Dgraph) RunDQLWithVars(ctx context.Context, nsName string, q string,
	vars map[string]string, opts ...TxnOption) (*apiv25.RunDQLResponse, error) {

	topts, err := buildTxnOptions(opts...)
	if err != nil {
		return nil, err
	}

	dc := d.anyClientv25()
	req := &apiv25.RunDQLRequest{NsName: nsName, DqlQuery: q, Vars: vars,
		ReadOnly: topts.readOnly, BestEffort: topts.bestEffort}
	return doWithRetryLogin(ctx, d, func() (*apiv25.RunDQLResponse, error) {
		return dc.RunDQL(d.getContext(ctx), req)
	})
}
