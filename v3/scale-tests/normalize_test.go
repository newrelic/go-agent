// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0
//

package main

import (
	"testing"
)

func TestInterquartileRangesEven(t *testing.T) {
	vals := []int64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	q1, q3 := interquartileRanges(vals)
	if q1 != 3 {
		t.Errorf("Expected Q1 to equal 3, got %v", q1)
	}
	if q3 != 8 {
		t.Errorf("Expected Q3 to equal 8, got %v", q3)
	}
}

func TestInterquartileRangesOdd(t *testing.T) {
	vals := []int64{1, 2, 3, 4, 5, 6, 7, 8, 9}
	q1, q3 := interquartileRanges(vals)
	if q1 != 2.5 {
		t.Errorf("Expected Q1 to equal 2.5, got %v", q1)
	}
	if q3 != 7.5 {
		t.Errorf("Expected Q3 to equal 7.5, got %v", q3)
	}
}

func TestNormalizeEven(t *testing.T) {
	vals := []int64{1, 2, 3, 4, 5, 6, 7, 8, 9, 100}
	expect := []int64{1, 2, 3, 4, 5, 6, 7, 8, 9}
	validTimes, sum := normalize(vals)

	if !AssertInt64SliceEquals(validTimes, expect) {
		t.Errorf("Array was not normalized: %v should be %v", vals, expect)
	}

	if sum != 45 {
		t.Errorf("Sum should be 45, got %v", sum)
	}
}

func TestNormalizeOdd(t *testing.T) {
	vals := []int64{2, 3, 4, 5, 6, 7, 8, 9, 100}
	expect := []int64{2, 3, 4, 5, 6, 7, 8, 9}
	validTimes, sum := normalize(vals)

	if !AssertInt64SliceEquals(validTimes, expect) {
		t.Errorf("Array was not normalized: %v should be %v", vals, expect)
	}

	if sum != 44 {
		t.Errorf("Sum should be 44, got %v", sum)
	}
}

func TestNormalizeNoop(t *testing.T) {
	vals := []int64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	expect := []int64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	validTimes, sum := normalize(vals)

	if !AssertInt64SliceEquals(validTimes, expect) {
		t.Errorf("Array was not normalized: %v should be %v", vals, expect)
	}

	if sum != 55 {
		t.Errorf("Sum should be 55, got %v", sum)
	}
}

func AssertInt64SliceEquals(a, b []int64) bool {
	if len(a) != len(b) {
		return false
	}

	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}
