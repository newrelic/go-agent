// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package nrelasticsearch

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"testing"

	elasticsearch "github.com/elastic/go-elasticsearch/v7"
	"github.com/elastic/go-elasticsearch/v7/esapi"
	"github.com/newrelic/go-agent/v3/internal"
	"github.com/newrelic/go-agent/v3/internal/integrationsupport"
	newrelic "github.com/newrelic/go-agent/v3/newrelic"
)

func TestParseRequest(t *testing.T) {
	testcases := []struct {
		// Input
		Method string
		Path   string
		// Expect
		Collection string
		Operation  string
	}{
		// These index operations are not worth worrying about.  They
		// are only going to be used in db setup.
		{Method: "DELETE", Path: "/{index}/{type}/{id}", Collection: "", Operation: ""},
		{Method: "HEAD", Path: "/{index}/{type}/{id}", Collection: "", Operation: ""},
		{Method: "GET", Path: "/{index}/{type}/{id}", Collection: "", Operation: ""},
		{Method: "POST", Path: "/{index}/{type}", Collection: "", Operation: ""},
		{Method: "POST", Path: "/{index}/{type}/{id}", Collection: "", Operation: ""},
		{Method: "PUT", Path: "/{index}/{type}/{id}", Collection: "", Operation: ""},
		{Method: "PUT", Path: "/{index}", Collection: "", Operation: ""},
		{Method: "DELETE", Path: "/{index}", Collection: "", Operation: ""},
		{Method: "HEAD", Path: "/{index}", Collection: "", Operation: ""},
		{Method: "GET", Path: "/{index}", Collection: "", Operation: ""},

		{Method: "GET", Path: "/", Collection: "", Operation: "info"},
		{Method: "HEAD", Path: "/", Collection: "", Operation: "ping"},

		{Method: "DELETE", Path: "/{index}/_alias/{name}", Collection: "{index}", Operation: "alias"},
		{Method: "HEAD", Path: "/_alias/{name}", Operation: "alias"},
		{Method: "HEAD", Path: "/{index}/_alias/{name}", Collection: "{index}", Operation: "alias"},
		{Method: "GET", Path: "/_alias", Operation: "alias"},
		{Method: "GET", Path: "/_alias/{name}", Operation: "alias"},
		{Method: "GET", Path: "/{index}/_alias", Collection: "{index}", Operation: "alias"},
		{Method: "GET", Path: "/{index}/_alias/{name}", Collection: "{index}", Operation: "alias"},
		{Method: "POST", Path: "/{index}/_alias/{name}", Collection: "{index}", Operation: "alias"},
		{Method: "PUT", Path: "/{index}/_alias/{name}", Collection: "{index}", Operation: "alias"},
		{Method: "DELETE", Path: "/{index}/_aliases/{name}", Collection: "{index}", Operation: "aliases"},
		{Method: "POST", Path: "/{index}/_aliases/{name}", Collection: "{index}", Operation: "aliases"},
		{Method: "PUT", Path: "/{index}/_aliases/{name}", Collection: "{index}", Operation: "aliases"},
		{Method: "POST", Path: "/_aliases", Operation: "aliases"},
		{Method: "GET", Path: "/_analyze", Operation: "analyze"},
		{Method: "GET", Path: "/{index}/_analyze", Collection: "{index}", Operation: "analyze"},
		{Method: "POST", Path: "/_analyze", Operation: "analyze"},
		{Method: "POST", Path: "/{index}/_analyze", Collection: "{index}", Operation: "analyze"},
		{Method: "POST", Path: "/_bulk", Operation: "bulk"},
		{Method: "POST", Path: "/{index}/_bulk", Collection: "{index}", Operation: "bulk"},
		{Method: "POST", Path: "/{index}/{type}/_bulk", Collection: "{index}", Operation: "bulk"},
		{Method: "PUT", Path: "/_bulk", Operation: "bulk"},
		{Method: "PUT", Path: "/{index}/_bulk", Collection: "{index}", Operation: "bulk"},
		{Method: "PUT", Path: "/{index}/{type}/_bulk", Collection: "{index}", Operation: "bulk"},
		{Method: "POST", Path: "/_cache/clear", Operation: "cache"},
		{Method: "POST", Path: "/{index}/_cache/clear", Collection: "{index}", Operation: "cache"},
		{Method: "GET", Path: "/_cat/aliases", Operation: "cat"},
		{Method: "GET", Path: "/_cat/aliases/{name}", Operation: "cat"},
		{Method: "GET", Path: "/_cat/allocation", Operation: "cat"},
		{Method: "GET", Path: "/_cat/allocation/{node_id}", Operation: "cat"},
		{Method: "GET", Path: "/_cat/count", Operation: "cat"},
		{Method: "GET", Path: "/_cat/count/{index}", Collection: "", Operation: "cat"},
		{Method: "GET", Path: "/_cat/fielddata", Operation: "cat"},
		{Method: "GET", Path: "/_cat/fielddata/{fields}", Operation: "cat"},
		{Method: "GET", Path: "/_cat/health", Operation: "cat"},
		{Method: "GET", Path: "/_cat", Operation: "cat"},
		{Method: "GET", Path: "/_cat/indices", Operation: "cat"},
		{Method: "GET", Path: "/_cat/indices/{index}", Collection: "", Operation: "cat"},
		{Method: "GET", Path: "/_cat/master", Operation: "cat"},
		{Method: "GET", Path: "/_cat/nodeattrs", Operation: "cat"},
		{Method: "GET", Path: "/_cat/nodes", Operation: "cat"},
		{Method: "GET", Path: "/_cat/pending_tasks", Operation: "cat"},
		{Method: "GET", Path: "/_cat/plugins", Operation: "cat"},
		{Method: "GET", Path: "/_cat/recovery", Operation: "cat"},
		{Method: "GET", Path: "/_cat/recovery/{index}", Collection: "", Operation: "cat"},
		{Method: "GET", Path: "/_cat/repositories", Operation: "cat"},
		{Method: "GET", Path: "/_cat/segments", Operation: "cat"},
		{Method: "GET", Path: "/_cat/segments/{index}", Collection: "", Operation: "cat"},
		{Method: "GET", Path: "/_cat/shards", Operation: "cat"},
		{Method: "GET", Path: "/_cat/shards/{index}", Collection: "", Operation: "cat"},
		{Method: "GET", Path: "/_cat/snapshots", Operation: "cat"},
		{Method: "GET", Path: "/_cat/snapshots/{repository}", Operation: "cat"},
		{Method: "GET", Path: "/_cat/tasks", Operation: "cat"},
		{Method: "GET", Path: "/_cat/templates", Operation: "cat"},
		{Method: "GET", Path: "/_cat/templates/{name}", Operation: "cat"},
		{Method: "GET", Path: "/_cat/thread_pool", Operation: "cat"},
		{Method: "GET", Path: "/_cat/thread_pool/{thread_pool_patterns}", Operation: "cat"},
		{Method: "POST", Path: "/{index}/_clone/{target}", Collection: "{index}", Operation: "clone"},
		{Method: "PUT", Path: "/{index}/_clone/{target}", Collection: "{index}", Operation: "clone"},
		{Method: "POST", Path: "/{index}/_close", Collection: "{index}", Operation: "close"},
		{Method: "GET", Path: "/_cluster/allocation/explain", Operation: "cluster"},
		{Method: "POST", Path: "/_cluster/allocation/explain", Operation: "cluster"},
		{Method: "GET", Path: "/_cluster/settings", Operation: "cluster"},
		{Method: "GET", Path: "/_cluster/health", Operation: "cluster"},
		{Method: "GET", Path: "/_cluster/health/{index}", Collection: "", Operation: "cluster"},
		{Method: "GET", Path: "/_cluster/pending_tasks", Operation: "cluster"},
		{Method: "PUT", Path: "/_cluster/settings", Operation: "cluster"},
		{Method: "POST", Path: "/_cluster/reroute", Operation: "cluster"},
		{Method: "GET", Path: "/_cluster/state", Operation: "cluster"},
		{Method: "GET", Path: "/_cluster/state/{metric}", Operation: "cluster"},
		{Method: "GET", Path: "/_cluster/state/{metric}/{index}", Collection: "", Operation: "cluster"},
		{Method: "GET", Path: "/_cluster/stats", Operation: "cluster"},
		{Method: "GET", Path: "/_cluster/stats/nodes/{node_id}", Operation: "cluster"},
		{Method: "GET", Path: "/_cluster/nodes/hot_threads", Operation: "cluster"},
		{Method: "GET", Path: "/_cluster/nodes/hotthreads", Operation: "cluster"},
		{Method: "GET", Path: "/_cluster/nodes/{node_id}/hot_threads", Operation: "cluster"},
		{Method: "GET", Path: "/_cluster/nodes/{node_id}/hotthreads", Operation: "cluster"},
		{Method: "GET", Path: "/_count", Operation: "count"},
		{Method: "GET", Path: "/{index}/_count", Collection: "{index}", Operation: "count"},
		{Method: "GET", Path: "/{index}/{type}/_count", Collection: "{index}", Operation: "count"},
		{Method: "POST", Path: "/_count", Operation: "count"},
		{Method: "POST", Path: "/{index}/_count", Collection: "{index}", Operation: "count"},
		{Method: "POST", Path: "/{index}/{type}/_count", Collection: "{index}", Operation: "count"},
		{Method: "POST", Path: "/{index}/_create/{id}", Collection: "{index}", Operation: "create"},
		{Method: "POST", Path: "/{index}/{type}/{id}/_create", Collection: "{index}", Operation: "create"},
		{Method: "PUT", Path: "/{index}/_create/{id}", Collection: "{index}", Operation: "create"},
		{Method: "PUT", Path: "/{index}/{type}/{id}/_create", Collection: "{index}", Operation: "create"},
		{Method: "POST", Path: "/{index}/_delete_by_query", Collection: "{index}", Operation: "delete_by_query"},
		{Method: "POST", Path: "/{index}/{type}/_delete_by_query", Collection: "{index}", Operation: "delete_by_query"},
		{Method: "POST", Path: "/_delete_by_query/{task_id}/_rethrottle", Operation: "delete_by_query"},

		{Method: "DELETE", Path: "/{index}/_doc/{id}", Collection: "{index}", Operation: "delete"},
		{Method: "HEAD", Path: "/{index}/_doc/{id}", Collection: "{index}", Operation: "exists"},
		{Method: "GET", Path: "/{index}/_doc/{id}", Collection: "{index}", Operation: "get"},
		{Method: "POST", Path: "/{index}/_doc", Collection: "{index}", Operation: "create"},
		{Method: "POST", Path: "/{index}/_doc/{id}", Collection: "{index}", Operation: "create"},
		{Method: "PUT", Path: "/{index}/_doc/{id}", Collection: "{index}", Operation: "update"},

		{Method: "GET", Path: "/{index}/_explain/{id}", Collection: "{index}", Operation: "explain"},
		{Method: "GET", Path: "/{index}/{type}/{id}/_explain", Collection: "{index}", Operation: "explain"},
		{Method: "POST", Path: "/{index}/_explain/{id}", Collection: "{index}", Operation: "explain"},
		{Method: "POST", Path: "/{index}/{type}/{id}/_explain", Collection: "{index}", Operation: "explain"},
		{Method: "GET", Path: "/_field_caps", Operation: "field_caps"},
		{Method: "GET", Path: "/{index}/_field_caps", Collection: "{index}", Operation: "field_caps"},
		{Method: "POST", Path: "/_field_caps", Operation: "field_caps"},
		{Method: "POST", Path: "/{index}/_field_caps", Collection: "{index}", Operation: "field_caps"},
		{Method: "GET", Path: "/_flush", Operation: "flush"},
		{Method: "GET", Path: "/{index}/_flush", Collection: "{index}", Operation: "flush"},
		{Method: "POST", Path: "/_flush", Operation: "flush"},
		{Method: "POST", Path: "/{index}/_flush", Collection: "{index}", Operation: "flush"},
		{Method: "GET", Path: "/_flush/synced", Operation: "flush"},
		{Method: "GET", Path: "/{index}/_flush/synced", Collection: "{index}", Operation: "flush"},
		{Method: "POST", Path: "/_flush/synced", Operation: "flush"},
		{Method: "POST", Path: "/{index}/_flush/synced", Collection: "{index}", Operation: "flush"},
		{Method: "POST", Path: "/_forcemerge", Operation: "forcemerge"},
		{Method: "POST", Path: "/{index}/_forcemerge", Collection: "{index}", Operation: "forcemerge"},
		{Method: "DELETE", Path: "/_ingest/pipeline/{id}", Operation: "ingest"},
		{Method: "GET", Path: "/_ingest/pipeline", Operation: "ingest"},
		{Method: "GET", Path: "/_ingest/pipeline/{id}", Operation: "ingest"},
		{Method: "GET", Path: "/_ingest/processor/grok", Operation: "ingest"},
		{Method: "PUT", Path: "/_ingest/pipeline/{id}", Operation: "ingest"},
		{Method: "GET", Path: "/_ingest/pipeline/_simulate", Operation: "ingest"},
		{Method: "GET", Path: "/_ingest/pipeline/{id}/_simulate", Operation: "ingest"},
		{Method: "POST", Path: "/_ingest/pipeline/_simulate", Operation: "ingest"},
		{Method: "POST", Path: "/_ingest/pipeline/{id}/_simulate", Operation: "ingest"},
		{Method: "HEAD", Path: "/{index}/_mapping/{type}", Collection: "{index}", Operation: "mapping"},
		{Method: "GET", Path: "/_mapping/field/{fields}", Operation: "mapping"},
		{Method: "GET", Path: "/_mapping/{type}/field/{fields}", Operation: "mapping"},
		{Method: "GET", Path: "/{index}/_mapping/field/{fields}", Collection: "{index}", Operation: "mapping"},
		{Method: "GET", Path: "/{index}/_mapping/{type}/field/{fields}", Collection: "{index}", Operation: "mapping"},
		{Method: "GET", Path: "/_mapping", Operation: "mapping"},
		{Method: "GET", Path: "/_mapping/{type}", Operation: "mapping"},
		{Method: "GET", Path: "/{index}/_mapping", Collection: "{index}", Operation: "mapping"},
		{Method: "GET", Path: "/{index}/_mapping/{type}", Collection: "{index}", Operation: "mapping"},
		{Method: "POST", Path: "/_mapping/{type}", Operation: "mapping"},
		{Method: "POST", Path: "/{index}/_mapping/{type}", Collection: "{index}", Operation: "mapping"},
		{Method: "POST", Path: "/{index}/{type}/_mapping", Collection: "{index}", Operation: "mapping"},
		{Method: "POST", Path: "{index}/_mapping", Collection: "{index}", Operation: "mapping"},
		{Method: "PUT", Path: "/_mapping/{type}", Operation: "mapping"},
		{Method: "PUT", Path: "/{index}/_mapping/{type}", Collection: "{index}", Operation: "mapping"},
		{Method: "PUT", Path: "/{index}/{type}/_mapping", Collection: "{index}", Operation: "mapping"},
		{Method: "PUT", Path: "{index}/_mapping", Collection: "{index}", Operation: "mapping"},
		{Method: "POST", Path: "/_mappings/{type}", Operation: "mappings"},
		{Method: "POST", Path: "/{index}/_mappings/{type}", Collection: "{index}", Operation: "mappings"},
		{Method: "POST", Path: "/{index}/{type}/_mappings", Collection: "{index}", Operation: "mappings"},
		{Method: "POST", Path: "{index}/_mappings", Collection: "{index}", Operation: "mappings"},
		{Method: "PUT", Path: "/_mappings/{type}", Operation: "mappings"},
		{Method: "PUT", Path: "/{index}/_mappings/{type}", Collection: "{index}", Operation: "mappings"},
		{Method: "PUT", Path: "/{index}/{type}/_mappings", Collection: "{index}", Operation: "mappings"},
		{Method: "PUT", Path: "{index}/_mappings", Collection: "{index}", Operation: "mappings"},
		{Method: "GET", Path: "/_mget", Operation: "mget"},
		{Method: "GET", Path: "/{index}/_mget", Collection: "{index}", Operation: "mget"},
		{Method: "GET", Path: "/{index}/{type}/_mget", Collection: "{index}", Operation: "mget"},
		{Method: "POST", Path: "/_mget", Operation: "mget"},
		{Method: "POST", Path: "/{index}/_mget", Collection: "{index}", Operation: "mget"},
		{Method: "POST", Path: "/{index}/{type}/_mget", Collection: "{index}", Operation: "mget"},
		{Method: "GET", Path: "/_msearch", Operation: "msearch"},
		{Method: "GET", Path: "/{index}/_msearch", Collection: "{index}", Operation: "msearch"},
		{Method: "GET", Path: "/{index}/{type}/_msearch", Collection: "{index}", Operation: "msearch"},
		{Method: "POST", Path: "/_msearch", Operation: "msearch"},
		{Method: "POST", Path: "/{index}/_msearch", Collection: "{index}", Operation: "msearch"},
		{Method: "POST", Path: "/{index}/{type}/_msearch", Collection: "{index}", Operation: "msearch"},
		{Method: "GET", Path: "/_msearch/template", Operation: "msearch"},
		{Method: "GET", Path: "/{index}/_msearch/template", Collection: "{index}", Operation: "msearch"},
		{Method: "GET", Path: "/{index}/{type}/_msearch/template", Collection: "{index}", Operation: "msearch"},
		{Method: "POST", Path: "/_msearch/template", Operation: "msearch"},
		{Method: "POST", Path: "/{index}/_msearch/template", Collection: "{index}", Operation: "msearch"},
		{Method: "POST", Path: "/{index}/{type}/_msearch/template", Collection: "{index}", Operation: "msearch"},
		{Method: "GET", Path: "/_mtermvectors", Operation: "mtermvectors"},
		{Method: "GET", Path: "/{index}/_mtermvectors", Collection: "{index}", Operation: "mtermvectors"},
		{Method: "GET", Path: "/{index}/{type}/_mtermvectors", Collection: "{index}", Operation: "mtermvectors"},
		{Method: "POST", Path: "/_mtermvectors", Operation: "mtermvectors"},
		{Method: "POST", Path: "/{index}/_mtermvectors", Collection: "{index}", Operation: "mtermvectors"},
		{Method: "POST", Path: "/{index}/{type}/_mtermvectors", Collection: "{index}", Operation: "mtermvectors"},
		{Method: "GET", Path: "/_nodes/hot_threads", Operation: "nodes"},
		{Method: "GET", Path: "/_nodes/hotthreads", Operation: "nodes"},
		{Method: "GET", Path: "/_nodes/{node_id}/hot_threads", Operation: "nodes"},
		{Method: "GET", Path: "/_nodes/{node_id}/hotthreads", Operation: "nodes"},
		{Method: "GET", Path: "/_nodes", Operation: "nodes"},
		{Method: "GET", Path: "/_nodes/{metric}", Operation: "nodes"},
		{Method: "GET", Path: "/_nodes/{node_id}", Operation: "nodes"},
		{Method: "GET", Path: "/_nodes/{node_id}/{metric}", Operation: "nodes"},
		{Method: "POST", Path: "/_nodes/reload_secure_settings", Operation: "nodes"},
		{Method: "POST", Path: "/_nodes/{node_id}/reload_secure_settings", Operation: "nodes"},
		{Method: "GET", Path: "/_nodes/stats", Operation: "nodes"},
		{Method: "GET", Path: "/_nodes/stats/{metric}", Operation: "nodes"},
		{Method: "GET", Path: "/_nodes/stats/{metric}/{index_metric}", Operation: "nodes"},
		{Method: "GET", Path: "/_nodes/{node_id}/stats", Operation: "nodes"},
		{Method: "GET", Path: "/_nodes/{node_id}/stats/{metric}", Operation: "nodes"},
		{Method: "GET", Path: "/_nodes/{node_id}/stats/{metric}/{index_metric}", Operation: "nodes"},
		{Method: "GET", Path: "/_nodes/usage", Operation: "nodes"},
		{Method: "GET", Path: "/_nodes/usage/{metric}", Operation: "nodes"},
		{Method: "GET", Path: "/_nodes/{node_id}/usage", Operation: "nodes"},
		{Method: "GET", Path: "/_nodes/{node_id}/usage/{metric}", Operation: "nodes"},
		{Method: "POST", Path: "/{index}/_open", Collection: "{index}", Operation: "open"},
		{Method: "GET", Path: "/_rank_eval", Operation: "rank_eval"},
		{Method: "GET", Path: "/{index}/_rank_eval", Collection: "{index}", Operation: "rank_eval"},
		{Method: "POST", Path: "/_rank_eval", Operation: "rank_eval"},
		{Method: "POST", Path: "/{index}/_rank_eval", Collection: "{index}", Operation: "rank_eval"},
		{Method: "GET", Path: "/_recovery", Operation: "recovery"},
		{Method: "GET", Path: "/{index}/_recovery", Collection: "{index}", Operation: "recovery"},
		{Method: "GET", Path: "/_refresh", Operation: "refresh"},
		{Method: "GET", Path: "/{index}/_refresh", Collection: "{index}", Operation: "refresh"},
		{Method: "POST", Path: "/_refresh", Operation: "refresh"},
		{Method: "POST", Path: "/{index}/_refresh", Collection: "{index}", Operation: "refresh"},
		{Method: "POST", Path: "/_reindex", Operation: "reindex"},
		{Method: "POST", Path: "/_reindex/{task_id}/_rethrottle", Operation: "reindex"},
		{Method: "GET", Path: "/_remote/info", Operation: "remote"},
		{Method: "GET", Path: "/_render/template", Operation: "render"},
		{Method: "GET", Path: "/_render/template/{id}", Operation: "render"},
		{Method: "POST", Path: "/_render/template", Operation: "render"},
		{Method: "POST", Path: "/_render/template/{id}", Operation: "render"},
		{Method: "POST", Path: "/{alias}/_rollover", Operation: "rollover", Collection: "{alias}"},
		{Method: "POST", Path: "/{alias}/_rollover/{new_index}", Operation: "rollover", Collection: "{alias}"},
		{Method: "DELETE", Path: "/_scripts/{id}", Operation: "scripts"},
		{Method: "GET", Path: "/_scripts/{id}", Operation: "scripts"},
		{Method: "POST", Path: "/_scripts/{id}", Operation: "scripts"},
		{Method: "POST", Path: "/_scripts/{id}/{context}", Operation: "scripts"},
		{Method: "PUT", Path: "/_scripts/{id}", Operation: "scripts"},
		{Method: "PUT", Path: "/_scripts/{id}/{context}", Operation: "scripts"},
		{Method: "GET", Path: "/_scripts/painless/_execute", Operation: "scripts"},
		{Method: "POST", Path: "/_scripts/painless/_execute", Operation: "scripts"},

		{Method: "DELETE", Path: "/_search/scroll", Operation: "clear_scroll"},
		{Method: "DELETE", Path: "/_search/scroll/{scroll_id}", Operation: "clear_scroll"},
		{Method: "GET", Path: "/_search/scroll", Operation: "scroll"},
		{Method: "GET", Path: "/_search/scroll/{scroll_id}", Operation: "scroll"},
		{Method: "POST", Path: "/_search/scroll", Operation: "scroll"},
		{Method: "POST", Path: "/_search/scroll/{scroll_id}", Operation: "scroll"},
		{Method: "GET", Path: "/_search", Operation: "search"},
		{Method: "GET", Path: "/{index}/_search", Collection: "{index}", Operation: "search"},
		{Method: "GET", Path: "/{index}/{type}/_search", Collection: "{index}", Operation: "search"},
		{Method: "POST", Path: "/_search", Operation: "search"},
		{Method: "POST", Path: "/{index}/_search", Collection: "{index}", Operation: "search"},
		{Method: "POST", Path: "/{index}/{type}/_search", Collection: "{index}", Operation: "search"},
		{Method: "GET", Path: "/_search/template", Operation: "search_template"},
		{Method: "GET", Path: "/{index}/_search/template", Collection: "{index}", Operation: "search_template"},
		{Method: "GET", Path: "/{index}/{type}/_search/template", Collection: "{index}", Operation: "search_template"},
		{Method: "POST", Path: "/_search/template", Operation: "search_template"},
		{Method: "POST", Path: "/{index}/_search/template", Collection: "{index}", Operation: "search_template"},
		{Method: "POST", Path: "/{index}/{type}/_search/template", Collection: "{index}", Operation: "search_template"},

		{Method: "GET", Path: "/_search_shards", Operation: "search_shards"},
		{Method: "GET", Path: "/{index}/_search_shards", Collection: "{index}", Operation: "search_shards"},
		{Method: "POST", Path: "/_search_shards", Operation: "search_shards"},
		{Method: "POST", Path: "/{index}/_search_shards", Collection: "{index}", Operation: "search_shards"},
		{Method: "GET", Path: "/_segments", Operation: "segments"},
		{Method: "GET", Path: "/{index}/_segments", Collection: "{index}", Operation: "segments"},
		{Method: "GET", Path: "/_settings", Operation: "settings"},
		{Method: "GET", Path: "/_settings/{name}", Operation: "settings"},
		{Method: "GET", Path: "/{index}/_settings", Collection: "{index}", Operation: "settings"},
		{Method: "GET", Path: "/{index}/_settings/{name}", Collection: "{index}", Operation: "settings"},
		{Method: "PUT", Path: "/_settings", Operation: "settings"},
		{Method: "PUT", Path: "/{index}/_settings", Collection: "{index}", Operation: "settings"},
		{Method: "GET", Path: "/_shard_stores", Operation: "shard_stores"},
		{Method: "GET", Path: "/{index}/_shard_stores", Collection: "{index}", Operation: "shard_stores"},
		{Method: "POST", Path: "/{index}/_shrink/{target}", Collection: "{index}", Operation: "shrink"},
		{Method: "PUT", Path: "/{index}/_shrink/{target}", Collection: "{index}", Operation: "shrink"},
		{Method: "POST", Path: "/_snapshot/{repository}/_cleanup", Operation: "snapshot"},
		{Method: "POST", Path: "/_snapshot/{repository}/{snapshot}", Operation: "snapshot"},
		{Method: "PUT", Path: "/_snapshot/{repository}/{snapshot}", Operation: "snapshot"},
		{Method: "POST", Path: "/_snapshot/{repository}", Operation: "snapshot"},
		{Method: "PUT", Path: "/_snapshot/{repository}", Operation: "snapshot"},
		{Method: "DELETE", Path: "/_snapshot/{repository}/{snapshot}", Operation: "snapshot"},
		{Method: "DELETE", Path: "/_snapshot/{repository}", Operation: "snapshot"},
		{Method: "GET", Path: "/_snapshot/{repository}/{snapshot}", Operation: "snapshot"},
		{Method: "GET", Path: "/_snapshot", Operation: "snapshot"},
		{Method: "GET", Path: "/_snapshot/{repository}", Operation: "snapshot"},
		{Method: "POST", Path: "/_snapshot/{repository}/{snapshot}/_restore", Operation: "snapshot"},
		{Method: "GET", Path: "/_snapshot/_status", Operation: "snapshot"},
		{Method: "GET", Path: "/_snapshot/{repository}/_status", Operation: "snapshot"},
		{Method: "GET", Path: "/_snapshot/{repository}/{snapshot}/_status", Operation: "snapshot"},
		{Method: "POST", Path: "/_snapshot/{repository}/_verify", Operation: "snapshot"},
		{Method: "HEAD", Path: "/{index}/_source/{id}", Collection: "{index}", Operation: "source"},
		{Method: "HEAD", Path: "/{index}/{type}/{id}/_source", Collection: "{index}", Operation: "source"},
		{Method: "GET", Path: "/{index}/_source/{id}", Collection: "{index}", Operation: "source"},
		{Method: "GET", Path: "/{index}/{type}/{id}/_source", Collection: "{index}", Operation: "source"},
		{Method: "POST", Path: "/{index}/_split/{target}", Collection: "{index}", Operation: "split"},
		{Method: "PUT", Path: "/{index}/_split/{target}", Collection: "{index}", Operation: "split"},
		{Method: "GET", Path: "/_stats", Operation: "stats"},
		{Method: "GET", Path: "/_stats/{metric}", Operation: "stats"},
		{Method: "GET", Path: "/{index}/_stats", Collection: "{index}", Operation: "stats"},
		{Method: "GET", Path: "/{index}/_stats/{metric}", Collection: "{index}", Operation: "stats"},
		{Method: "POST", Path: "/_tasks/_cancel", Operation: "tasks"},
		{Method: "POST", Path: "/_tasks/{task_id}/_cancel", Operation: "tasks"},
		{Method: "GET", Path: "/_tasks/{task_id}", Operation: "tasks"},
		{Method: "GET", Path: "/_tasks", Operation: "tasks"},
		{Method: "DELETE", Path: "/_template/{name}", Operation: "template"},
		{Method: "HEAD", Path: "/_template/{name}", Operation: "template"},
		{Method: "GET", Path: "/_template", Operation: "template"},
		{Method: "GET", Path: "/_template/{name}", Operation: "template"},
		{Method: "POST", Path: "/_template/{name}", Operation: "template"},
		{Method: "PUT", Path: "/_template/{name}", Operation: "template"},
		{Method: "GET", Path: "/{index}/_termvectors", Collection: "{index}", Operation: "termvectors"},
		{Method: "GET", Path: "/{index}/_termvectors/{id}", Collection: "{index}", Operation: "termvectors"},
		{Method: "GET", Path: "/{index}/{type}/_termvectors", Collection: "{index}", Operation: "termvectors"},
		{Method: "GET", Path: "/{index}/{type}/{id}/_termvectors", Collection: "{index}", Operation: "termvectors"},
		{Method: "POST", Path: "/{index}/_termvectors", Collection: "{index}", Operation: "termvectors"},
		{Method: "POST", Path: "/{index}/_termvectors/{id}", Collection: "{index}", Operation: "termvectors"},
		{Method: "POST", Path: "/{index}/{type}/_termvectors", Collection: "{index}", Operation: "termvectors"},
		{Method: "POST", Path: "/{index}/{type}/{id}/_termvectors", Collection: "{index}", Operation: "termvectors"},
		{Method: "POST", Path: "/{index}/_update/{id}", Collection: "{index}", Operation: "update"},
		{Method: "POST", Path: "/{index}/{type}/{id}/_update", Collection: "{index}", Operation: "update"},
		{Method: "POST", Path: "/{index}/_update_by_query", Collection: "{index}", Operation: "update_by_query"},
		{Method: "POST", Path: "/{index}/{type}/_update_by_query", Collection: "{index}", Operation: "update_by_query"},
		{Method: "POST", Path: "/_update_by_query/{task_id}/_rethrottle", Operation: "update_by_query"},
		{Method: "GET", Path: "/_upgrade", Operation: "upgrade"},
		{Method: "GET", Path: "/{index}/_upgrade", Collection: "{index}", Operation: "upgrade"},
		{Method: "POST", Path: "/_upgrade", Operation: "upgrade"},
		{Method: "POST", Path: "/{index}/_upgrade", Collection: "{index}", Operation: "upgrade"},
		{Method: "GET", Path: "/_validate/query", Operation: "validate"},
		{Method: "GET", Path: "/{index}/_validate/query", Collection: "{index}", Operation: "validate"},
		{Method: "GET", Path: "/{index}/{type}/_validate/query", Collection: "{index}", Operation: "validate"},
		{Method: "POST", Path: "/_validate/query", Operation: "validate"},
		{Method: "POST", Path: "/{index}/_validate/query", Collection: "{index}", Operation: "validate"},
		{Method: "POST", Path: "/{index}/{type}/_validate/query", Collection: "{index}", Operation: "validate"},
	}

	for _, tc := range testcases {
		r := &http.Request{
			URL: &url.URL{
				Path: tc.Path,
			},
			Method: tc.Method,
		}
		segment := parseRequest(r)
		if segment.Operation != tc.Operation {
			t.Error("wrong operation", tc.Method, tc.Path, segment.Operation, tc.Operation)
		}
		if segment.Collection != tc.Collection {
			t.Error("wrong operation", tc.Method, tc.Path, segment.Collection, tc.Collection)
		}
	}
}

