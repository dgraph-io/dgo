/*
 * SPDX-FileCopyrightText: © 2017-2025 Istari Digital, Inc.
 * SPDX-License-Identifier: Apache-2.0
 */

package dgo

import (
	"context"

	api "github.com/dgraph-io/dgo/v250/protos/api"
)

// AllocateUIDs allocates a given number of Node UIDs in the Graph and returns a start and end UIDs,
// end excluded. The UIDs in the range [start, end) can then be used by the client in the mutations
// going forward. Note that, each node in a Graph is assigned a UID in Dgraph. Dgraph ensures that
// these UIDs are not allocated anywhere else throughout the operation of this cluster. This is useful
// in bulk loader or live loader or similar applications.
func (d *Dgraph) AllocateUIDs(ctx context.Context, howMany uint64) (uint64, uint64, error) {
	return d.allocateIDs(ctx, howMany, api.LeaseType_UID)
}

// AllocateTimestamps gets a sequence of timestamps allocated from Dgraph. These timestamps can be
// used in bulk loader and similar applications.
func (d *Dgraph) AllocateTimestamps(ctx context.Context, howMany uint64) (uint64, uint64, error) {
	return d.allocateIDs(ctx, howMany, api.LeaseType_TS)
}

// AllocateNamespaces allocates a given number of namespaces in the Graph and returns a start and end
// namespaces, end excluded. The namespaces in the range [start, end) can then be used by the client.
// Dgraph ensures that these namespaces are NOT allocated anywhere else throughout the operation of
// this cluster. This is useful in bulk loader or live loader or similar applications.
func (d *Dgraph) AllocateNamespaces(ctx context.Context, howMany uint64) (uint64, uint64, error) {
	return d.allocateIDs(ctx, howMany, api.LeaseType_NS)
}

func (d *Dgraph) allocateIDs(ctx context.Context, howMany uint64,
	leaseType api.LeaseType) (uint64, uint64, error) {

	req := &api.AllocateIDsRequest{HowMany: howMany, LeaseType: leaseType}
	resp, err := doWithRetryLogin(ctx, d, func(dc api.DgraphClient) (*api.AllocateIDsResponse, error) {
		return dc.AllocateIDs(d.getContext(ctx), req)
	})
	if err != nil {
		return 0, 0, err
	}
	return resp.Start, resp.End, nil
}
