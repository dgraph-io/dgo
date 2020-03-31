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

	"github.com/dgraph-io/dgo/v200/protos/api"
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