var (
	errSomething = errors.New("something went wrong")
)

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (fn roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return fn(r)
}

func createTestApp() integrationsupport.ExpectApp {
	return integrationsupport.NewTestApp(nil, integrationsupport.ConfigFullTraces)
}

func TestInfo(t *testing.T) {
	app := createTestApp()
	txn := app.StartTransaction("txnName")
	ctx := newrelic.NewContext(context.Background(), txn)

	client, err := elasticsearch.NewClient(elasticsearch.Config{
		Transport: NewRoundTripper(roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			return nil, errSomething
		})),
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.Info(client.Info.WithContext(ctx))
	if err != errSomething {
		t.Fatal(err)
	}
	txn.End()
	app.ExpectMetrics(t, []internal.WantMetric{
		{Name: "OtherTransactionTotalTime/Go/txnName"},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/all"},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/allOther"},
		{Name: "OtherTransaction/Go/txnName"},
		{Name: "OtherTransaction/all"},
		{Name: "OtherTransactionTotalTime"},
		{Name: "Datastore/all", Scope: "", Forced: nil, Data: nil},
		{Name: "Datastore/allOther", Scope: "", Forced: nil, Data: nil},
		{Name: "Datastore/Elasticsearch/all", Scope: "", Forced: nil, Data: nil},
		{Name: "Datastore/Elasticsearch/allOther", Scope: "", Forced: nil, Data: nil},
		{Name: "Datastore/operation/Elasticsearch/info", Scope: "", Forced: nil, Data: nil},
		{Name: "Datastore/operation/Elasticsearch/info", Scope: "OtherTransaction/Go/txnName", Forced: nil, Data: nil},
	})
}

