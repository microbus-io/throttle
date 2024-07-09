# Throttle

`Throttle` is a rate limiter that implements the sliding window algorithm.

`Throttle` was designed with both memory efficiency and performance in mind.
It avoids locking by using `atomic` integers and takes just about 32 bytes of memory.
A call to `Allow` completes in approximately 50 nanoseconds.

Rate limiting is a technique that controls the rate of requests sent or received by a network, server, or other resource.
There are four common algorithms for rate limiting, each with its own pros and cons:

* Leaky bucket
* Token bucket
* Fixed window 
* Sliding window

There is a standard library implementation of the [token bucket algorithm](https://pkg.go.dev/golang.org/x/time/rate).

`Throttle` is licensed by Microbus LLC under the [Apache License 2.0](http://www.apache.org/licenses/LICENSE-2.0).
