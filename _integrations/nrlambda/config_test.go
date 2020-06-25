// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package nrlambda

import (
	"testing"
	"time"
)

func TestNewConfig(t *testing.T) {
	cfg := newConfigInternal(func(key string) string {
		switch key {
		case "NEW_RELIC_ACCOUNT_ID":
			return "the-account-id"
		case "NEW_RELIC_TRUSTED_ACCOUNT_KEY":
			return "the-trust-key"
		case "NEW_RELIC_PRIMARY_APPLICATION_ID":
			return "the-app-id"
		case "NEW_RELIC_APDEX_T":
			return "2"
		default:
			return ""
		}
	})
	if !cfg.ServerlessMode.Enabled {
		t.Error(cfg.ServerlessMode.Enabled)
	}
	if cfg.ServerlessMode.AccountID != "the-account-id" {
		t.Error(cfg.ServerlessMode.AccountID)
	}
	if cfg.ServerlessMode.TrustedAccountKey != "the-trust-key" {
		t.Error(cfg.ServerlessMode.TrustedAccountKey)
	}
	if cfg.ServerlessMode.PrimaryAppID != "the-app-id" {
		t.Error(cfg.ServerlessMode.PrimaryAppID)
	}
	if cfg.ServerlessMode.ApdexThreshold != 2*time.Second {
		t.Error(cfg.ServerlessMode.ApdexThreshold)
	}
}
