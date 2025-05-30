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
-include secrets.mk

# Include the manifests for the integration and core tests
include integration-tests.mk
include core-tests.mk

info:
	@echo
	@echo "$$(go version)"
	@echo
	@echo "Integration Tests:"
	@echo $(GO_INTEGRATION_TESTS)
	@echo
	@echo "Core Tests:"
	@echo $(GO_CORE_TESTS)
	@echo

# Test targets
.PHONY: tidy
tidy:
	@cd $(MODULE_DIR); $(GO) mod edit -replace github.com/newrelic/go-agent/v3="$(BASEDIR)/$(MODULE_DIR)"; $(GO) mod tidy

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

.PHONY: core-test
core-test:
	@echo; echo "# TEST=$(TEST), COVERAGE=$(COVERAGE)"; \
	cd $(MODULE_DIR)/$(TEST); \
	$(GO) mod edit -replace github.com/newrelic/go-agent/v3="$(BASEDIR)/$(MODULE_DIR)"; \
	$(GO) mod tidy; \
	if [ "$(COVERAGE)" == "1" ]; then \
		$(GO) test -coverprofile=coverage.txt; \
	else \
		$(GO) test; \
	fi; \
	echo "# TEST=$(TEST)"; \
	cd $(BASEDIR);

.PHONY: core-suite
core-suite:
	@for TEST in $(GO_CORE_TESTS); do \
		$(MAKE) core-test TEST=$${TEST} COVERAGE=$(COVERAGE); \
	done

.PHONY: integration-test
integration-test:
	@echo; echo "# TEST=$(TEST)"; \
	cd $(MODULE_DIR)/integrations/$(TEST); \
	$(GO) mod edit -replace github.com/newrelic/go-agent/v3="$(BASEDIR)/$(MODULE_DIR)";\
	$(GO) mod tidy; \
	$(GO) test -race -benchtime=$(BENCHTIME) -bench=. ./...; \
	$(GO) vet ./...; \
	echo "# TEST=$(TEST)"; \
	cd $(BASEDIR);

.PHONY: integration-suite
integration-suite:
	@for TEST in $(GO_INTEGRATION_TESTS); do \
		$(MAKE) integration-test TEST=$${TEST}; \
	done

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
