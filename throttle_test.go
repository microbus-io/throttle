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
	"fmt"
	"math/rand"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// drive offers the throttle one call every step for d of simulated time, weighing each call with wt, and returns
// the weight admitted and refused. Simulated time makes the accuracy tests independent of how much CPU the test
// process gets: a busy loop on a loaded machine is starved for whole windows and under-admits.
func drive(th *Throttle, d, step time.Duration, wt func(elapsed time.Duration) int) (allowed, disallowed int) {
	for elapsed := time.Duration(0); elapsed < d; elapsed += step {
		w := wt(elapsed)
		if w == 0 {
			continue
		}
		th.mux.Lock()
		admit, _ := th.checkAt(elapsed, w, true)
		th.mux.Unlock()
		if admit {
			allowed += w
		} else {
			disallowed += w
		}
	}
	return allowed, disallowed
}

func Test_Constant(t *testing.T) {
	t.Parallel()

	throttle := New(10*time.Millisecond, 5) // 5 ops per 10 ms, or 500 per second
	allowed, disallowed := drive(throttle, time.Second, 10*time.Microsecond, func(time.Duration) int { return 1 })
	if allowed < 490 || allowed > 510 {
		t.Error(allowed)
	}
	fmt.Printf("Allowed %d\nDisallowed %d\n", allowed, disallowed)
}

// Test_Concurrent runs on the real clock, so a loaded machine can starve its goroutines and under-admit. Only
// over-admission is a throttle bug, so the upper bound is the tight one.
func Test_Concurrent(t *testing.T) {
	t.Parallel()

	throttle := New(10*time.Millisecond, 5) // 5 ops per 10 ms, or 500 per second
	var allowed atomic.Int32
	var disallowed atomic.Int32
	var wg sync.WaitGroup
	t0 := time.Now()
	for i := 0; i < runtime.NumCPU(); i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for time.Since(t0) < time.Second {
				admit, _ := throttle.Allow()
				if admit {
					allowed.Add(1)
				} else {
					disallowed.Add(1)
				}
			}
		}()
	}
	wg.Wait()
	// The goroutines can run a little past the second, so allow one window's worth over.
	if allowed.Load() < 100 || allowed.Load() > 515 {
		t.Error(allowed.Load())
	}
	fmt.Printf("Allowed %d\nDisallowed %d\n", allowed.Load(), disallowed.Load())
}

// Test_Overflow runs on the real clock; see Test_Concurrent for its bounds.
func Test_Overflow(t *testing.T) {
	t.Parallel()

	throttle := New(10*time.Millisecond, 500) // 500 op weight per 10 ms
	var allowed atomic.Int32
	var disallowed atomic.Int32
	var wg sync.WaitGroup
	t0 := time.Now()
	for i := 0; i < runtime.NumCPU(); i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for time.Since(t0) < time.Second {
				admit, _ := throttle.AllowN(300)
				if admit {
					allowed.Add(1)
				} else {
					disallowed.Add(1)
				}
			}
		}()
	}
	wg.Wait()
	if allowed.Load() < 20 || allowed.Load() > 111 {
		t.Error(allowed.Load())
	}
	fmt.Printf("Allowed %d\nDisallowed %d\n", allowed.Load(), disallowed.Load())
}

func Test_Weight(t *testing.T) {
	t.Parallel()

	throttle := New(10*time.Millisecond, 5) // 5 ops per 10 ms, or 500 per second
	allowed, disallowed := drive(throttle, time.Second, 10*time.Microsecond, func(time.Duration) int { return 5 })
	if allowed < 490 || allowed > 510 {
		t.Error(allowed)
	}
	fmt.Printf("Allowed %d\nDisallowed %d\n", allowed, disallowed)
}

func Test_RandomWeight(t *testing.T) {
	t.Parallel()

	r := rand.New(rand.NewSource(time.Now().UnixMilli()))

	maxWeight := 20
	throttle := New(10*time.Millisecond, maxWeight) // 20 ops per 10 ms, or 2000 per second
	allowed, disallowed := drive(throttle, time.Second, 10*time.Microsecond, func(time.Duration) int {
		return r.Intn(maxWeight) + 1
	})
	if allowed < maxWeight*90 || allowed > maxWeight*100 {
		t.Error(allowed)
	}
	fmt.Printf("Allowed %d\nDisallowed %d\n", allowed, disallowed)
}

