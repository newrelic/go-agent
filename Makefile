#
# Copyright 2025 New Relic Corporation. All rights reserved.
# SPDX-License-Identifier: Apache-2.0
#

#
# The top level Makefile
#
SHELL = /bin/bash
GIT   ?= git

# Default go invocation command
GO := go

# module path
GO_MODULE := github.com/newrelic/go-agent/v3/newrelic
BASEDIR   := $(PWD)
MODULE_DIR:= ./v3
BENCHTIME := 1ms

# Include the secrets file if it exists, but if it doesn't, that's OK too.
-include make/secrets.mk

# Test targets
.PHONY: tidy
tidy:
	@cd $(MODULE_DIR); $(GO) mod edit -replace github.com/newrelic/go-agent/v3="$(BASEDIR)/v3"; $(GO) mod tidy

.PHONY: bench
bench: tidy
	@cd $(MODULE_DIR); $(GO) test -race -benchtime=$(BENCHTIME) -bench=. ./...

.PHONY: test
test: tidy
	@cd $(MODULE_DIR); $(GO) test ./...

.PHONY: vet
vet: tidy
	@cd $(MODULE_DIR); $(GO) vet ./...

.PHONY: format
format: tidy
	@cd $(MODULE_DIR); $(GO) fmt ./...

test-services-start:
	docker compose --profile test pull $(SERVICES)
	docker compose --profile test up --wait --remove-orphans -d $(SERVICES)

test-services-stop:
	docker compose --profile test stop

# Developer targets
devenv-image:
	@docker compose --profile dev build devenv

dev-shell: devenv-image
	docker compose --profile dev up --pull missing --remove-orphans -d
	docker compose exec -it devenv bash -c "bash"

dev-stop:
	docker compose --profile dev stop
