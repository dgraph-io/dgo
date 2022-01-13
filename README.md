# dgo [![GoDoc](https://godoc.org/github.com/dgraph-io/dgo?status.svg)](https://godoc.org/github.com/dgraph-io/dgo) [![Build Status](https://teamcity.dgraph.io/guestAuth/app/rest/builds/buildType:(id:dgo_integration)/statusIcon.svg)](https://teamcity.dgraph.io/viewLog.html?buildTypeId=dgo_integration&buildId=lastFinished&guest=1)

Official Dgraph Go client which communicates with the server using [gRPC](https://grpc.io/).

Before using this client, we highly recommend that you go through [tour.dgraph.io] and [docs.dgraph.io]
to understand how to run and work with Dgraph.

**Use [Discuss Issues](https://discuss.dgraph.io/c/issues/35/dgo/46) for reporting issues about this repository.**

[docs.dgraph.io]:https://docs.dgraph.io
[tour.dgraph.io]:https://tour.dgraph.io


## Table of contents

- [Supported Versions](#supported-versions)
- [Using a client](#using-a-client)
  - [Creating a client](#creating-a-client)
  - [Altering the database](#altering-the-database)
  - [Creating a transaction](#creating-a-transaction)
  - [Running a mutation](#running-a-mutation)
  - [Running a query](#running-a-query)
  - [Running an Upsert: Query + Mutation](#running-an-upsert-query--mutation)
  - [Running Conditional Upsert](#running-conditional-upsert)
  - [Committing a transaction](#committing-a-transaction)
  - [Setting Metadata Headers](#setting-metadata-headers)
  - [Connecting To Slash Endpoint](#connecting-to-slash-endpoint)
- [Development](#development)
  - [Running tests](#running-tests)

## Supported Versions

Depending on the version of Dgraph that you are connecting to, you will have to
use a different version of this client and their corresponding import paths.

Dgraph version   | dgo version   |        dgo import path          |
---------------  | -----------   | ------------------------------- |
  dgraph 1.0.X   |  dgo 1.X.Y    |   "github.com/dgraph-io/dgo"    |
  dgraph 1.1.X   |  dgo 2.X.Y    | "github.com/dgraph-io/dgo/v2"   |
  dgraph 20.03.0 |  dgo 200.03.0 | "github.com/dgraph-io/dgo/v200" |
  dgraph 20.07.0 |  dgo 200.03.0 | "github.com/dgraph-io/dgo/v200" |
  dgraph 20.11.0 |  dgo 200.03.0 | "github.com/dgraph-io/dgo/v200" |
  dgraph 21.03.0 |  dgo 210.03.0 | "github.com/dgraph-io/dgo/v210" |

Note: One of the most important API breakages from dgo v1 to v2 is in
the function `dgo.Txn.Mutate`. This function returns an `*api.Assigned`
value in v1 but an `*api.Response` in v2.

Note: There is no breaking API change from v2 to v200 but we have decided
to follow the [CalVer Versioning Scheme](https://dgraph.io/blog/post/dgraph-calendar-versioning).

## Using a client

### Creating a client

`dgraphClient` object can be initialized by passing it a list of `api.DgraphClient` clients as
variadic arguments. Connecting to multiple Dgraph servers in the same cluster allows for better
distribution of workload.

The following code snippet shows just one connection.

```go
conn, err := grpc.Dial("localhost:9080", grpc.WithInsecure())
if err != nil {
  log.Fatal(err)
}
defer conn.Close()
dgraphClient := dgo.NewDgraphClient(api.NewDgraphClient(conn))
```

### Login into a namespace

If your server has Access Control Lists enabled (Dgraph v1.1 or above), the client must be 
logged in for accessing data. Use `Login` endpoint:

Calling login will obtain and remember the access and refresh JWT tokens. All subsequent operations 
via the logged in client will send along the stored access token.

```go
err := dgraphClient.Login(ctx, "user", "passwd")
// Check error
```

If your server additionally has namespaces (Dgraph v21.03 or above), use the 
`LoginIntoNamespace` API.

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

`Operation` contains other fields as well, including `DropAttr` and `DropAll`.
`DropAll` is useful if you wish to discard all the data, and start from a clean
slate, without bringing the instance down. `DropAttr` is used to drop all the data
related to a predicate.

Starting Dgraph version 20.03.0, indexes can be computed in the background.
You can set `RunInBackground` field of the `api.Operation` to `true`
before passing it to the `Alter` function. You can find more details
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
Calling `txn.Discard(ctx)` after `txn.Commit(ctx)` is a no-op. Furthermore, `txn.Discard(ctx)`
can be called multiple times with no additional side-effects.

```go
txn := dgraphClient.NewTxn()
defer txn.Discard(ctx)
```

Read-only transactions can be created by calling `c.NewReadOnlyTxn()`. Read-only
transactions are useful to increase read speed because they can circumvent the
usual consensus protocol. Read-only transactions cannot contain mutations and
trying to call `txn.Commit()` will result in an error. Calling `txn.Discard()`
will be a no-op.

### Running a mutation

`txn.Mutate(ctx, mu)` runs a mutation. It takes in a `context.Context` and a
`*api.Mutation` object. You can set the data using JSON or RDF N-Quad format.

To use JSON, use the fields SetJson and DeleteJson, which accept a string
representing the nodes to be added or removed respectively (either as a JSON map
or a list). To use RDF, use the fields SetNquads and DelNquads, which accept
a string representing the valid RDF triples (one per line) to added or removed
respectively. This protobuf object also contains the Set and Del fields which
accept a list of RDF triples that have already been parsed into our internal
format. As such, these fields are mainly used internally and users should use
the SetNquads and DelNquads instead if they are planning on using RDF.

We define a Person struct to represent a Person and marshal an instance of it to
use with `Mutation` object.

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
if err != nil {
	log.Fatal(err)
}

mu := &api.Mutation{
	SetJson: pb,
}
res, err := txn.Mutate(ctx, mu)
if err != nil {
	log.Fatal(err)
}
```

For a more complete example, see
[GoDoc](https://godoc.org/github.com/dgraph-io/dgo#example-package--SetObject).

Sometimes, you only want to commit a mutation, without querying anything
further. In such cases, you can use `mu.CommitNow = true` to indicate that the
mutation must be immediately committed.

Mutation can be run using `txn.Do` as well.

```go
mu := &api.Mutation{
  SetJson: pb,
}
req := &api.Request{CommitNow:true, Mutations: []*api.Mutation{mu}}
res, err := txn.Do(ctx, req)
if err != nil {
  log.Fatal(err)
}
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

res, err := txn.QueryWithVars(ctx, q, map[string]string{"$a": "Alice"})
fmt.Printf("%s\n", res.Json)
```

You can also use `txn.Do` function to run a query.

```go
req := &api.Request{
  Query: q,
  Vars: map[string]string{"$a": "Alice"},
}
res, err := txn.Do(ctx, req)
if err != nil {
  log.Fatal(err)
}
fmt.Printf("%s\n", res.Json)
```

When running a schema query for predicate `name`, the schema response is found
in the `Json` field of `api.Response` as shown below:

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

res, err := txn.Query(ctx, q)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("%s\n", res.Json)
```

### Running an Upsert: Query + Mutation

The `txn.Do` function allows you to run upserts consisting of one query and
one mutation. Variables can be defined in the query and used in the mutation.
You could also use `txn.Do` to perform a query followed by a mutation.

To know more about upsert, we highly recommend going through the docs
at https://docs.dgraph.io/mutations/#upsert-block.

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
if _, err := dg.NewTxn().Do(ctx, req); err != nil {
  log.Fatal(err)
}
```

### Running Conditional Upsert

The upsert block also allows specifying a conditional mutation block using an `@if` directive.
The mutation is executed only when the specified condition is true. If the condition is false,
the mutation is silently ignored.

See more about Conditional Upsert [Here](https://docs.dgraph.io/mutations/#conditional-upsert).

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
if _, err := dg.NewTxn().Do(ctx, req); err != nil {
  log.Fatal(err)
}
```

### Committing a transaction

A transaction can be committed using the `txn.Commit(ctx)` method. If your transaction
consisted solely of calls to `txn.Query` or `txn.QueryWithVars`, and no calls to
`txn.Mutate`, then calling `txn.Commit` is not necessary.

An error will be returned if other transactions running concurrently modify the same
data that was modified in this transaction. It is up to the user to retry
transactions when they fail.

```go
txn := dgraphClient.NewTxn()
// Perform some queries and mutations.

err := txn.Commit(ctx)
if err == y.ErrAborted {
  // Retry or handle error
}
```

### Setting Metadata Headers
Metadata headers such as authentication tokens can be set through the context of gRPC methods. Below is an example of how to set a header named "auth-token".
```go
// The following piece of code shows how one can set metadata with
// auth-token, to allow Alter operation, if the server requires it.
md := metadata.New(nil)
md.Append("auth-token", "the-auth-token-value")
ctx := metadata.NewOutgoingContext(context.Background(), md)
dg.Alter(ctx, &op)
```

### Connecting To Slash Endpoint

Please use the following snippet to connect to a Slash GraphQL or Slash Enterprise backend.

```go
conn, err := dgo.DialSlashEndpoint("https://your.endpoint.dgraph.io/graphql", "api-token")
if err != nil {
  log.Fatal(err)
}
defer conn.Close()
dgraphClient := dgo.NewDgraphClient(api.NewDgraphClient(conn))
```

## Development

### Running tests

Make sure you have `dgraph` installed before you run the tests. This script will run the unit and
integration tests.

```sh
go test -v ./...
```