func Test_Intermittent(t *testing.T) {
	t.Parallel()

	throttle := New(10*time.Millisecond, 5) // 5 ops per 10 ms, or 500 per second
	// Calls only in alternate 100ms stretches.
	allowed, disallowed := drive(throttle, time.Second, 10*time.Microsecond, func(elapsed time.Duration) int {
		if elapsed.Milliseconds()/100%2 == 0 {
			return 1
		}
		return 0
	})
	if allowed < 250 || allowed > 500 {
		t.Error(allowed)
	}
	fmt.Printf("Allowed %d\nDisallowed %d\n", allowed, disallowed)
}

func Test_Random(t *testing.T) {
	t.Parallel()

	r := rand.New(rand.NewSource(time.Now().UnixMilli()))

	throttle := New(10*time.Millisecond, 5) // 5 ops per 10 ms, or 500 per second
	allowed, disallowed := drive(throttle, time.Second, 10*time.Microsecond, func(time.Duration) int {
		if r.Intn(5) == 0 {
			return 1
		}
		return 0
	})
	if allowed < 490 || allowed > 510 {
		t.Error(allowed)
	}
	fmt.Printf("Allowed %d\nDisallowed %d\n", allowed, disallowed)
}

func Test_Rotation(t *testing.T) {
	t.Parallel()

	window := 100 * time.Millisecond
	throttle := New(window, 100)
	sum := func() int { return throttle.counter[0] + throttle.counter[1] }
	allowAt := func(elapsed time.Duration) {
		throttle.mux.Lock()
		throttle.checkAt(elapsed, 1, true)
		throttle.mux.Unlock()
	}
	if sum() != 0 {
		t.Error(sum())
	}
	allowAt(10 * time.Millisecond)
	allowAt(10 * time.Millisecond)
	allowAt(10 * time.Millisecond)
	if sum() != 3 {
		t.Error(sum())
	}
	allowAt(110 * time.Millisecond) // the next period keeps the previous one's counter
	if sum() != 4 {
		t.Error(sum())
	}
	allowAt(210 * time.Millisecond) // one period later, the first period's counter is reset
	if sum() != 2 {
		t.Error(sum())
	}
	allowAt(410 * time.Millisecond) // two periods later, both are
	if sum() != 1 {
		t.Error(sum())
	}
}

// Test_SubMillisecondWindow pins that a window under 1ms works. Truncated to whole milliseconds it became 0, and
// dividing by it either refused everything after the first fill (arm64) or admitted everything (amd64).
func Test_SubMillisecondWindow(t *testing.T) {
	t.Parallel()

	throttle := New(500*time.Microsecond, 3) // 3 ops per 0.5 ms
	allowed, _ := drive(throttle, 100*time.Millisecond, time.Microsecond, func(time.Duration) int { return 1 })
	if allowed < 590 || allowed > 610 {
		t.Error(allowed)
	}

	// The same on the real clock: the broken arithmetic gave 3, or hundreds of thousands.
	throttle = New(500*time.Microsecond, 3)
	n := 0
	t0 := time.Now()
	for time.Since(t0) < 50*time.Millisecond {
		if admit, _ := throttle.Allow(); admit {
			n++
		}
	}
	if n < 10 || n > 320 {
		t.Error(n)
	}
}

// Test_FractionalWindow pins that a window is not rounded to whole milliseconds: at 1.5ms, rounding down to 1ms
// would admit 50% more.
func Test_FractionalWindow(t *testing.T) {
	t.Parallel()

	throttle := New(1500*time.Microsecond, 3) // 3 ops per 1.5 ms, or 2000 per second
	allowed, _ := drive(throttle, time.Second, time.Microsecond, func(time.Duration) int { return 1 })
	if allowed < 1990 || allowed > 2010 {
		t.Error(allowed)
	}
}