func TestSearch(t *testing.T) {
	app := createTestApp()
	txn := app.StartTransaction("txnName")
	ctx := newrelic.NewContext(context.Background(), txn)

	client, err := elasticsearch.NewClient(elasticsearch.Config{
		Transport: NewRoundTripper(roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			return nil, errSomething
		})),
	})
	if err != nil {
		t.Fatal(err)
	}
	body := `{"query":{"match":{"title":"test"}}}`
	_, err = client.Search(
		client.Search.WithContext(ctx),
		client.Search.WithIndex("myindex"),
		client.Search.WithBody(strings.NewReader(body)),
	)
	if err != errSomething {
		t.Fatal(err)
	}
	txn.End()
	app.ExpectMetrics(t, []internal.WantMetric{
		{Name: "OtherTransactionTotalTime/Go/txnName"},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/all"},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/allOther"},
		{Name: "OtherTransaction/Go/txnName"},
		{Name: "OtherTransaction/all"},
		{Name: "OtherTransactionTotalTime"},
		{Name: "Datastore/all"},
		{Name: "Datastore/allOther"},
		{Name: "Datastore/Elasticsearch/all"},
		{Name: "Datastore/Elasticsearch/allOther"},
		{Name: "Datastore/operation/Elasticsearch/info"},
		{Name: "Datastore/operation/Elasticsearch/info", Scope: "OtherTransaction/Go/txnName", Forced: nil, Data: nil},
	})
}

