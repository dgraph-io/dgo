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


