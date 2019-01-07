/*
 * Copyright (C) 2017 Dgraph Labs, Inc. and Contributors
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
	"fmt"
	"math/rand"
	"strings"
	"sync"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"google.golang.org/grpc/metadata"

	"github.com/dgraph-io/dgo/protos/api"
)

// Dgraph is a transaction aware client to a set of dgraph server instances.
type Dgraph struct {
	jwtMutex sync.RWMutex
	jwt      api.Jwt
	dc       []api.DgraphClient
}

// NewDgraphClient creates a new Dgraph for interacting with the Dgraph store connected to in
// conns.
// The client can be backed by multiple connections (to the same server, or multiple servers in a
// cluster).
//
// A single client is thread safe for sharing with multiple go routines.
func NewDgraphClient(clients ...api.DgraphClient) *Dgraph {
	dg := &Dgraph{
		dc: clients,
	}

	return dg
}

func (d *Dgraph) Login(ctx context.Context, userid string, password string) error {
	d.jwtMutex.Lock()
	defer d.jwtMutex.Unlock()

	dc := d.anyClient()
	loginRequest := &api.LoginRequest{
		Userid:   userid,
		Password: password,
	}
	resp, err := dc.Login(ctx, loginRequest)
	if err != nil {
		return err
	}

	return d.jwt.Unmarshal(resp.Json)
}

func (d *Dgraph) loginWithRefreshJwt(ctx context.Context) error {
	d.jwtMutex.Lock()
	defer d.jwtMutex.Unlock()

	if len(d.jwt.RefreshJwt) == 0 {
		return fmt.Errorf("refresh jwt should not be empty")
	}

	dc := d.anyClient()
	loginRequest := &api.LoginRequest{
		RefreshToken: d.jwt.RefreshJwt,
	}
	resp, err := dc.Login(ctx, loginRequest)
	if err != nil {
		return err
	}

	d.jwtMutex.Lock()
	defer d.jwtMutex.Unlock()
	return d.jwt.Unmarshal(resp.Json)
}

func (d *Dgraph) getContext(ctx context.Context) context.Context {
	md, ok := metadata.FromOutgoingContext(ctx)
	if !ok {
		return ctx
	}

	d.jwtMutex.RLock()
	defer d.jwtMutex.RUnlock()
	if len(d.jwt.AccessJwt) > 0 {
		md.Append("accessJwt", d.jwt.AccessJwt)
	}

	return metadata.NewOutgoingContext(ctx, md)
}

// By setting various fields of api.Operation, Alter can be used to do the
// following:
//
// 1. Modify the schema.
//
// 2. Drop a predicate.
//
// 3. Drop the database.
func (d *Dgraph) Alter(ctx context.Context, op *api.Operation) error {
	dc := d.anyClient()

	ctxWithJwt := d.getContext(ctx)
	_, err := dc.Alter(ctxWithJwt, op)
	st, ok := status.FromError(err)
	if ok && st.Code() == codes.Unauthenticated &&
		strings.Contains(err.Error(), "jwt token has expired") {
		// try to login with the refreshJwt
		if err = d.loginWithRefreshJwt(ctx); err != nil {
			return fmt.Errorf("unauthenticated to alter and unable to login with "+
				"refresh jwt: %v", err)
		}

		ctxWithJwt = d.getContext(ctx)
		_, err = dc.Alter(ctxWithJwt, op)
		return err
	}

	return err
}

// the query API should only be used within a Txn object
func (d *Dgraph) query(ctx context.Context, op *api.Request) (*api.Response, error) {
	dc := d.anyClient()

	ctxWithJwt := d.getContext(ctx)
	resp, err := dc.Query(ctxWithJwt, op)
	st, ok := status.FromError(err)
	if ok && st.Code() == codes.Unauthenticated &&
		strings.Contains(err.Error(), "jwt token has expired") {
		// try to login with the refreshJwt
		if err = d.loginWithRefreshJwt(ctx); err != nil {
			return nil, fmt.Errorf("unauthenticated to query and unable to login with "+
				"refresh jwt: %v", err)
		}

		ctxWithJwt = d.getContext(ctx)
		return dc.Query(ctxWithJwt, op)
	}
	return resp, err
}

// the mutate API should only be used within a Txn object
func (d *Dgraph) mutate(ctx context.Context, mu *api.Mutation) (*api.Assigned, error) {
	dc := d.anyClient()

	ctxWithJwt := d.getContext(ctx)
	assigned, err := dc.Mutate(ctxWithJwt, mu)
	st, ok := status.FromError(err)
	if ok && st.Code() == codes.Unauthenticated &&
		strings.Contains(err.Error(), "jwt token has expired") {
		// try to login with the refreshJwt
		if err = d.loginWithRefreshJwt(ctx); err != nil {
			return nil, fmt.Errorf("unauthenticated to mutate and unable to login with "+
				"refresh jwt: %v", err)
		}

		ctxWithJwt = d.getContext(ctx)
		return dc.Mutate(ctxWithJwt, mu)
	}

	return assigned, err
}

func (d *Dgraph) commitOrAbort(ctx context.Context, txn *api.TxnContext) (*api.TxnContext, error) {
	dc := d.anyClient()

	ctxWithJwt := d.getContext(ctx)
	txnContext, err := dc.CommitOrAbort(ctxWithJwt, txn)
	st, ok := status.FromError(err)
	if ok && st.Code() == codes.Unauthenticated &&
		strings.Contains(err.Error(), "jwt token has expired") {
		// try to login with the refreshJwt
		if err = d.loginWithRefreshJwt(ctx); err != nil {
			return nil, fmt.Errorf("unauthenticated to commit or abort and "+
				"unable to login with refresh jwt: %v", err)
		}

		ctxWithJwt = d.getContext(ctx)
		return dc.CommitOrAbort(ctxWithJwt, txn)
	}
	return txnContext, err
}

func (d *Dgraph) anyClient() api.DgraphClient {
	return d.dc[rand.Intn(len(d.dc))]
}

// DeleteEdges sets the edges corresponding to predicates on the node with the given uid
// for deletion.
// This helper function doesn't run the mutation on the server. It must be done by the user
// after the function returns.
func DeleteEdges(mu *api.Mutation, uid string, predicates ...string) {
	for _, predicate := range predicates {
		mu.Del = append(mu.Del, &api.NQuad{
			Subject:   uid,
			Predicate: predicate,
			// _STAR_ALL is defined as x.Star in x package.
			ObjectValue: &api.Value{Val: &api.Value_DefaultVal{"_STAR_ALL"}},
		})
	}
}