func TestInfoRequest(t *testing.T) {
	// Test that the instrumentation works as expected when the Do()
	// request pattern is used.
	app := createTestApp()
	txn := app.StartTransaction("txnName")
	ctx := newrelic.NewContext(context.Background(), txn)

	client, err := elasticsearch.NewClient(elasticsearch.Config{
		Transport: NewRoundTripper(roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			return nil, errSomething
		})),
	})
	if err != nil {
		t.Fatal(err)
	}
	req := esapi.InfoRequest{}
	_, err = req.Do(ctx, client)
	if err != errSomething {
		t.Fatal(err)
	}
	txn.End()
	app.ExpectMetrics(t, []internal.WantMetric{
		{Name: "OtherTransactionTotalTime/Go/txnName"},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/all"},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/allOther"},
		{Name: "OtherTransaction/Go/txnName"},
		{Name: "OtherTransaction/all"},
		{Name: "OtherTransactionTotalTime"},
		{Name: "Datastore/all", Scope: "", Forced: nil, Data: nil},
		{Name: "Datastore/allOther", Scope: "", Forced: nil, Data: nil},
		{Name: "Datastore/Elasticsearch/all", Scope: "", Forced: nil, Data: nil},
		{Name: "Datastore/Elasticsearch/allOther", Scope: "", Forced: nil, Data: nil},
		{Name: "Datastore/operation/Elasticsearch/info", Scope: "", Forced: nil, Data: nil},
		{Name: "Datastore/operation/Elasticsearch/info", Scope: "OtherTransaction/Go/txnName", Forced: nil, Data: nil},
	})
}
