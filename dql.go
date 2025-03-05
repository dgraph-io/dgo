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
	to := &txnOptions{}
	for _, opt := range opts {
		if err := opt(to); err != nil {
			return nil, err
		}
	}
	if to.bestEffort {
		to.readOnly = true
	}

	return to, nil
}

func (d *Dgraph) RunDQL(ctx context.Context, nsName string, q string, opts ...TxnOption) (*apiv25.RunDQLResponse, error) {
	to, err := buildTxnOptions(opts...)
	if err != nil {
		return nil, err
	}

	dc := d.anyClientv25()
	req := &apiv25.RunDQLRequest{NsName: nsName, DqlQuery: q,
		ReadOnly: to.readOnly, BestEffort: to.bestEffort}
	return doWithRetryLogin(ctx, d, func() (*apiv25.RunDQLResponse, error) {
		return dc.RunDQL(d.getContext(ctx), req)
	})
}

func (d *Dgraph) RunDQLWithVars(ctx context.Context, nsName string, q string,
	vars map[string]string, opts ...TxnOption) (*apiv25.RunDQLResponse, error) {

	to, err := buildTxnOptions(opts...)
	if err != nil {
		return nil, err
	}

	dc := d.anyClientv25()
	req := &apiv25.RunDQLRequest{NsName: nsName, DqlQuery: q, Vars: vars,
		ReadOnly: to.readOnly, BestEffort: to.bestEffort}
	return doWithRetryLogin(ctx, d, func() (*apiv25.RunDQLResponse, error) {
		return dc.RunDQL(d.getContext(ctx), req)
	})
}
