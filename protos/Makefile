#
#  Copyright 2018 Dgraph Labs, Inc.
#
#  This file is available under the Apache License, Version 2.0,
#  with the Commons Clause restriction.
#

.PHONY: help
help:
	@echo Use \"make regenerate\" to proceed.

clean:
	@mkdir -p api && rm -f api/api.pb.go

regenerate: clean
	@protoc \
		--go_out=plugins=grpc:api \
		--go_opt=paths=source_relative \
		-I=. \
		api.proto
	@echo Done.
