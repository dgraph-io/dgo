#
# SPDX-FileCopyrightText: © Hypermode Inc. <hello@hypermode.com>
# SPDX-License-Identifier: Apache-2.0
#

patchContent := "package api\n\nimport (\n\tgrpc \"google.golang.org/grpc\"\n)\n\nfunc GetConn(c DgraphClient) grpc.ClientConnInterface {\n\treturn c.(*dgraphClient).cc\n}"

.PHONY: clean
clean:
	@rm -rf api && mkdir -p api
	@rm -rf api.v2 && mkdir -p api.v2

.PHONY: check
check:
	echo "Installing proto libraries to versions in go.mod." ; \
		go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.0 ; \
		go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.4.0

.PHONY: regenerate
regenerate: check clean
	@protoc --go_out=api --go-grpc_out=api --go_opt=paths=source_relative --go-grpc_opt=paths=source_relative api.proto
	@protoc --go_out=api.v2 --go-grpc_out=api.v2 --go_opt=paths=source_relative --go-grpc_opt=paths=source_relative api.v2.proto
	@echo $(patchContent) > api/cc.go
	@echo Done.
