/*
 * SPDX-FileCopyrightText: © Hypermode Inc. <hello@hypermode.com>
 * SPDX-License-Identifier: Apache-2.0
 */

package dgo_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/dgraph-io/dgo/v250"
	apiv2 "github.com/dgraph-io/dgo/v250/protos/api.v2"

	"github.com/stretchr/testify/require"
)

func TestREADME(t *testing.T) {
	client, close := getDgraphClient()
	defer close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	// Drop everything and set schema
	require.NoError(t, client.DropAllNamespaces(ctx))
	require.NoError(t, client.SetSchema(ctx, dgo.RootNamespace,
		`name: string @index(exact) .
		email: string @index(exact) @unique .
		age: int .`))

	query := `schema(pred: [name, age]) {type}`
	resp, err := client.RunDQL(ctx, dgo.RootNamespace, query)
	require.NoError(t, err)
	require.JSONEq(t, `{"schema":[{"predicate":"age","type":"int"},{"predicate":"name","type":"string"}]}`,
		string(resp.QueryResult))

	// Do a mutation
	mutationDQL := `{
			set {
			  _:alice <name> "Alice" .
			  _:alice <email> "alice@example.com" .
			  _:alice <age> "29" .
			}
		  }`
	resp, err = client.RunDQL(ctx, dgo.RootNamespace, mutationDQL)
	require.NoError(t, err)
	require.NotEmpty(t, resp.BlankUids["alice"])

	// Run a query and check we got the result back
	queryDQL := `{
		alice(func: eq(name, "Alice")) {
		  name
		  email
		  age
		}
	  }`
	resp, err = client.RunDQL(ctx, dgo.RootNamespace, queryDQL)
	require.NoError(t, err)
	var m map[string][]struct {
		Name  string `json:"name"`
		Email string `json:"email"`
		Age   int    `json:"age"`
	}
	require.NoError(t, json.Unmarshal(resp.QueryResult, &m))
	require.Equal(t, m["alice"][0].Name, "Alice")
	require.Equal(t, m["alice"][0].Email, "alice@example.com")
	require.Equal(t, m["alice"][0].Age, 29)

	// Run the query with variables
	queryDQLWithVar := `query Alice($name: string) {
		alice(func: eq(name, $name)) {
		  name
		  email
		  age
		}
	  }`
	vars := map[string]string{"$name": "Alice"}
	resp, err = client.RunDQLWithVars(ctx, dgo.RootNamespace, queryDQLWithVar, vars)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(resp.QueryResult, &m))
	require.Equal(t, m["alice"][0].Name, "Alice")
	require.Equal(t, m["alice"][0].Email, "alice@example.com")
	require.Equal(t, m["alice"][0].Age, 29)

	// Best Effort
	resp, err = client.RunDQL(ctx, dgo.RootNamespace, queryDQL, dgo.WithBestEffort())
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(resp.QueryResult, &m))
	require.Equal(t, m["alice"][0].Name, "Alice")
	require.Equal(t, m["alice"][0].Email, "alice@example.com")
	require.Equal(t, m["alice"][0].Age, 29)

	// ReadOnly
	resp, err = client.RunDQL(ctx, dgo.RootNamespace, queryDQL, dgo.WithReadOnly())
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(resp.QueryResult, &m))
	require.Equal(t, m["alice"][0].Name, "Alice")
	require.Equal(t, m["alice"][0].Email, "alice@example.com")
	require.Equal(t, m["alice"][0].Age, 29)

	// RDF Response, note that we can execute the RDFs received from query response
	resp, err = client.RunDQL(ctx, dgo.RootNamespace, queryDQL, dgo.WithResponseFormat(apiv2.RespFormat_RDF))
	require.NoError(t, err)
	mutationDQL = fmt.Sprintf(`{
		set {
		  %s
		}
	  }`, resp.QueryResult)
	resp, err = client.RunDQL(ctx, dgo.RootNamespace, mutationDQL)
	require.NoError(t, err)
	require.Empty(t, resp.BlankUids)
	resp, err = client.RunDQL(ctx, dgo.RootNamespace, queryDQL, dgo.WithReadOnly())
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(resp.QueryResult, &m))
	require.Equal(t, m["alice"][0].Name, "Alice")
	require.Equal(t, m["alice"][0].Email, "alice@example.com")
	require.Equal(t, m["alice"][0].Age, 29)

	// Running an upsert
	upsertQuery := `upsert {
	  query {
		  user as var(func: eq(email, "alice@example.com"))
	  }
	  mutation {
	  	set {
		  uid(user) <age> "30" .
		  uid(user) <name> "Alice Sayum" .
		}
	  }
	}`
	resp, err = client.RunDQL(ctx, dgo.RootNamespace, upsertQuery)
	require.NoError(t, err)
	require.Empty(t, resp.BlankUids)
	require.Equal(t, m["alice"][0].Name, "Alice")

	queryDQL = `{
		alice(func: eq(email, "alice@example.com")) {
		  name
		  email
		  age
		}
	  }`
	resp, err = client.RunDQL(ctx, dgo.RootNamespace, queryDQL, dgo.WithReadOnly())
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(resp.QueryResult, &m))
	require.Equal(t, m["alice"][0].Name, "Alice Sayum")
	require.Equal(t, m["alice"][0].Email, "alice@example.com")
	require.Equal(t, m["alice"][0].Age, 30)

	// Running a Conditional Upsert
	upsertQuery = `upsert {
		query {
		  user as var(func: eq(email, "alice@example.com"))
		}
		mutation @if(eq(len(user), 1)) {
		  set {
			uid(user) <age> "31" .
		  }
		}
	  }`
	resp, err = client.RunDQL(ctx, dgo.RootNamespace, upsertQuery)
	require.NoError(t, err)
	require.Empty(t, resp.BlankUids)
	require.Equal(t, m["alice"][0].Name, "Alice Sayum")

	resp, err = client.RunDQL(ctx, dgo.RootNamespace, queryDQL, dgo.WithReadOnly())
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(resp.QueryResult, &m))
	require.Equal(t, m["alice"][0].Age, 31)
}

func TestNamespaces(t *testing.T) {
	client, close := getDgraphClient()
	defer close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	// Drop everything and set schema
	require.NoError(t, client.DropAllNamespaces(ctx))
	require.NoError(t, client.CreateNamespace(ctx, "finance-graph"))
	require.NoError(t, client.CreateNamespace(ctx, "inventory-graph"))

	require.NoError(t, client.SetSchema(ctx, "finance-graph", "name: string @index(exact) ."))
	require.NoError(t, client.SetSchema(ctx, "inventory-graph", "name: string @index(exact) ."))

	// Rename namespace
	require.NoError(t, client.RenameNamespace(ctx, "finance-graph", "new-finance-graph"))

	namespaces, err := client.ListNamespaces(ctx)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(namespaces), 2)

	// Drop namespaces
	require.NoError(t, client.DropNamespace(ctx, "new-finance-graph"))
	require.NoError(t, client.DropNamespace(ctx, "finance-graph"))
	require.NoError(t, client.DropNamespace(ctx, "inventory-graph"))
}
