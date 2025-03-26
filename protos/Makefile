#
# SPDX-FileCopyrightText: © Hypermode Inc. <hello@hypermode.com>
# SPDX-License-Identifier: Apache-2.0
#

.PHONY: clean
clean:
	@rm -rf api && mkdir -p api

.PHONY: check
check:
	echo "Installing proto libraries to versions in go.mod." ; \
		go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.31.0 ; \
		go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.3.0

.PHONY: regenerate
regenerate: check clean
	@protoc --go_out=api --go-grpc_out=api --go_opt=paths=source_relative --go-grpc_opt=paths=source_relative api.proto
	@echo Done.
