// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package nrpkgerrors_test

import (
	newrelic "github.com/newrelic/go-agent"
	"github.com/newrelic/go-agent/_integrations/nrpkgerrors"
	"github.com/pkg/errors"
)

type rootError string

func (e rootError) Error() string { return string(e) }

func makeRootError() error {
	return errors.WithStack(rootError("this is the original error"))
}

func Example() {
	var txn newrelic.Transaction
	e := errors.Wrap(makeRootError(), "extra information")
	// Wrap the error to record stack-trace and class type information from
	// the error's root cause.  Here, "rootError" will be recored as the
	// class and top stack-trace frame will be inside makeRootError().
	// Without nrpkgerrors.Wrap, "*errors.withStack" would be recorded as
	// the class and the top stack-trace frame would be site of the
	// NoticeError call.
	txn.NoticeError(nrpkgerrors.Wrap(e))
}
