#!/bin/bash

# You might need to go get -v github.com/gogo/protobuf/...

dgraph_io=${GOPATH-$HOME/go}/src/github.com/dgraph-io
protos=$dgraph_io/dgo/protos
pushd $protos > /dev/null
protoc -I=. --gogofaster_out=plugins=grpc:api api.proto
