# dgo [![GoDoc](https://godoc.org/github.com/dgraph-io/dgo?status.svg)](https://godoc.org/github.com/dgraph-io/dgo)

Official Dgraph Go client which communicates with the server using [gRPC](https://grpc.io/).

Before using this client, we highly recommend that you go through [docs.dgraph.io],
and understand how to run and work with Dgraph.

[docs.dgraph.io]:https://docs.dgraph.io

## Table of contents

- [Install](#install)
- [Quickstart](#quickstart)
- [Using a client](#using-a-client)
  - [Create a client](#create-a-client)
  - [Alter the database](#alter-the-database)
  - [Create a transaction](#create-a-transaction)
  - [Run a mutation](#run-a-mutation)
  - [Run a query](#run-a-query)
  - [Commit a transaction](#commit-a-transaction)
  - [Cleanup Resources](#cleanup-resources)
  - [Debug mode](#debug-mode)
- [Development](#development)
  - [Building the source](#building-the-source)
  - [Running tests](#running-tests)

## Install

```sh
go get -u -v github.com/dgraph-io/dgo
```

## Using a client

### Create a client

```go
	conn, err := grpc.Dial("localhost:9080", grpc.WithInsecure())
	if err != nil {
		log.Fatal(err)
	}
	dgraphClient := dgo.NewDgraphClient(api.NewDgraphClient(conn))
```

### Alter the database

To set the schema, using the `Alter` endpoint.

```go
	op := &api.Operation{
		Schema: `name: string @index(exact) .`,
	}
	err := dgraphClient.Alter(context.Background(), op)
	// Check error

```

`Operation` contains other fields as well, including drop predicate and drop all.
Drop all is useful if you wish to discard all the data, and start from a clean
slate, without bringing the instance down.
