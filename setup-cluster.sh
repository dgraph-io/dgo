#!/usr/bin/env bash
if [ -z $GOPATH ]; then
    echo "Error: the GOPATH environment variable is not set"; exit 1
else
    cd $GOPATH/src/github.com/dgraph-io/dgraph/dgraph; ./run.sh
fi