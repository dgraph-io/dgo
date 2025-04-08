/*
 * SPDX-FileCopyrightText: © Hypermode Inc. <hello@hypermode.com>
 * SPDX-License-Identifier: Apache-2.0
 */

package dgo

import (
	"context"

	apiv25 "github.com/dgraph-io/dgo/v250/protos/api.v25"
)

func (d *Dgraph) DropAllNamespaces(ctx context.Context) error {
	req := &apiv25.AlterRequest{Op: apiv25.AlterOp_DROP_ALL}
	return d.doAlter(ctx, req)
}

func (d *Dgraph) DropAll(ctx context.Context, nsName string) error {
	req := &apiv25.AlterRequest{
		Op:     apiv25.AlterOp_DROP_ALL_IN_NS,
		NsName: nsName,
	}
	return d.doAlter(ctx, req)
}

func (d *Dgraph) DropData(ctx context.Context, nsName string) error {
	req := &apiv25.AlterRequest{
		Op:     apiv25.AlterOp_DROP_DATA_IN_NS,
		NsName: nsName,
	}
	return d.doAlter(ctx, req)
}

func (d *Dgraph) DropPredicate(ctx context.Context, nsName, predicate string) error {
	req := &apiv25.AlterRequest{
		Op:              apiv25.AlterOp_DROP_PREDICATE_IN_NS,
		NsName:          nsName,
		PredicateToDrop: predicate,
	}
	return d.doAlter(ctx, req)
}

func (d *Dgraph) DropType(ctx context.Context, nsName, typeName string) error {
	req := &apiv25.AlterRequest{
		Op:         apiv25.AlterOp_DROP_TYPE_IN_NS,
		NsName:     nsName,
		TypeToDrop: typeName,
	}
	return d.doAlter(ctx, req)
}

func (d *Dgraph) SetSchema(ctx context.Context, nsName string, schema string) error {
	req := &apiv25.AlterRequest{
		Op:     apiv25.AlterOp_SCHEMA_IN_NS,
		NsName: nsName,
		Schema: schema,
	}
	return d.doAlter(ctx, req)
}

func (d *Dgraph) doAlter(ctx context.Context, req *apiv25.AlterRequest) error {
	_, err := doWithRetryLogin(ctx, d, func(dc apiv25.DgraphClient) (*apiv25.AlterResponse, error) {
		return dc.Alter(d.getContext(ctx), req)
	})
	return err
}
