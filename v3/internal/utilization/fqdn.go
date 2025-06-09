// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

//go:build go1.8
// +build go1.8

package utilization

import (
	"context"
	"net"
	"strings"
)

func lookupAddr(addr string) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), lookupAddrTimeout)
	defer cancel()

	r := &net.Resolver{}

	return r.LookupAddr(ctx, addr)
}

func getFQDN(candidateIPs []string) string {
	for _, ip := range candidateIPs {
		names, _ := lookupAddr(ip)
		if len(names) > 0 {
			return strings.TrimSuffix(names[0], ".")
		}
	}
	return ""
}