// Test_NegativeWeight pins that a negative weight is refused and leaves the counters alone: admitted, it lowered
// the count and let later ops past the limit.
func Test_NegativeWeight(t *testing.T) {
	t.Parallel()

	throttle := New(time.Second, 2)
	throttle.Allow()
	throttle.Allow()
	if admit, _ := throttle.AllowN(-100); admit {
		t.Error("a negative weight was admitted")
	}
	if admit, _ := throttle.PeekN(-1); admit {
		t.Error("a negative weight peeked as admitted")
	}
	if admit, observed := throttle.Allow(); admit || observed != 2 {
		t.Errorf("after a negative weight: admit=%v observed=%d", admit, observed)
	}
}

func Test_Peek(t *testing.T) {
	t.Parallel()

	throttle := New(time.Second, 3)
	// Empty throttle: peek admits, observed is 0.
	admit, observed := throttle.Peek()
	if !admit || observed != 0 {
		t.Errorf("empty peek: admit=%v observed=%d", admit, observed)
	}
	// Allow on an empty throttle: admits with pre-commit observed of 0.
	admit, observed = throttle.Allow()
	if !admit || observed != 0 {
		t.Errorf("first Allow: admit=%v observed=%d", admit, observed)
	}
	// Peek after 1 commit: observed reflects the prior op.
	admit, observed = throttle.Peek()
	if !admit || observed != 1 {
		t.Errorf("peek after 1: admit=%v observed=%d", admit, observed)
	}
	// Peek does not consume - repeated peeks give the same answer.
	admit, observed = throttle.Peek()
	if !admit || observed != 1 {
		t.Errorf("repeat peek: admit=%v observed=%d", admit, observed)
	}
	// Commit 2 more ops, filling to the limit. Pre-commit observed grows 1,2.
	for i, want := range []int{1, 2} {
		admit, observed = throttle.Allow()
		if !admit || observed != want {
			t.Errorf("Allow %d: admit=%v observed=%d want=%d", i, admit, observed, want)
		}
	}
	// At the limit: peek refuses, observed equals the limit.
	admit, observed = throttle.Peek()
	if admit || observed != 3 {
		t.Errorf("full peek: admit=%v observed=%d", admit, observed)
	}
	// Allow refuses too, observed unchanged (no increment on rejection).
	admit, observed = throttle.Allow()
	if admit || observed != 3 {
		t.Errorf("full Allow: admit=%v observed=%d", admit, observed)
	}
}

func Test_SetLimit(t *testing.T) {
	t.Parallel()

	throttle := New(time.Second, 5)
	if throttle.Limit() != 5 {
		t.Errorf("initial limit: got %d", throttle.Limit())
	}
	// Fill to the limit.
	for i := 0; i < 5; i++ {
		throttle.Allow()
	}
	admit, observed := throttle.Peek()
	if admit || observed != 5 {
		t.Errorf("at limit: admit=%v observed=%d", admit, observed)
	}
	// Raise the limit; counters preserved, room reopens.
	throttle.SetLimit(10)
	if throttle.Limit() != 10 {
		t.Errorf("after SetLimit: got %d", throttle.Limit())
	}
	admit, observed = throttle.Peek()
	if !admit || observed != 5 {
		t.Errorf("after raise: admit=%v observed=%d", admit, observed)
	}
	// Lower the limit below the current observed load; further admissions refused.
	throttle.SetLimit(3)
	admit, observed = throttle.Peek()
	if admit || observed != 5 {
		t.Errorf("after lower: admit=%v observed=%d", admit, observed)
	}
}

func Benchmark_Allow(b *testing.B) {
	throttle := New(time.Minute, b.N/2)
	for i := 0; i < b.N; i++ {
		throttle.Allow()
	}
}

func Benchmark_AllowN(b *testing.B) {
	throttle := New(time.Minute, b.N*5)
	for i := 0; i < b.N; i++ {
		throttle.AllowN(5)
	}
}

func Benchmark_Peek(b *testing.B) {
	throttle := New(time.Minute, b.N)
	for i := 0; i < b.N; i++ {
		throttle.Peek()
	}
}

func Benchmark_AllowConcurrent(b *testing.B) {
	throttle := New(time.Minute, b.N/2)
	var wg sync.WaitGroup
	for i := 0; i < runtime.NumCPU(); i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < b.N/runtime.NumCPU(); j++ {
				throttle.Allow()
			}
		}()
	}
	wg.Wait()
	// Benchmark suggests a 3X negative performance impact due to mutex contention
}
