// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

// Package nrelasticsearch instruments https://github.com/elastic/go-elasticsearch.
//
// Use this package to instrument your elasticsearch v7 calls without having to
// manually create DatastoreSegments.
package nrelasticsearch

import (
	"net/http"
	"strings"

	"github.com/newrelic/go-agent/v3/internal"
	newrelic "github.com/newrelic/go-agent/v3/newrelic"
)

func init() { internal.TrackUsage("integration", "datastore", "elasticsearch") }

func parseRequest(r *http.Request) (segment newrelic.DatastoreSegment) {

	segment.StartTime = newrelic.FromContext(r.Context()).StartSegmentNow()
	segment.Product = newrelic.DatastoreElasticsearch

	path := strings.TrimPrefix(r.URL.Path, "/")
	method := r.Method

	if "" == path {
		switch method {
		case "GET":
			segment.Operation = "info"
		case "HEAD":
			segment.Operation = "ping"
		}
		return
	}

	segments := strings.Split(path, "/")
	for idx, s := range segments {
		switch s {
		case "_alias",
			"_aliases",
			"_analyze",
			"_bulk",
			"_cache",
			"_cat",
			"_clone",
			"_close",
			"_cluster",
			"_count",
			"_create",
			"_delete_by_query",
			"_explain",
			"_field_caps",
			"_flush",
			"_forcemerge",
			"_ingest",
			"_mapping",
			"_mappings",
			"_mget",
			"_msearch",
			"_mtermvectors",
			"_nodes",
			"_open",
			"_rank_eval",
			"_recovery",
			"_refresh",
			"_reindex",
			"_remote",
			"_render",
			"_rollover",
			"_scripts",
			"_search_shards",
			"_segments",
			"_settings",
			"_shard_stores",
			"_shrink",
			"_snapshot",
			"_source",
			"_split",
			"_stats",
			"_tasks",
			"_template",
			"_termvectors",
			"_update",
			"_update_by_query",
			"_upgrade",
			"_validate":
			segment.Operation = strings.TrimPrefix(s, "_")
			if idx > 0 {
				segment.Collection = segments[0]
			}
			return
		case "_doc":
			switch method {
			case "DELETE":
				segment.Operation = "delete"
			case "HEAD":
				segment.Operation = "exists"
			case "GET":
				segment.Operation = "get"
			case "PUT":
				segment.Operation = "update"
			case "POST":
				segment.Operation = "create"
			}
			if idx > 0 {
				segment.Collection = segments[0]
			}
			return
		case "_search":
			// clear_scroll.json      DELETE   /_search/scroll
			// clear_scroll.json      DELETE   /_search/scroll/{scroll_id}
			// scroll.json            GET      /_search/scroll
			// scroll.json            GET      /_search/scroll/{scroll_id}
			// scroll.json            POST     /_search/scroll
			// scroll.json            POST     /_search/scroll/{scroll_id}
			// search.json            GET      /_search
			// search.json            GET      /{index}/_search
			// search.json            GET      /{index}/{type}/_search
			// search.json            POST     /_search
			// search.json            POST     /{index}/_search
			// search.json            POST     /{index}/{type}/_search
			// search_template.json   GET      /_search/template
			// search_template.json   GET      /{index}/_search/template
			// search_template.json   GET      /{index}/{type}/_search/template
			// search_template.json   POST     /_search/template
			// search_template.json   POST     /{index}/_search/template
			// search_template.json   POST     /{index}/{type}/_search/template
			if method == "DELETE" {
				segment.Operation = "clear_scroll"
				return
			}
			if idx == len(segments)-1 {
				segment.Operation = "search"
				if idx > 0 {
					segment.Collection = segments[0]
				}
				return
			}
			next := segments[idx+1]
			if next == "scroll" {
				segment.Operation = "scroll"
				return
			}
			if next == "template" {
				segment.Operation = "search_template"
				if idx > 0 {
					segment.Collection = segments[0]
				}
				return
			}
			return
		}
	}
	return
}

type roundtripper struct{ original http.RoundTripper }

func (t roundtripper) RoundTrip(r *http.Request) (*http.Response, error) {
	segment := parseRequest(r)
	defer segment.End()

	return t.original.RoundTrip(r)
}

// NewRoundTripper creates a new http.RoundTripper to instrument elasticsearch
// calls.  If an http.RoundTripper parameter is not provided, then the returned
// http.RoundTripper will delegate to http.DefaultTransport.
func NewRoundTripper(original http.RoundTripper) http.RoundTripper {
	if nil == original {
		original = http.DefaultTransport
	}
	return roundtripper{original: original}
}
