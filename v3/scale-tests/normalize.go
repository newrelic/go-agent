// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0
//

package main

import (
	"sort"
)

// Normalizes the data by removing outliers based on z score
func normalize(times []int64) ([]int64, int64) {
	// not enough entries to do this calculation correctly
	if len(times) < 3 {
		var sum int64
		for _, time := range times {
			sum += time
		}
		return times, sum
	}

	var sum int64
	validTimes := make([]int64, 0, len(times))

	q1, q3 := interquartileRanges(times)
	iqr := q3 - q1
	upperFence := int64(q3 + (1.5 * iqr))
	lowerFence := int64(q1 - (1.5 * iqr))

	for _, time := range times {
		if time >= lowerFence && time <= upperFence {
			validTimes = append(validTimes, time)
			sum += time
		}
	}

	return validTimes, sum
}

func interquartileRanges(times []int64) (float64, float64) {
	sorted := make([]int, len(times))
	for i, val := range times {
		sorted[i] = int(val)
	}

	sort.Ints(sorted)

	var r1, r2 []int

	if len(sorted)%2 == 1 {
		r1 = sorted[:(len(sorted) / 2)]
		r2 = sorted[(len(sorted)/2)+1:]
	} else {
		r1 = sorted[:(len(sorted))/2]
		r2 = sorted[(len(sorted) / 2):]
	}

	q1 := median(r1)
	q3 := median(r2)

	return float64(q1), float64(q3)
}

func median(n []int) float64 {
	if len(n) == 0 {
		return 0
	}
	if len(n) == 1 {
		return float64(n[0])
	}
	if len(n)%2 == 1 {
		return float64(n[len(n)/2])
	} else {
		return float64((n[len(n)/2-1] + n[len(n)/2])) / 2
	}
}
