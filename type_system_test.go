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
	"encoding/json"
	"fmt"
	"testing"

	"github.com/dgraph-io/dgo/v200"
	"github.com/dgraph-io/dgo/v200/protos/api"

	"github.com/stretchr/testify/require"
)

var (
	alicename = "alice"

	personType = `
	type person {
		name: string
		age: int
	}`

	empType = `
	type employee {
		name: string
		email: string
		works_at: string
	}`
)

func addType(t *testing.T, dg *dgo.Dgraph, newType string) {
	op := &api.Operation{
		Schema: newType,
	}
	err := dg.Alter(context.Background(), op)
	require.NoError(t, err, "error while creating type: %s", newType)
}

func initializeDBTypeSystem(t *testing.T, dg *dgo.Dgraph) {
	op := &api.Operation{
		Schema: `
		email: string @index(exact) .
		name: string @index(exact) .
		works_at: string @index(exact) .
		age: int .`,
	}

	err := dg.Alter(context.Background(), op)
	require.NoError(t, err)

	mu := &api.Mutation{
		CommitNow: true,
		SetNquads: []byte(`
		_:user <email> "alice@company1.io" .
		_:user <name> "alice" .
		_:user <empid> "1" .
		_:user <works_at> "company1" .
		_:user <age> "20" .`),
	}

	_, err = dg.NewTxn().Mutate(context.Background(), mu)
	require.NoError(t, err, "unable to insert record")
}

func TestNoType(t *testing.T) {
	dg, cancel := getDgraphClient()
	defer cancel()

	ctx := context.Background()
	err := dg.Alter(ctx, &api.Operation{DropAll: true})
	require.NoError(t, err)

	initializeDBTypeSystem(t, dg)

	q := `{
		q(func: eq(name, "%s")) {
			dgraph.type
		}
	}`

	res, err := dg.NewReadOnlyTxn().Query(context.Background(), fmt.Sprintf(q, alicename))
	var ts struct {
		Q []struct {
			DgraphType []string `json:"dgraph.type"`
		} `json:"q"`
	}
	err = json.Unmarshal(res.Json, &ts)
	require.NoError(t, err, "unable to parse type response")

	require.Zero(t, len(ts.Q), "no dgraph type has been assigned")
}

func TestSingleType(t *testing.T) {
	// Setup.
	dg, cancel := getDgraphClient()
	defer cancel()

	ctx := context.Background()
	err := dg.Alter(ctx, &api.Operation{DropAll: true})
	require.NoError(t, err)

	initializeDBTypeSystem(t, dg)
	addType(t, dg, personType)

	// Update type of user to person.
	q1 := `{
		v as var(func: eq(name, "%s"))
	}`

	req := &api.Request{
		CommitNow: true,
		Query:     fmt.Sprintf(q1, alicename),
		Mutations: []*api.Mutation{
			{SetNquads: []byte(`uid(v) <dgraph.type> "person" .`)},
		},
	}
	_, err = dg.NewTxn().Do(context.Background(), req)
	require.NoError(t, err, "unable to add type person to user")

	// Verify if type of user is person.
	q2 := `{
		q(func: eq(name, "%s")) {
			dgraph.type
		}
	}`

	res1, err := dg.NewReadOnlyTxn().Query(context.Background(), fmt.Sprintf(q2, alicename))
	var ts struct {
		Q []struct {
			DgraphType []string `json:"dgraph.type"`
		} `json:"q"`
	}
	err = json.Unmarshal(res1.Json, &ts)
	require.NoError(t, err, "unable to parse type response")

	require.Equal(t, 1, len(ts.Q[0].DgraphType), "one dgraph type has been assigned")

	// Perform expand(_all_) on
	q3 := `{
		q(func: eq(name, "%s")) {
			expand(_all_)
		}
	}`

	res2, err := dg.NewReadOnlyTxn().Query(context.Background(), fmt.Sprintf(q3, alicename))
	require.NoError(t, err, "unable to expand for type user")

	var ts2 struct {
		Q []struct {
			Name string `json:"name"`
			Age  int    `json:"age"`
		} `json:"q"`
	}

	err = json.Unmarshal(res2.Json, &ts2)
	require.NoError(t, err, "unable to parse json in expand response")

	require.Equal(t, len(ts2.Q), 1)
	require.Equal(t, ts2.Q[0].Name, alicename)
	require.Equal(t, ts2.Q[0].Age, 20)

	// Delete S * *
	q4 := `{
		v as var(func: eq(name, "%s"))
	}`

	req2 := &api.Request{
		CommitNow: true,
		Query:     fmt.Sprintf(q4, alicename),
		Mutations: []*api.Mutation{
			{DelNquads: []byte(`uid(v) * * .`)},
		},
	}
	_, err = dg.NewTxn().Do(context.Background(), req2)
	require.NoError(t, err, "unable to execute S * *")

	q5 := `{
		q(func: eq(name, %s)) {
			name
			age
			email
			works_at
		}
	}`

	res3, err := dg.NewReadOnlyTxn().Query(context.Background(), fmt.Sprintf(q5, alicename))
	require.NoError(t, err, "error while querying after delete S * *")

	var ts3 struct {
		Q []struct {
			Name    string `json:"name"`
			Age     int    `json:"age"`
			Email   string `json:"email"`
			WorksAt string `json:"works_at"`
		} `json:"q"`
	}
	require.NoError(t, json.Unmarshal(res3.Json, &ts3))
	require.Zero(t, len(ts3.Q))
}

