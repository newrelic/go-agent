// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package newrelic

// BrowserTimingHeader encapsulates the JavaScript required to enable New
// Relic's Browser product.
type BrowserTimingHeader struct{}

// WithTags returns the browser timing JavaScript which includes the enclosing
// <script> and </script> tags.  This method returns nil if the receiver is
// nil, the feature is disabled, the application is not yet connected, or an
// error occurs.  The byte slice returned is in UTF-8 format.
func (h *BrowserTimingHeader) WithTags() []byte {
	return nil
}

// WithoutTags returns the browser timing JavaScript without any enclosing tags,
// which may then be embedded within any JavaScript code.  This method returns
// nil if the receiver is nil, the feature is disabled, the application is not
// yet connected, or an error occurs.  The byte slice returned is in UTF-8
// format.
func (h *BrowserTimingHeader) WithoutTags() []byte {
	return nil
}
