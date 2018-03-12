#!/bin/bash

dgo_scripts=$GOPATH/src/github.com/dgraph-io/dgo/contrib/scripts

$dgo_scripts/install_dgraph.sh
$dgo_scripts/check_dep.sh
$dgo_scripts/transaction.sh
