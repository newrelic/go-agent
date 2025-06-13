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

# Test targets
.PHONY: tidy
tidy:
	@cd $(MODULE_DIR); $(GO) mod edit -replace github.com/newrelic/go-agent/v3="$(BASEDIR)/$(MODULE_DIR)"; $(GO) mod tidy

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
		$(GO) test -coverprofile=coverage.txt || exit 1; \
	else \
		$(GO) test || exit 1; \
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
	@echo; echo "# TEST=$(TEST), COVERAGE=$(COVERAGE)"; \
	cd $(MODULE_DIR)/integrations/$(TEST); \
	WD=$(shell pwd); \
	$(GO) mod edit -replace github.com/newrelic/go-agent/v3="$${WD}/${MODULE_DIR}";\
	if [ "$(TEST)" == "nrnats" ]; then \
		GOPROXY=direct $(GO) mod tidy; \
	else \
		$(GO) mod tidy; \
	fi; \
	if [ "$(COVERAGE)" == "1" ]; then \
		$(GO) test -coverprofile=coverage.txt -race -benchtime=$(BENCHTIME) -bench=. ./... || exit 1; \
	else \
		$(GO) test -race -benchtime=$(BENCHTIME) -bench=. ./... || exit 1; \
	fi; \
	$(GO) vet ./... || exit 1; \
	echo "# TEST=$(TEST)"; \
	cd $(BASEDIR);

.PHONY: integration-suite
integration-suite:
	@for TEST in $(GO_INTEGRATION_TESTS); do \
		$(MAKE) integration-test TEST=$${TEST} COVERAGE=$(COVERAGE); \
	done

test-services-start:
	@if [ ! -z $(PACKAGE) ]; then \
		docker compose --profile test --profile $(PACKAGE) pull $(SERVICES); \
		docker compose --profile test --profile $(PACKAGE) up --wait --remove-orphans -d $(SERVICES); \
	else \
		docker compose --profile test pull $(SERVICES); \
		docker compose --profile test up --wait --remove-orphans -d $(SERVICES); \
	fi;

test-services-stop:
	@if [ ! -z $(PACKAGE) ]; then \
		docker compose --profile test --profile $(PACKAGE) stop; \
	else \
		docker compose --profile test stop; \
	fi;

# Developer targets
devenv-image:
	@docker compose --profile dev build devenv

dev-shell: devenv-image
	docker compose --profile dev up --pull missing --remove-orphans -d
	docker compose exec -it devenv bash -c "bash"

dev-stop:
	docker compose --profile dev stop

# Utility targets
.PHONY: integration-to-json
integration-to-json:
	@TESTS="$(shell echo $(GO_INTEGRATION_TESTS))"; \
		echo $$TESTS | jq -R 'split(" ")' | sed "s/ //g" | tr -d '\n';

.PHONY: core-to-json
core-to-json:
	@TESTS="$(shell echo $(GO_CORE_TESTS))"; \
		echo $$TESTS | jq -R 'split(" ")' | sed "s/ //g" | tr -d '\n';

.PHONY: info
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
