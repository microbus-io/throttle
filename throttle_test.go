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
	"testing"
	"time"
)

func Test_Constant(t *testing.T) {
	t.Parallel()

	throttle := NewThrottle(10, 5) // 5 ops per 10 ms, or 500 per second
	allowed := 0
	disallowed := 0
	t0 := time.Now()
	for time.Since(t0) < time.Second {
		if throttle.Allow() {
			allowed++
		} else {
			disallowed++
		}
	}
	if allowed < 500 || allowed > 510 {
		t.Fail()
	}
	fmt.Printf("Allowed %d\nDisallowed %d\n", allowed, disallowed)
}

func Test_Weight(t *testing.T) {
	t.Parallel()

	throttle := NewThrottle(10, 5) // 5 ops per 10 ms, or 500 per second
	allowed := 0
	disallowed := 0
	t0 := time.Now()
	for time.Since(t0) < time.Second {
		if throttle.AllowN(5) { // Weight=5
			allowed += 5
		} else {
			disallowed += 5
		}
	}
	if allowed < 500 || allowed > 510 {
		t.Fail()
	}
	fmt.Printf("Allowed %d\nDisallowed %d\n", allowed, disallowed)
}

func Test_RandomWeight(t *testing.T) {
	t.Parallel()

	r := rand.New(rand.NewSource(time.Now().UnixMilli()))

	maxWeight := 20
	throttle := NewThrottle(10, int32(maxWeight)) // 20 ops per 10 ms, or 2000 per second
	allowed := 0
	disallowed := 0
	t0 := time.Now()
	for time.Since(t0) < time.Second {
		wt := r.Intn(maxWeight) + 1
		if throttle.AllowN(int32(wt)) {
			allowed += wt
		} else {
			disallowed += wt
		}
	}
	if allowed < maxWeight*90 || allowed > maxWeight*100 {
		t.Fail()
	}
	fmt.Printf("Allowed %d\nDisallowed %d\n", allowed, disallowed)
}

func Test_Intermittent(t *testing.T) {
	t.Parallel()

	throttle := NewThrottle(10, 5) // 5 ops per 10 ms, or 500 per second
	allowed := 0
	disallowed := 0
	t0 := time.Now()
	for time.Since(t0) < time.Second {
		if time.Since(t0).Milliseconds()/100%2 == 0 {
			if throttle.AllowN(1) {
				allowed++
			} else {
				disallowed++
			}
		}
	}
	if allowed < 250 || allowed > 500 {
		t.Fail()
	}
	fmt.Printf("Allowed %d\nDisallowed %d\n", allowed, disallowed)
}

func Test_Random(t *testing.T) {
	t.Parallel()

	r := rand.New(rand.NewSource(time.Now().UnixMilli()))

	throttle := NewThrottle(10, 5) // 5 ops per 10 ms, or 500 per second
	allowed := 0
	disallowed := 0
	t0 := time.Now()
	for time.Since(t0) < time.Second {
		if r.Intn(5) == 0 {
			if throttle.AllowN(1) {
				allowed++
			} else {
				disallowed++
			}
		}
	}
	if allowed < 500 || allowed > 510 {
		t.Fail()
	}
	fmt.Printf("Allowed %d\nDisallowed %d\n", allowed, disallowed)
}

func Benchmark_Allow(b *testing.B) {
	throttle := NewThrottle(60000, int32(b.N/2))
	for i := 0; i < b.N; i++ {
		throttle.Allow()
	}
}
