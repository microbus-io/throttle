# Throttle

`Throttle` is a rate limiter that implements the sliding window algorithm.
It uses two counters - one for the current fixed window and one for the previous fixed window - to estimate the load of operations in the sliding window.
It is therefore not 100% accurate but it uses very little memory (under 64 bytes).  

```go
// Create a new throttle allowing 20 ops/second
th := throttle.New(time.Second, 20)

// Check if op is allowed
if th.Allow() {
    ...
}
```

Rate limiting is a technique that controls the rate of requests sent or received by a network, server, or other resource.
There are four common algorithms for rate limiting, each with its own pros and cons:

* Leaky bucket
* Token bucket
* Fixed window 
* Sliding window

There is a standard library implementation of the [token bucket algorithm](https://pkg.go.dev/golang.org/x/time/rate).

`Throttle` is licensed by Microbus LLC under the [Apache License 2.0](http://www.apache.org/licenses/LICENSE-2.0).

Inspired by ["Rate Limiter — Sliding Window Counter"](https://medium.com/@avocadi/rate-limiter-sliding-window-counter-7ec08dbe21d6).
