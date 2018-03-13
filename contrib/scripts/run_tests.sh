#!/bin/bash

dgo=$GOPATH/src/github.com/dgraph-io/dgo
dgo_scripts=$dgo/contrib/scripts

pushd $dgo
go test -v .
popd

$dgo_scripts/check_dep.sh
$dgo_scripts/transaction.sh
