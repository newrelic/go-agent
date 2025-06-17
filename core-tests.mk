#
# Copyright 2025 New Relic Corporation. All rights reserved.
# SPDX-License-Identifier: Apache-2.0
#

# Core Tests on 3 most recent major Go versions
GO_CORE_TESTS=$${CORE_TESTS:-\
	newrelic \
	newrelic/integrationsupport \
	internal \
	internal/awssupport \
	internal/cat \
	internal/com_newrelic_trace_v1 \
	internal/crossagent \
	internal/jsonx \
	internal/logcontext \
	internal/logger \
	internal/stacktracetest \
	internal/sysinfo \
	internal/utilization \
}
