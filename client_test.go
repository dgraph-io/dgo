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
	"github.com/dgraph-io/dgo/v250/protos/api"

	"github.com/stretchr/testify/require"
)

// This test only ensures that connection strings are parsed correctly.
func TestOpen(t *testing.T) {
	var err error

	_, err = dgo.Open("127.0.0.1:9180")
	require.ErrorContains(t, err, "first path segment in URL cannot contain colon")

	_, err = dgo.Open("localhost:9180")
	require.ErrorContains(t, err, "invalid scheme: must start with dgraph://")

	_, err = dgo.Open("dgraph://localhost:9180")
	require.NoError(t, err)

	_, err = dgo.Open("dgraph://localhost")
	require.ErrorContains(t, err, "invalid connection string: host url must have both host and port")

	_, err = dgo.Open("dgraph://localhost:")
	require.ErrorContains(t, err, "invalid connection string: missing port after port-separator colon")

	_, err = dgo.Open("dgraph://localhost:9180?sslmode=verify-ca")
	require.ErrorContains(t, err, "first record does not look like a TLS handshake")

	_, err = dgo.Open("dgraph://localhost:9180?sslmode=prefer")
	require.ErrorContains(t, err, "invalid SSL mode: prefer (must be one of disable, require, verify-ca)")

	_, err = dgo.Open("dgraph://localhost:9180?sslmode=disable&bearertoken=abc")
	require.ErrorContains(t, err, "grpc: the credentials require transport level security")

	_, err = dgo.Open("dgraph://localhost:9180?sslmode=disable&apikey=abc")
	require.ErrorContains(t, err, "grpc: the credentials require transport level security")

	_, err = dgo.Open("dgraph://localhost:9180?sslmode=disable&apikey=abc&bearertoken=bgf")
	require.ErrorContains(t, err, "invalid connection string: both apikey and bearertoken cannot be provided")

	_, err = dgo.Open("dgraph://localhost:9180?sslmode=verify-ca&bearertoken=hfs")
	require.ErrorContains(t, err, "first record does not look like a TLS handshake")

	_, err = dgo.Open("dgraph://localhost:9180?sslmode=verify-ca&apikey=hfs")
	require.ErrorContains(t, err, "first record does not look like a TLS handshake")

	_, err = dgo.Open("dgraph://localhost:9180?sslmode=require&bearertoken=hfs")
	require.ErrorContains(t, err, "first record does not look like a TLS handshake")

	_, err = dgo.Open("dgraph://localhost:9180?sslmode=require&apikey=hfs")
	require.ErrorContains(t, err, "first record does not look like a TLS handshake")

	_, err = dgo.Open("dgraph://localhost:9180?sslm")
	require.NoError(t, err)

	_, err = dgo.Open("dgraph://localhost:9180?sslm")
	require.NoError(t, err)

	_, err = dgo.Open("dgraph://user:pass@localhost:9180")
	require.ErrorContains(t, err, "invalid username or password")

	_, err = dgo.Open("dgraph://user:pass@localhost:9180?namespace=root")
	require.ErrorContains(t, err, "invalid namespace ID: strconv.ParseUint: parsing \"root\": invalid syntax")

	_, err = dgo.Open("dgraph://user:pass@localhost:9180?namespace=1")
	require.ErrorContains(t, err, "invalid username or password")

	_, err = dgo.Open("dgraph://groot:password@localhost:9180")
	require.NoError(t, err)
}

