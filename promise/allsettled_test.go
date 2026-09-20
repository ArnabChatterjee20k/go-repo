package main

import (
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

// makeFactory returns a factory (not a running promise). It records the peak
// number of promises running at once so we can assert the semaphore caps it.
func makeFactory(id int, running, maxRunning *int32) func() *Promise {
	var P Promises
	return func() *Promise {
		return P.New(func(resolve func(any), reject func(error)) {
			cur := atomic.AddInt32(running, 1)
			for {
				m := atomic.LoadInt32(maxRunning)
				if cur <= m || atomic.CompareAndSwapInt32(maxRunning, m, cur) {
					break
				}
			}
			time.Sleep(30 * time.Millisecond)
			atomic.AddInt32(running, -1)
			if id%2 == 0 {
				resolve(id)
			} else {
				reject(errors.New("odd"))
			}
		})
	}
}

func TestAllSettledResults(t *testing.T) {
	var P Promises
	var running, maxRunning int32

	n := 6
	factories := make([]func() *Promise, n)
	for i := range n {
		factories[i] = makeFactory(i, &running, &maxRunning)
	}

	res, err := P.AllSettled(2, factories...).Await()
	if err != nil {
		t.Fatalf("AllSettled must never reject; got %v", err)
	}
	results := res.([]Result)
	if len(results) != n {
		t.Fatalf("got %d results; want %d", len(results), n)
	}
	for i, r := range results {
		if i%2 == 0 && (r.value != i || r.err != nil) {
			t.Errorf("results[%d] = %+v; want value=%d, no err", i, r, i)
		}
		if i%2 == 1 && r.err == nil {
			t.Errorf("results[%d] = %+v; want an error", i, r)
		}
	}
}

// The whole point of the factory design: the semaphore must cap how many
// promises run at once.
func TestAllSettledRespectsLimit(t *testing.T) {
	var P Promises
	var running, maxRunning int32

	n, limit := 6, 2
	factories := make([]func() *Promise, n)
	for i := range n {
		factories[i] = makeFactory(i, &running, &maxRunning)
	}

	P.AllSettled(limit, factories...).Await()

	if got := atomic.LoadInt32(&maxRunning); got > int32(limit) {
		t.Errorf("max concurrent = %d; must be <= limit %d", got, limit)
	} else {
		t.Logf("limit=%d, n=%d -> max concurrent = %d", limit, n, got)
	}
}
