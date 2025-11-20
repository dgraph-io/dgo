/*
 * SPDX-FileCopyrightText: © 2017-2025 Istari Digital, Inc.
 * SPDX-License-Identifier: Apache-2.0
 */

package dgo

import (
	"context"

	"github.com/dgraph-io/dgo/v250/protos/api"
)

func (d *Dgraph) DropAll(ctx context.Context) error {
	req := &api.Operation{DropAll: true}
	return d.doAlter(ctx, req)
}

func (d *Dgraph) DropData(ctx context.Context) error {
	req := &api.Operation{DropOp: api.Operation_DATA}
	return d.doAlter(ctx, req)
}

func (d *Dgraph) DropPredicate(ctx context.Context, predicate string) error {
	req := &api.Operation{DropOp: api.Operation_ATTR, DropValue: predicate}
	return d.doAlter(ctx, req)
}

func (d *Dgraph) DropType(ctx context.Context, typeName string) error {
	req := &api.Operation{DropOp: api.Operation_TYPE, DropValue: typeName}
	return d.doAlter(ctx, req)
}

func (d *Dgraph) SetSchema(ctx context.Context, schema string) error {
	req := &api.Operation{Schema: schema}
	return d.doAlter(ctx, req)
}

func (d *Dgraph) doAlter(ctx context.Context, req *api.Operation) error {
	_, err := doWithRetryLogin(ctx, d, func(dc api.DgraphClient) (*api.Payload, error) {
		return dc.Alter(d.getContext(ctx), req)
	})
	return err
}
