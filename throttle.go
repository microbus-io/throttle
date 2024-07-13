/*
Copyright 2024 Microbus LLC and various contributors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package throttle

import (
	"sync"
	"time"
)

// Throttle implements the sliding window algorithm for rate limiting.
type Throttle struct {
	mux             sync.Mutex
	windowMillis    int
	limit           int
	counter         [2]int
	lastPeriodIndex int64
}

// New creates a throttle that allows no more than a limited number of operations to occur in a sliding time window.
// The smallest granularity for the time window is 1 millisecond.
func New(window time.Duration, limit int) *Throttle {
	return &Throttle{
		windowMillis: int(window.Milliseconds()),
		limit:        limit,
	}
}

// Allow returns whether or not the operation is allowed. If it is, the internal count of operations is incremented.
func (t *Throttle) Allow() bool {
	return t.AllowN(1)
}

// AllowN returns whether or not the operation is allowed, given a weight.
func (t *Throttle) AllowN(wt int) bool {
	// Divide the timeline into fixed windows. Identify where now falls
	// |---------|---------|---------|---------|
	//             ^
	//            now
	now := time.Now().UnixMilli()
	periodIndex := float64(now) / float64(t.windowMillis) // e.g. 12345.2
	periodIndexInt := int64(periodIndex)                  // e.g. 12345

	// counter[0] is for even periods, counter[1] is for odd periods
	currentCounter := 0
	if periodIndexInt%2 != 0 {
		currentCounter = 1
	}
	previousCounter := 1 - currentCounter

	// Prorate the counter of the previous period based on how much of the current period has elapsed
	// For example, if 20% of the current period elapsed, take 80% of the counter of the previous period
	// |---------|---------|---------|---------|
	//   >         <
	// start      now
	proration := 1.0 - (periodIndex - float64(periodIndexInt)) // e.g. 0.8

	t.mux.Lock()
	defer t.mux.Unlock()

	// Reset counter(s) if the last call happened in a previous period
	if periodIndexInt > t.lastPeriodIndex {
		t.counter[currentCounter] = 0
		if periodIndexInt > t.lastPeriodIndex+1 {
			t.counter[previousCounter] = 0
		}
		t.lastPeriodIndex = periodIndexInt
	}

	// The sliding window load is estimated to be the counter of the current period, plus the proration of the counter of the previous period
	estimatedLoad := t.counter[currentCounter] + int(float64(t.counter[previousCounter])*proration)

	// Check against limit
	if estimatedLoad > t.limit-wt {
		return false
	}
	// Increment current counter if op is allowed
	t.counter[currentCounter] += wt
	return true
}
