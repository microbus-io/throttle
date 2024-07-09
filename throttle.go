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
	"math"
	"sync/atomic"
	"time"
)

// Throttle implements the sliding window algorithm for rate limiting.
type Throttle struct {
	windowMillis int32
	limit        int32
	count        [2]atomic.Int32
	last         atomic.Int64
}

// NewThrottle creates a new sliding window throttle, allowing no more than a limited number of operations to happen in a given time window.
func NewThrottle(windowMillis int32, limit int32) *Throttle {
	return &Throttle{
		windowMillis: windowMillis,
		limit:        limit,
	}
}

// Allow returns whether or not the operation is allowed. If it is, the internal count of operations is incremented.
func (t *Throttle) Allow() bool {
	return t.AllowN(1)
}

// AllowN returns whether or not the operation is allowed, given a weight.
func (t *Throttle) AllowN(wt int32) bool {
	now := time.Now().UnixMilli()
	div := float64(now) / float64(2*t.windowMillis)
	divX2 := int64(div * 2)

	diff := int32(float64(now) - math.Floor(div)*float64(2*t.windowMillis))
	current := int8(diff / t.windowMillis)

	// Shift counters if necessary
	last := t.last.Load()
	if divX2 > last {
		t.count[current].Store(0)
		if divX2 > last+2 {
			t.count[1-current].Store(0)
		}
		t.last.Store(divX2)
	}

	mod := diff % t.windowMillis
	proratedPrev := float64(t.count[1-current].Load()) * float64(t.windowMillis-mod) / float64(t.windowMillis)
	sum := t.count[current].Load() + int32(proratedPrev)
	if sum+wt > t.limit {
		return false
	}
	t.count[current].Add(wt)
	return true
}