func TestREADME(t *testing.T) {
	client, close := getDgraphClient()
	defer close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	// Drop everything and set schema
	require.NoError(t, client.DropAll(ctx))
	require.NoError(t, client.SetSchema(ctx,
		`name: string @index(exact) .
		email: string @index(exact) @unique .
		age: int .`))

	query := `schema(pred: [name, age]) {type}`
	resp, err := client.RunDQL(ctx, query, dgo.WithBestEffort())
	require.NoError(t, err)
	require.JSONEq(t, `{"schema":[{"predicate":"age","type":"int"},{"predicate":"name","type":"string"}]}`,
		string(resp.Json))

	// Do a mutation
	mutationDQL := `{
			set {
			  _:alice <name> "Alice" .
			  _:alice <email> "alice@example.com" .
			  _:alice <age> "29" .
			}
		  }`
	resp, err = client.RunDQL(ctx, mutationDQL)
	require.NoError(t, err)
	require.NotEmpty(t, resp.Uids["alice"])

	// Run a query and check we got the result back
	queryDQL := `{
		alice(func: eq(name, "Alice")) {
		  name
		  email
		  age
		}
	  }`
	resp, err = client.RunDQL(ctx, queryDQL)
	require.NoError(t, err)
	var m map[string][]struct {
		Name  string `json:"name"`
		Email string `json:"email"`
		Age   int    `json:"age"`
	}
	require.NoError(t, json.Unmarshal(resp.Json, &m))
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
	resp, err = client.RunDQLWithVars(ctx, queryDQLWithVar, vars)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(resp.Json, &m))
	require.Equal(t, m["alice"][0].Name, "Alice")
	require.Equal(t, m["alice"][0].Email, "alice@example.com")
	require.Equal(t, m["alice"][0].Age, 29)

	// Best Effort
	resp, err = client.RunDQL(ctx, queryDQL, dgo.WithBestEffort())
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(resp.Json, &m))
	require.Equal(t, m["alice"][0].Name, "Alice")
	require.Equal(t, m["alice"][0].Email, "alice@example.com")
	require.Equal(t, m["alice"][0].Age, 29)

	// ReadOnly
	resp, err = client.RunDQL(ctx, queryDQL, dgo.WithReadOnly())
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(resp.Json, &m))
	require.Equal(t, m["alice"][0].Name, "Alice")
	require.Equal(t, m["alice"][0].Email, "alice@example.com")
	require.Equal(t, m["alice"][0].Age, 29)

	// RDF Response, note that we can execute the RDFs received from query response
	resp, err = client.RunDQL(ctx, queryDQL, dgo.WithResponseFormat(api.Request_RDF))
	require.NoError(t, err)
	mutationDQL = fmt.Sprintf(`{
		set {
		  %s
		}
	  }`, resp.Json)
	resp, err = client.RunDQL(ctx, mutationDQL)
	require.NoError(t, err)
	require.Empty(t, resp.Uids)
	resp, err = client.RunDQL(ctx, queryDQL, dgo.WithReadOnly())
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(resp.Json, &m))
	require.Equal(t, m["alice"][0].Name, "Alice")
	require.Equal(t, m["alice"][0].Email, "alice@example.com")
	require.Equal(t, m["alice"][0].Age, 29)

	// JSON Response, check that we can execute the JSON received from query response
	// DQL with JSON Data format should be valid as per https://docs.hypermode.com/dgraph/dql/json
	jsonBytes, marshallErr := json.Marshal(m["alice"])
	require.NoError(t, marshallErr)
	mutationDQL = fmt.Sprintf(`{
		"set": %s
	  }`, jsonBytes)
	resp, err = client.RunDQL(ctx, mutationDQL)
	require.NoError(t, err)
	require.NotEmpty(t, resp.Uids["alice"])
	resp, err = client.RunDQL(ctx, queryDQL, dgo.WithReadOnly())
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(resp.Json, &m))
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
	resp, err = client.RunDQL(ctx, upsertQuery)
	require.NoError(t, err)
	require.Empty(t, resp.Uids)
	require.Equal(t, m["alice"][0].Name, "Alice")

	queryDQL = `{
		alice(func: eq(email, "alice@example.com")) {
		  name
		  email
		  age
		}
	  }`
	resp, err = client.RunDQL(ctx, queryDQL, dgo.WithReadOnly())
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(resp.Json, &m))
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
	resp, err = client.RunDQL(ctx, upsertQuery)
	require.NoError(t, err)
	require.Empty(t, resp.Uids)
	require.Equal(t, m["alice"][0].Name, "Alice Sayum")

	resp, err = client.RunDQL(ctx, queryDQL, dgo.WithReadOnly())
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(resp.Json, &m))
	require.Equal(t, m["alice"][0].Age, 31)
}
