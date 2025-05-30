#
# Copyright 2025 New Relic Corporation. All rights reserved.
# SPDX-License-Identifier: Apache-2.0
#

# Core Tests on 3 most recent major Go versions
GO_CORE_TESTS=$${CORE_TESTS:-\
	newrelic \
	internal \
}
