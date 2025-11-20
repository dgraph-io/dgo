/*
 * SPDX-FileCopyrightText: © 2017-2025 Istari Digital, Inc.
 * SPDX-License-Identifier: Apache-2.0
 */

package dgo_test

import (
	"context"
	"fmt"
	"log"

	"github.com/dgraph-io/dgo/v250/protos/api"
)

func Example_getSchema() {
	dg, cancel := getDgraphClient()
	defer cancel()

	op := &api.Operation{}
	op.Schema = `
		name: string @index(exact) .
		age: int .
		married: bool .
		loc: geo .
		dob: datetime .
	`

	ctx := context.Background()
	err := dg.Alter(ctx, op)
	if err != nil {
		log.Fatal(err)
	}

	// Ask for the type of name and age.
	resp, err := dg.NewTxn().Query(ctx, `schema(pred: [name, age]) {type}`)
	if err != nil {
		log.Fatal(err)
	}

	// resp.Json contains the schema query response.
	fmt.Println(string(resp.Json))
	// Output: {"schema":[{"predicate":"age","type":"int"},{"predicate":"name","type":"string"}]}
}
