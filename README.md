# Throttle

`Throttle` is a rate limiter that implements the sliding window counter algorithm.
It uses two counters - one for the current fixed window and one for the previous fixed window - to estimate the load of operations in the sliding window.
It uses very little memory (under 64 bytes) but is not 100% accurate. Data suggests 0.003% of requests may be incorrectly categorized.

```go
// Create a new throttle allowing 20 ops/second
th := throttle.New(time.Second, 20)

// Check if op is allowed
if th.Allow() {
    ...
}
```

Rate limiting is a technique that controls the rate of requests sent or received by a network, server, or other resource.
There are a few common algorithms for rate limiting, each with its own pros and cons:

* Leaky bucket
* Token bucket - implemented in the [standard library](https://pkg.go.dev/golang.org/x/time/rate)
* Fixed window counter
* Sliding window log
* Sliding window counter - this library

`Throttle` is licensed by Microbus LLC under the [Apache License 2.0](http://www.apache.org/licenses/LICENSE-2.0).

Inspired by ["Rate Limiter — Sliding Window Counter"](https://medium.com/@avocadi/rate-limiter-sliding-window-counter-7ec08dbe21d6).
