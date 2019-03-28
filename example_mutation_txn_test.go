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
	"fmt"
	"log"

	"github.com/dgraph-io/dgo"
	"github.com/dgraph-io/dgo/protos/api"
	"google.golang.org/grpc"
)

// ExampleTxn_Mutation__Txn shows a txn mutation with a conditional query.
// The complete mutation looks like this:
//
// txn {
//   query {
//     me(func: eq(email, "gus@dgrapho.io")) {
//       v as uid
//     }
//   }
//   mutation {
//     set {
//       uid(v) <email> "gus@dgraph.io" .
//     }
//   }
// }
func Example_txnMutation() {
	conn, err := grpc.Dial("127.0.0.1:9180", grpc.WithInsecure())
	if err != nil {
		log.Fatal("While trying to dial gRPC")
	}
	defer conn.Close()

	dc := api.NewDgraphClient(conn)
	dg := dgo.NewDgraphClient(dc)

	op := &api.Operation{}
	op.Schema = `
		name: string .
		email: string @index(exact) .
	`

	ctx := context.Background()
	err = dg.Alter(ctx, op)
	if err != nil {
		log.Fatal(err)
	}

	mutation1 := `
		_:n1 <name> "srfrog" .
		_:n1 <email> "gus@dgraphO.io" .
	`

	mu := &api.Mutation{
		SetNquads: []byte(mutation1),
		CommitNow: true,
	}

	// add a node
	_, err = dg.NewTxn().Mutate(ctx, mu)
	if err != nil {
		log.Fatal(err)
	}

	// conditional query on v var
	condQuery := `
		query {
			me(func: eq(email, "gus@dgraphO.io")) {
				v as uid
			}
		}
	`
	// mutation based on conditional
	mutation2 := `
		uid(v) <email> "gus@dgraph.io" .
	`

	mu.CondQuery = condQuery
	mu.SetNquads = []byte(mutation2)

	// Update email only if matching uid found.
	_, err = dg.NewTxn().Mutate(ctx, mu)
	if err != nil {
		log.Fatal(err)
	}

	query := `
	{
		me(func: eq(email, "gus@dgraph.io")) {
			name
			email
		}
	}`

	// query to verify the change
	resp, err := dg.NewTxn().Query(ctx, query)
	if err != nil {
		log.Fatal(err)
	}

	// resp.Json contains the updated value.
	fmt.Println(string(resp.Json))
	// Output: {"me":[{"name":"srfrog","email":"gus@dgraph.io"}]}
}
