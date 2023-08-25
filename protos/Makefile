#
# Copyright 2023 Dgraph Labs, Inc. and Contributors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#

.PHONY: clean
clean:
	@mkdir -p api && rm -f api/api.pb.go

.PHONY: check
check:
	echo "Installing proto libraries to versions in go.mod." ; \
		go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.31.0 ; \
		go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.3.0

.PHONY: regenerate
regenerate: check clean
	@protoc --go_out=api --go-grpc_out=api --go_opt=paths=source_relative --go-grpc_opt=paths=source_relative api.proto
	@echo Done.
