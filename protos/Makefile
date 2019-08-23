#
#  Copyright 2018 Dgraph Labs, Inc.
#
#  This file is available under the Apache License, Version 2.0,
#  with the Commons Clause restriction.
#

PROTO_PATH := ${GOPATH}/src:.
PROTO_PATH := ${PROTO_PATH}:${GOPATH}/src/github.com/gogo/protobuf

.PHONY: help
help:
	@echo Use \"make regenerate\" to proceed.

clean:
	@mkdir -p api && rm -f api/api.pb.go

regenerate: clean
	@protoc \
		--proto_path=${PROTO_PATH} \
		--gofast_out=plugins=grpc:api \
		api.proto
	@echo Done.
