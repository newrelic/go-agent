#
# Copyright 2025 New Relic Corporation. All rights reserved.
# SPDX-License-Identifier: Apache-2.0
#

# Integration tests for the highest supported Go Version
GO_INTEGRATION_TESTS=$${INTEGRATION_TESTS:-\
	logcontext/nrlogrusplugin \
	logcontext-v2/logWriter \
	logcontext-v2/nrlogrus \
	logcontext-v2/nrslog \
	logcontext-v2/nrwriter \
	logcontext-v2/nrzap \
	logcontext-v2/nrzerolog \
	logcontext-v2/zerologWriter \
  	nramqp \
  	nrawsbedrock \
	nrawssdk-v1 \
	nrawssdk-v2 \
	nrb3 \
	nrecho-v3 \
	nrecho-v4 \
	nrelasticsearch-v7 \
  	nrfasthttp \
	nrfiber \
	nrfiber/example \
	nrgin \
	nrgochi \
	nrgorilla \
	nrgraphgophers \
	nrgraphqlgo \
	nrgrpc \
	nrhttprouter \
	nrlambda \
	nrlogrus \
	nrlogxi \
	nrmicro \
	nrmongo \
	nrmongo-v2 \
	nrmssql \
	nrmysql \
	nrnats \
	nropenai \
	nrpgx \
	nrpgx5 \
  	nrpkgerrors \
	nrpq \
	nrredis-v7 \
	nrredis-v8 \
	nrredis-v9 \
	nrsarama \
	nrsecurityagent \
	nrslog \
	nrsnowflake \
	nrsqlite3 \
	nrstan/test \
	nrzap \
	nrzerolog \
}
