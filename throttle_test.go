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
	"fmt"
	"math/rand"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func Test_Constant(t *testing.T) {
	t.Parallel()

	throttle := New(10*time.Millisecond, 5) // 5 ops per 10 ms, or 500 per second
	allowed := 0
	disallowed := 0
	t0 := time.Now()
	for time.Since(t0) < time.Second {
		admit, _ := throttle.Allow()
		if admit {
			allowed++
		} else {
			disallowed++
		}
	}
	if allowed < 490 || allowed > 510 {
		t.Error(allowed)
	}
	fmt.Printf("Allowed %d\nDisallowed %d\n", allowed, disallowed)
}

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
	if allowed.Load() < 490 || allowed.Load() > 510 {
		t.Error(allowed.Load())
	}
	fmt.Printf("Allowed %d\nDisallowed %d\n", allowed.Load(), disallowed.Load())
}

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
	if allowed.Load() < 90 || allowed.Load() > 110 {
		t.Error(allowed.Load())
	}
	fmt.Printf("Allowed %d\nDisallowed %d\n", allowed.Load(), disallowed.Load())
}

func Test_Weight(t *testing.T) {
	t.Parallel()

	throttle := New(10*time.Millisecond, 5) // 5 ops per 10 ms, or 500 per second
	allowed := 0
	disallowed := 0
	t0 := time.Now()
	for time.Since(t0) < time.Second {
		admit, _ := throttle.AllowN(5) // Weight=5
		if admit {
			allowed += 5
		} else {
			disallowed += 5
		}
	}
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
	allowed := 0
	disallowed := 0
	t0 := time.Now()
	for time.Since(t0) < time.Second {
		wt := r.Intn(maxWeight) + 1
		admit, _ := throttle.AllowN(wt)
		if admit {
			allowed += wt
		} else {
			disallowed += wt
		}
	}
	if allowed < maxWeight*90 || allowed > maxWeight*100 {
		t.Error(allowed)
	}
	fmt.Printf("Allowed %d\nDisallowed %d\n", allowed, disallowed)
}

func Test_Intermittent(t *testing.T) {
	t.Parallel()

	throttle := New(10*time.Millisecond, 5) // 5 ops per 10 ms, or 500 per second
	allowed := 0
	disallowed := 0
	t0 := time.Now()
	for time.Since(t0) < time.Second {
		if time.Since(t0).Milliseconds()/100%2 == 0 {
			admit, _ := throttle.AllowN(1)
			if admit {
				allowed++
			} else {
				disallowed++
			}
		}
	}
	if allowed < 250 || allowed > 500 {
		t.Error(allowed)
	}
	fmt.Printf("Allowed %d\nDisallowed %d\n", allowed, disallowed)
}

func Test_Random(t *testing.T) {
	t.Parallel()

	r := rand.New(rand.NewSource(time.Now().UnixMilli()))

	throttle := New(10*time.Millisecond, 5) // 5 ops per 10 ms, or 500 per second
	allowed := 0
	disallowed := 0
	t0 := time.Now()
	for time.Since(t0) < time.Second {
		if r.Intn(5) == 0 {
			admit, _ := throttle.AllowN(1)
			if admit {
				allowed++
			} else {
				disallowed++
			}
		}
	}
	if allowed < 490 || allowed > 510 {
		t.Error(allowed)
	}
	fmt.Printf("Allowed %d\nDisallowed %d\n", allowed, disallowed)
}

func Test_Rotation(t *testing.T) {
	t.Parallel()

	window := 100 * time.Millisecond
	time.Sleep(time.Duration(time.Now().UnixMilli()) % 100)
	throttle := New(window, 100)
	if throttle.counter[0]+throttle.counter[1] != 0 {
		t.Error(throttle.counter[0] + throttle.counter[1])
	}
	throttle.Allow()
	throttle.Allow()
	throttle.Allow()
	if throttle.counter[0]+throttle.counter[1] != 3 {
		t.Error(throttle.counter[0] + throttle.counter[1])
	}
	time.Sleep(window)
	if throttle.counter[0]+throttle.counter[1] != 3 {
		t.Error(throttle.counter[0] + throttle.counter[1])
	}
	throttle.Allow()
	if throttle.counter[0]+throttle.counter[1] != 4 {
		t.Error(throttle.counter[0] + throttle.counter[1])
	}
	time.Sleep(window)
	throttle.Allow()
	if throttle.counter[0]+throttle.counter[1] != 2 {
		t.Error(throttle.counter[0] + throttle.counter[1])
	}
	time.Sleep(2 * window)
	throttle.Allow()
	if throttle.counter[0]+throttle.counter[1] != 1 {
		t.Error(throttle.counter[0] + throttle.counter[1])
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