func TestMultipleType(t *testing.T) {
	// Setup.
	dg, cancel := getDgraphClient()
	defer cancel()

	ctx := context.Background()
	err := dg.Alter(ctx, &api.Operation{DropAll: true})
	require.NoError(t, err)

	initializeDBTypeSystem(t, dg)
	addType(t, dg, personType)
	addType(t, dg, empType)

	// Add person and employee to user.
	q1 := `{
		v as var(func: eq(name, "%s"))
	}`
	mu := `
	uid(v) <dgraph.type> "person" .
	uid(v) <dgraph.type> "employee" .
	`

	req := &api.Request{
		CommitNow: true,
		Query:     fmt.Sprintf(q1, alicename),
		Mutations: []*api.Mutation{
			{SetNquads: []byte(mu)},
		},
	}
	_, err = dg.NewTxn().Do(context.Background(), req)
	require.NoError(t, err, "unable to add type person to user")

	// Read Types for user.
	q2 := `{
		q(func: eq(name, "%s")) {
			dgraph.type
		}
	}`

	res1, err := dg.NewReadOnlyTxn().Query(context.Background(), fmt.Sprintf(q2, alicename))
	var ts struct {
		Q []struct {
			DgraphType []string `json:"dgraph.type"`
		} `json:"q"`
	}
	err = json.Unmarshal(res1.Json, &ts)
	require.NoError(t, err, "unable to parse type response")

	require.Equal(t, 2, len(ts.Q[0].DgraphType), "one dgraph type has been assigned")

	// Test expand(_all_) for user.
	q3 := `{
		q(func: eq(name, "%s")) {
			expand(_all_)
		}
	}`

	res2, err := dg.NewReadOnlyTxn().Query(context.Background(), fmt.Sprintf(q3, alicename))
	require.NoError(t, err, "unable to expand for type user")

	var ts2 struct {
		Q []struct {
			Name    string `json:"name"`
			Age     int    `json:"age"`
			Email   string `json:"email"`
			WorksAt string `json:"works_at"`
		} `json:"q"`
	}

	err = json.Unmarshal(res2.Json, &ts2)
	require.NoError(t, err, "unable to parse json in expand response")

	require.Equal(t, len(ts2.Q), 1)
	require.Equal(t, ts2.Q[0].Name, alicename)
	require.Equal(t, ts2.Q[0].Age, 20)
	require.Equal(t, ts2.Q[0].Email, "alice@company1.io")
	require.Equal(t, ts2.Q[0].WorksAt, "company1")
}
