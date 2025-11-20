# dgo [![GoDoc](https://pkg.go.dev/badge/github.com/dgraph-io/dgo)](https://pkg.go.dev/github.com/dgraph-io/dgo/v250)

Official Dgraph Go client which communicates with the server using [gRPC](https://grpc.io/).

Before using this client, we highly recommend that you go through [dgraph.io/tour] and
[dgraph.io/docs] to understand how to run and work with Dgraph.

[dgraph.io/docs]: https://dgraph.io/docs
[dgraph.io/tour]: https://dgraph.io/tour

**Use [Github Issues](https://github.com/dgraph-io/dgo/issues) for reporting issues about this
repository.**

## Table of contents

- [Supported Versions](#supported-versions)
- [New APIs in v25](#new-apis-in-v25)
  - [Connection Strings](#connection-strings)
  - [Advanced Client Creation](#advanced-client-creation)
  - [Dropping All Data](#dropping-all-data)
  - [Set Schema](#set-schema)
  - [Running a Mutation](#running-a-mutation)
  - [Running a Query](#running-a-query)
  - [Running a Query With Variables](#running-a-query-with-variables)
  - [Running a Best Effort Query](#running-a-best-effort-query)
  - [Running a ReadOnly Query](#running-a-readonly-query)
  - [Running a Query with RDF Response](#running-a-query-with-rdf-response)
  - [Running an Upsert](#running-an-upsert)
  - [Running a Conditional Upsert](#running-a-conditional-upsert)
  - [Creating a New Namespace](#creating-a-new-namespace)
  - [Dropping a Namespace](#dropping-a-namespace)
  - [List All Namespaces](#list-all-namespaces)
- [Existing APIs](#existing-apis)
  - [Creating a Client](#creating-a-client)
  - [Login into a namespace](#login-into-a-namespace)
  - [Altering the database](#altering-the-database)
  - [Creating a transaction](#creating-a-transaction)
  - [Running a mutation](#running-a-mutation-1)
  - [Running a query](#running-a-query-1)
  - [Query with RDF response](#query-with-rdf-response)
  - [Running an Upsert: Query + Mutation](#running-an-upsert-query--mutation)
  - [Running Conditional Upsert](#running-conditional-upsert)
  - [Committing a transaction](#committing-a-transaction)
  - [Setting Metadata Headers](#setting-metadata-headers)
- [Development](#development)
  - [Running tests](#running-tests)

## Supported Versions

Depending on the version of Dgraph that you are connecting to, you will have to use a different
version of this client and their corresponding import paths.

| Dgraph version | dgo version | dgo import path                 |
| -------------- | ----------- | ------------------------------- |
| dgraph 23.X.Y  | dgo 230.X.Y | "github.com/dgraph-io/dgo/v230" |
| dgraph 24.X.Y  | dgo 240.X.Y | "github.com/dgraph-io/dgo/v240" |
| dgraph 25.X.Y  | dgo 240.X.Y | "github.com/dgraph-io/dgo/v240" |
| dgraph 25.X.Y  | dgo 250.X.Y | "github.com/dgraph-io/dgo/v250" |

## New APIs in v25

### Connection Strings

The dgo package supports connecting to a Dgraph cluster using connection strings. Dgraph connections
strings take the form `dgraph://{username:password@}host:port?args`.

`username` and `password` are optional. If username is provided, a password must also be present. If
supplied, these credentials are used to log into a Dgraph cluster through the ACL mechanism.

Valid connection string args:

| Arg         | Value                           | Description                                                                                                                                                   |
| ----------- | ------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| apikey      | \<key\>                         | a Dgraph Cloud API Key                                                                                                                                        |
| bearertoken | \<token\>                       | an access token                                                                                                                                               |
| sslmode     | disable \| require \| verify-ca | TLS option, the default is `disable`. If `verify-ca` is set, the TLS certificate configured in the Dgraph cluster must be from a valid certificate authority. |

Some example connection strings:

| Value                                                                                                        | Explanation                                                                         |
| ------------------------------------------------------------------------------------------------------------ | ----------------------------------------------------------------------------------- |
| dgraph://localhost:9080                                                                                      | Connect to localhost, no ACL, no TLS                                                |
| dgraph://sally:supersecret@dg.example.com:443?sslmode=verify-ca                                              | Connect to remote server, use ACL and require TLS and a valid certificate from a CA |
| dgraph://foo-bar.grpc.us-west-2.aws.cloud.dgraph.io:443?sslmode=verify-ca&apikey=\<your-api-connection-key\> | Connect to a Dgraph Cloud cluster                                                   |
| dgraph://foo-bar.grpc.example.com?sslmode=verify-ca&bearertoken=\<some access token\>                        | Connect to a Dgraph cluster protected by a secure gateway                           |

Using the `Open` function with a connection string:

```go
// open a connection to an ACL-enabled, non-TLS cluster and login as groot
client, err := dgo.Open("dgraph://groot:password@localhost:8090")
// Check error
defer client.Close()
// Use the clients
```

### Advanced Client Creation

For more control, you can create a client using the `NewClient` function.

```go
client, err := dgo.NewClient("localhost:9181",
  // add Dgraph ACL credentials
  dgo.WithACLCreds("groot", "password"),
  // add insecure transport credentials
  dgo.WithGrpcOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
)
// Check error
defer client.Close()
// Use the client
```

You can connect to multiple alphas using `NewRoundRobinClient`.

```go
client, err := dgo.NewRoundRobinClient([]string{"localhost:9181", "localhost:9182", "localhost:9183"},
  // add Dgraph ACL credentials
  dgo.WithACLCreds("groot", "password"),
  // add insecure transport credentials
  dgo.WithGrpcOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
)
// Check error
defer client.Close()
// Use the client
```

### Dropping All Data

In order to drop all data in the Dgraph Cluster and start fresh, use the `DropAll` function.

```go
err := client.DropAll(context.TODO())
// Handle error
```

### Set Schema

To set the schema, use the `SetSchema` function.

```go
sch := `
  name: string @index(exact) .
  email: string @index(exact) @unique .
  age: int .
`
err := client.SetSchema(context.TODO(), sch)
// Handle error
```

### Running a Mutation

To run a mutation, use the `RunDQL` function.

```go
mutationDQL := `{
  set {
    _:alice <name> "Alice" .
    _:alice <email> "alice@example.com" .
    _:alice <age> "29" .
  }
}`
resp, err := client.RunDQL(context.TODO(), mutationDQL)
// Handle error
// Print map of blank UIDs
fmt.Printf("%+v\n", resp.Uids)
```

### Running a Query

To run a query, use the same `RunDQL` function.

```go
queryDQL := `{
  alice(func: eq(name, "Alice")) {
    name
    email
    age
  }
}`
resp, err := client.RunDQL(context.TODO(), queryDQL)
// Handle error
fmt.Printf("%s\n", resp.Json)
```

### Running a Query With Variables

To run a query with variables, using `RunDQLWithVars`.

```go
queryDQL = `query Alice($name: string) {
  alice(func: eq(name, $name)) {
    name
    email
    age
  }
}`
vars := map[string]string{"$name": "Alice"}
resp, err := client.RunDQLWithVars(context.TODO(), queryDQL, vars)
// Handle error
fmt.Printf("%s\n", resp.Json)
```

### Running a Best Effort Query

To run a `BestEffort` query, use the same `RunDQL` function with `TxnOption`.

```go
queryDQL := `{
  alice(func: eq(name, "Alice")) {
    name
    email
    age
  }
}`
resp, err := client.RunDQL(context.TODO(), queryDQL, dgo.WithBestEffort())
// Handle error
fmt.Printf("%s\n", resp.Json)
```

### Running a ReadOnly Query

To run a `ReadOnly` query, use the same `RunDQL` function with `TxnOption`.

```go
queryDQL := `{
  alice(func: eq(name, "Alice")) {
    name
    email
    age
  }
}`
resp, err := client.RunDQL(context.TODO(), queryDQL, dgo.WithReadOnly())
// Handle error
fmt.Printf("%s\n", resp.Json)
```

### Running a Query with RDF Response

To get the query response in RDF format instead of JSON format, use the following `TxnOption`.

```go
queryDQL := `{
  alice(func: eq(name, "Alice")) {
    name
    email
    age
  }
}`
resp, err = client.RunDQL(context.TODO(), queryDQL, dgo.WithResponseFormat(api_v2.RespFormat_RDF))
// Handle error
fmt.Printf("%s\n", resp.Rdf)
```

### Running an Upsert

The `RunDQL` function also allows you to run upserts as well.

```go
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
resp, err := client.RunDQL(context.TODO(), upsertQuery)
// Handle error
fmt.Printf("%s\n", resp.Json)
fmt.Printf("%+v\n", resp.Uids)
```

### Running a Conditional Upsert

```go
upsertQuery := `upsert {
  query {
    user as var(func: eq(email, "alice@example.com"))
  }
  mutation @if(eq(len(user), 1)) {
    set {
      uid(user) <age> "30" .
      uid(user) <name> "Alice Sayum" .
    }
  }
}`
resp, err := client.RunDQL(context.TODO(), upsertQuery)
// Handle error
fmt.Printf("%s\n", resp.Json)
```

### Creating a New Namespace

Dgraph v25 supports creating namespaces using grpc API. You can create one using the dgo client.

```go
_, err := client.CreateNamespace(context.TODO())
// Handle error
```

### Dropping a Namespace

To drop a namespace:

```go
err := client.DropNamespace(context.TODO())
// Handle error
```

### List All Namespaces

```go
namespaces, err := client.ListNamespaces(context.TODO())
// Handle error
fmt.Printf("%+v\n", namespaces)
```

## Existing APIs

### Creating a Client

`dgraphClient` object can be initialized by passing it a list of `api.DgraphClient` clients as
variadic arguments. Connecting to multiple Dgraph servers in the same cluster allows for better
distribution of workload.

The following code snippet shows just one connection.

```go
conn, err := grpc.Dial("localhost:9080", grpc.WithInsecure())
// Check error
defer conn.Close()
dgraphClient := dgo.NewDgraphClient(api.NewDgraphClient(conn))
```

### Login into a namespace

If your server has Access Control Lists enabled (Dgraph v1.1 or above), the client must be logged in
for accessing data. If you do not use the `WithACLCreds` option with `NewClient` or a connection
string with username:password, use the `Login` endpoint.

Calling login will obtain and remember the access and refresh JWT tokens. All subsequent operations
via the logged in client will send along the stored access token.

```go
err := dgraphClient.Login(ctx, "user", "passwd")
// Check error
```

If your server additionally has namespaces (Dgraph v21.03 or above), use the `LoginIntoNamespace`
API.

```go
err := dgraphClient.LoginIntoNamespace(ctx, "user", "passwd", 0x10)
// Check error
```

### Altering the database

To set the schema, create an instance of `api.Operation` and use the `Alter` endpoint.

```go
op := &api.Operation{
  Schema: `name: string @index(exact) .`,
}
err := dgraphClient.Alter(ctx, op)
// Check error
```

`Operation` contains other fields as well, including `DropAttr` and `DropAll`. `DropAll` is useful
if you wish to discard all the data, and start from a clean slate, without bringing the instance
down. `DropAttr` is used to drop all the data related to a predicate.

Starting Dgraph version 20.03.0, indexes can be computed in the background. You can set
`RunInBackground` field of the `api.Operation` to `true` before passing it to the `Alter` function.
You can find more details
[here](https://docs.dgraph.io/master/query-language/#indexes-in-background).

```go
op := &api.Operation{
  Schema:          `name: string @index(exact) .`,
  RunInBackground: true
}
err := dgraphClient.Alter(ctx, op)
```

### Creating a transaction

To create a transaction, call `dgraphClient.NewTxn()`, which returns a `*dgo.Txn` object. This
operation incurs no network overhead.

It is a good practice to call `txn.Discard(ctx)` using a `defer` statement after it is initialized.
Calling `txn.Discard(ctx)` after `txn.Commit(ctx)` is a no-op. Furthermore, `txn.Discard(ctx)` can
be called multiple times with no additional side-effects.

```go
txn := dgraphClient.NewTxn()
defer txn.Discard(ctx)
```

Read-only transactions can be created by calling `c.NewReadOnlyTxn()`. Read-only transactions are
useful to increase read speed because they can circumvent the usual consensus protocol. Read-only
transactions cannot contain mutations and trying to call `txn.Commit()` will result in an error.
Calling `txn.Discard()` will be a no-op.

### Running a mutation

`txn.Mutate(ctx, mu)` runs a mutation. It takes in a `context.Context` and a `*api.Mutation` object.
You can set the data using JSON or RDF N-Quad format.

To use JSON, use the fields `SetJson` and `DeleteJson`, which accept a string representing the nodes
to be added or removed respectively (either as a JSON map or a list). To use RDF, use the fields
`SetNquads` and `DelNquads`, which accept a string representing the valid RDF triples (one per line)
to added or removed respectively. This protobuf object also contains the `Set` and `Del` fields
which accept a list of RDF triples that have already been parsed into our internal format. As such,
these fields are mainly used internally and users should use the `SetNquads` and `DelNquads` instead
if they are planning on using RDF.

We define a Person struct to represent a Person and marshal an instance of it to use with `Mutation`
object.

```go
type Person struct {
  Uid   string   `json:"uid,omitempty"`
  Name  string   `json:"name,omitempty"`
  DType []string `json:"dgraph.type,omitempty"`
}

p := Person{
  Uid:   "_:alice",
  Name:  "Alice",
  DType: []string{"Person"},
}

pb, err := json.Marshal(p)
// Check error

mu := &api.Mutation{
  SetJson: pb,
}
res, err := txn.Mutate(ctx, mu)
// Check error
```

For a more complete example, see
[Example](https://pkg.go.dev/github.com/dgraph-io/dgo#example-package-SetObject).

Sometimes, you only want to commit a mutation, without querying anything further. In such cases, you
can use `mu.CommitNow = true` to indicate that the mutation must be immediately committed.

Mutation can be run using `txn.Do` as well.

```go
mu := &api.Mutation{
  SetJson: pb,
}
req := &api.Request{CommitNow:true, Mutations: []*api.Mutation{mu}}
res, err := txn.Do(ctx, req)
// Check error
```

### Running a query

You can run a query by calling `txn.Query(ctx, q)`. You will need to pass in a DQL query string. If
you want to pass an additional map of any variables that you might want to set in the query, call
`txn.QueryWithVars(ctx, q, vars)` with the variables map as third argument.

Let's run the following query with a variable $a:

```go
q := `query all($a: string) {
    all(func: eq(name, $a)) {
      name
    }
  }`

resp, err := txn.QueryWithVars(ctx, q, map[string]string{"$a": "Alice"})
fmt.Printf("%s\n", resp.Json)
```

You can also use `txn.Do` function to run a query.

```go
req := &api.Request{
  Query: q,
  Vars: map[string]string{"$a": "Alice"},
}
resp, err := txn.Do(ctx, req)
// Check error
fmt.Printf("%s\n", resp.Json)
```

When running a schema query for predicate `name`, the schema response is found in the `Json` field
of `api.Response` as shown below:

```go
q := `schema(pred: [name]) {
  type
  index
  reverse
  tokenizer
  list
  count
  upsert
  lang
}`

resp, err := txn.Query(ctx, q)
// Check error
fmt.Printf("%s\n", resp.Json)
```

### Query with RDF response

You can get query result as an RDF response by calling `txn.QueryRDF`. The response would contain a
`Rdf` field, which has the RDF encoded result.

**Note:** If you are querying only for `uid` values, use a JSON format response.

```go
// Query the balance for Alice and Bob.
const q = `
{
  all(func: anyofterms(name, "Alice Bob")) {
    name
    balance
  }
}
`
resp, err := txn.QueryRDF(context.Background(), q)
// check error

// <0x17> <name> "Alice" .
// <0x17> <balance> 100 .
fmt.Println(resp.Rdf)
```

`txn.QueryRDFWithVars` is also available when you need to pass values for variables used in the
query.

### Running an Upsert: Query + Mutation

The `txn.Do` function allows you to run upserts consisting of one query and one mutation. Variables
can be defined in the query and used in the mutation. You could also use `txn.Do` to perform a query
followed by a mutation.

To know more about upsert, we highly recommend going through the docs at
[Upsert Block](https://dgraph.io/docs/dql/dql-syntax/dql-mutation/#upsert-block).

```go
query = `
  query {
      user as var(func: eq(email, "wrong_email@dgraph.io"))
  }`
mu := &api.Mutation{
  SetNquads: []byte(`uid(user) <email> "correct_email@dgraph.io" .`),
}
req := &api.Request{
  Query: query,
  Mutations: []*api.Mutation{mu},
  CommitNow:true,
}

// Update email only if matching uid found.
_, err := dg.NewTxn().Do(ctx, req)
// Check error
```

### Running Conditional Upsert

The upsert block also allows specifying a conditional mutation block using an `@if` directive. The
mutation is executed only when the specified condition is true. If the condition is false, the
mutation is silently ignored.

See more about Conditional Upsert
[Here](https://dgraph.io/docs/dql/dql-syntax/dql-mutation/#conditional-upsert).

```go
query = `
  query {
      user as var(func: eq(email, "wrong_email@dgraph.io"))
  }`
mu := &api.Mutation{
  Cond: `@if(eq(len(user), 1))`, // Only mutate if "wrong_email@dgraph.io" belongs to single user.
  SetNquads: []byte(`uid(user) <email> "correct_email@dgraph.io" .`),
}
req := &api.Request{
  Query: query,
  Mutations: []*api.Mutation{mu},
  CommitNow:true,
}

// Update email only if exactly one matching uid is found.
_, err := dg.NewTxn().Do(ctx, req)
// Check error
```

### Committing a transaction

A transaction can be committed using the `txn.Commit(ctx)` method. If your transaction consisted
solely of calls to `txn.Query` or `txn.QueryWithVars`, and no calls to `txn.Mutate`, then calling
`txn.Commit` is not necessary.

An error will be returned if other transactions running concurrently modify the same data that was
modified in this transaction. It is up to the user to retry transactions when they fail.

```go
txn := dgraphClient.NewTxn()
// Perform some queries and mutations.

err := txn.Commit(ctx)
if err == y.ErrAborted {
  // Retry or handle error
}
```

### Setting Metadata Headers

Metadata headers such as authentication tokens can be set through the context of gRPC methods. Below
is an example of how to set a header named "auth-token".

```go
// The following piece of code shows how one can set metadata with
// auth-token, to allow Alter operation, if the server requires it.
md := metadata.New(nil)
md.Append("auth-token", "the-auth-token-value")
ctx := metadata.NewOutgoingContext(context.Background(), md)
dg.Alter(ctx, &op)
```

## Development

### Running tests

Make sure you have `dgraph` installed in your GOPATH before you run the tests. The dgo test suite
requires that a Dgraph cluster with ACL enabled be running locally. To start such a cluster, you may
use the docker compose file located in the testing directory `t`.

```sh
docker compose -f t/docker-compose.yml up -d
# wait for cluster to be healthy
go test -v ./...
docker compose -f t/docker-compose.yml down
```
