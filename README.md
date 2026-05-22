# Throttle

`Throttle` is a rate limiter that implements the sliding window counter algorithm.
It uses two counters - one for the current fixed window and one for the previous fixed window - to estimate the load of operations in the sliding window.
It uses very little memory (under 64 bytes) but is not 100% accurate. Data suggests 0.003% of requests may be incorrectly categorized.

```go
// Create a new throttle allowing 20 ops/second
th := throttle.New(time.Second, 20)

// Check if op is allowed
admit, observed := th.Allow()
if admit {
    ...
}

// Check without consuming a slot
if admit, _ := th.Peek(); admit {
    ...
}

// Adjust the limit at runtime; counters are preserved
th.SetLimit(50)
```

`Allow` and `AllowN` return the admission decision along with the sliding window's observed load *measured before the call* - useful for anchoring a downstream rejection (e.g. a 429) to the actual emitted rate. Add the weight to `observed` if you want the post-call load.
`Peek` and `PeekN` answer the same question without modifying state.
`SetLimit` and `Limit` get and set the limit at runtime without resetting the window.

Rate limiting is a technique that controls the rate of requests sent or received by a network, server, or other resource.
There are a few common algorithms for rate limiting, each with its own pros and cons:

* Leaky bucket
* Token bucket - implemented in the [standard library](https://pkg.go.dev/golang.org/x/time/rate)
* Fixed window counter
* Sliding window log
* Sliding window counter - this library

`Throttle` is licensed by Microbus LLC under the [Apache License 2.0](http://www.apache.org/licenses/LICENSE-2.0).

Inspired by ["Rate Limiter — Sliding Window Counter"](https://medium.com/@avocadi/rate-limiter-sliding-window-counter-7ec08dbe21d6).
