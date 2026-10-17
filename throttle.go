/*
Copyright 2024-2026 Microbus LLC and various contributors

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
	window          time.Duration
	limit           int
	counter         [2]int
	lastPeriodIndex int64
	start           time.Time
}

// New creates a throttle that allows no more than a limited number of operations to occur in a sliding time window.
// A window of zero or less is taken as 1 nanosecond.
func New(window time.Duration, limit int) *Throttle {
	return &Throttle{
		window: max(window, 1),
		limit:  limit,
		start:  time.Now(),
	}
}

// SetLimit updates the throttle's limit. Counters are preserved, so the new
// limit takes effect against the existing sliding-window history rather than
// resetting it.
func (t *Throttle) SetLimit(limit int) {
	t.mux.Lock()
	t.limit = limit
	t.mux.Unlock()
}

// Limit returns the throttle's current limit.
func (t *Throttle) Limit() int {
	t.mux.Lock()
	defer t.mux.Unlock()
	return t.limit
}

// Allow returns whether the operation is allowed and the sliding window's
// observed load measured before the call. On admit, the count is then
// incremented by 1; on refusal, the count is unchanged. The returned observed
// never includes the current op - add the weight if you want the post-call
// load.
func (t *Throttle) Allow() (admit bool, observed int) {
	return t.AllowN(1)
}

// AllowN is Allow with an explicit weight. A negative weight is refused.
func (t *Throttle) AllowN(wt int) (admit bool, observed int) {
	return t.check(wt, true)
}

// Peek reports whether Allow would currently admit, along with the sliding
// window's observed load, without modifying state. The result is advisory:
// a concurrent caller may change the outcome of a follow-up Allow.
func (t *Throttle) Peek() (admit bool, observed int) {
	return t.PeekN(1)
}

// PeekN is Peek with an explicit weight. A negative weight is refused.
func (t *Throttle) PeekN(wt int) (admit bool, observed int) {
	return t.check(wt, false)
}

func (t *Throttle) check(wt int, commit bool) (admit bool, observed int) {
	t.mux.Lock()
	defer t.mux.Unlock()
	// Read the clock under the lock, so calls see time in the order they update the counters.
	return t.checkAt(time.Since(t.start), wt, commit)
}

// checkAt is check at elapsed time since start. The caller holds the lock.
func (t *Throttle) checkAt(elapsed time.Duration, wt int, commit bool) (admit bool, observed int) {
	// Divide the timeline into fixed windows. Identify where now falls
	// |---------|---------|---------|---------|
	//             ^
	//            now
	periodIndexInt := int64(elapsed / t.window) // e.g. 12345

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
	proration := 1.0 - float64(elapsed%t.window)/float64(t.window) // e.g. 0.8

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

	// Check against limit. A negative weight would lower the counter and admit ops past the limit.
	if wt < 0 || estimatedLoad > t.limit-wt {
		return false, estimatedLoad
	}
	if commit {
		t.counter[currentCounter] += wt
	}
	return true, estimatedLoad
}
