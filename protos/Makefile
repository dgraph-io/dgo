#
# SPDX-FileCopyrightText: © 2017-2025 Istari Digital, Inc.
# SPDX-License-Identifier: Apache-2.0
#

.PHONY: clean
clean:
	@rm -rf api && mkdir -p api
	@rm -rf api.v2 && mkdir -p api.v2

UNAME_S := $(shell uname -s)

.PHONY: check
check:
	echo "Installing proto libraries to versions in go.mod." ; \
		go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.0 ; \
		go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.4.0

ifneq ($(UNAME_S),Linux)
.PHONY: regenerate
regenerate:
	@echo "Running in Docker..."
	@docker run --rm -v $(shell pwd)/..:/go/src/github.com/dgraph-io/dgo \
		-w /go/src/github.com/dgraph-io/dgo/protos \
		golang:1.23.0 \
		/bin/bash -c "apt-get update && apt-get install -y protobuf-compiler && make regenerate"
else
.PHONY: regenerate
regenerate: check clean
	@protoc --go_out=api --go-grpc_out=api --go_opt=paths=source_relative --go-grpc_opt=paths=source_relative api.proto
	@echo Done.
endif
