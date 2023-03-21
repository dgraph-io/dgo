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
		go install github.com/golang/protobuf/protoc-gen-go ; \
		go install github.com/gogo/protobuf/protoc-gen-gogofaster

.PHONY: regenerate
regenerate: check clean
	@protoc --gogofaster_out=plugins=grpc:api api.proto
	@echo Done.
