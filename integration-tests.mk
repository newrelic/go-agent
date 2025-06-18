#
# Copyright 2025 New Relic Corporation. All rights reserved.
# SPDX-License-Identifier: Apache-2.0
#

# Integration tests for the highest supported Go Version
GO_INTEGRATION_TESTS=$${INTEGRATION_TESTS:-\
	nramqp \
	nrfasthttp \
	nrsarama \
	logcontext/nrlogrusplugin \
	logcontext-v2/nrlogrus \
	logcontext-v2/nrzerolog \
	logcontext-v2/nrzap \
	logcontext-v2/nrslog \
	logcontext-v2/nrwriter \
	logcontext-v2/zerologWriter \
	logcontext-v2/logWriter \
	nrawssdk-v1 \
	nrawssdk-v2 \
	nrecho-v3 \
	nrecho-v4 \
	nrelasticsearch-v7 \
	nrgin \
	nrfiber \
	nrgorilla \
	nrgraphgophers \
	nrlogrus \
	nrlogxi \
	nrpkgerrors \
	nrlambda \
	nrmysql \
	nrpq \
	nrpgx5 \
	nrpq/example/sqlx \
	nrredis-v7 \
	nrredis-v9 \
	nrsqlite3 \
	nrsnowflake \
	nrgrpc \
	nrmicro \
	nrnats \
	nrstan \
	nrstan/test \
	nrstan/examples \
	logcontext \
	nrzap \
	nrhttprouter \
	nrb3 \
	nrmongo \
	nrgraphqlgo \
	nrgraphqlgo/example \
	nrmssql \
	nropenai \
	nrslog \
	nrgochi \
}
