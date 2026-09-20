package main

import (
	"errors"
	"sync"
	"testing"
	"time"
)

// helper: a Promises factory (the receiver is unused, so a zero value is fine).
var P Promises

// --- settle / Await basics ---------------------------------------------------

func TestResolve(t *testing.T) {
	p := P.New(func(resolve func(any), reject func(error)) {
		resolve(30)
	})
	v, err := p.Await()
	if err != nil {
		t.Fatalf("Await err = %v; want nil", err)
	}
	if v != 30 {
		t.Errorf("Await value = %v; want 30", v)
	}
}

func TestReject(t *testing.T) {
	boom := errors.New("boom")
	p := P.New(func(resolve func(any), reject func(error)) {
		reject(boom)
	})
	v, err := p.Await()
	if !errors.Is(err, boom) {
		t.Errorf("Await err = %v; want %v", err, boom)
	}
	if v != nil {
		t.Errorf("Await value = %v; want nil on rejection", v)
	}
}

// Await must block until the async work finishes, not return early.
func TestAwaitBlocksUntilSettled(t *testing.T) {
	start := time.Now()
	p := P.New(func(resolve func(any), reject func(error)) {
		time.Sleep(50 * time.Millisecond)
		resolve("done")
	})
	v, _ := p.Await()
	if elapsed := time.Since(start); elapsed < 50*time.Millisecond {
		t.Errorf("Await returned after %v; should have blocked ~50ms", elapsed)
	}
	if v != "done" {
		t.Errorf("value = %v; want done", v)
	}
}

// Once settled, Await returns instantly and can be called many times (closed
// channel never blocks a receiver).
func TestAwaitAfterSettleIsInstantAndRepeatable(t *testing.T) {
	p := P.New(func(resolve func(any), reject func(error)) {
		resolve(7)
	})
	p.Await() // settle it

	start := time.Now()
	for range 3 {
		v, err := p.Await()
		if v != 7 || err != nil {
			t.Fatalf("repeat Await = %v, %v; want 7, nil", v, err)
		}
	}
	if elapsed := time.Since(start); elapsed > 10*time.Millisecond {
		t.Errorf("repeat Awaits took %v; should be ~instant", elapsed)
	}
}

// --- settle-exactly-once -----------------------------------------------------

// resolve then reject: the first settle wins, the second is a silent no-op
// (no "close of closed channel" panic).
func TestDoubleSettleFirstWins(t *testing.T) {
	p := &Promise{fulfilled: make(chan struct{})}
	p.settle(30, nil)
	p.settle(nil, errors.New("boom")) // must be ignored, not panic

	v, err := p.Await()
	if v != 30 || err != nil {
		t.Fatalf("first settle should win: got %v, %v; want 30, nil", v, err)
	}
}

// Many goroutines settle the same promise at once: exactly one wins, no panic,
// no data race (run with -race).
func TestConcurrentSettleIsRaceSafe(t *testing.T) {
	p := &Promise{fulfilled: make(chan struct{})}

	const n = 50
	var wg sync.WaitGroup
	wg.Add(n)
	for i := range n {
		go func() {
			defer wg.Done()
			p.settle(i, nil)
		}()
	}
	wg.Wait()

	v, err := p.Await()
	if err != nil {
		t.Fatalf("err = %v; want nil", err)
	}
	got, ok := v.(int)
	if !ok || got < 0 || got >= n {
		t.Errorf("winning value = %v; want one of 0..%d", v, n-1)
	}
}

// --- Then chaining -----------------------------------------------------------

func TestThenTransformsValue(t *testing.T) {
	v, err := P.New(func(resolve func(any), reject func(error)) {
		resolve(30)
	}).Then(func(value any) (any, error) {
		return value.(int) + 10, nil
	}).Await()

	if err != nil {
		t.Fatalf("err = %v; want nil", err)
	}
	if v != 40 {
		t.Errorf("value = %v; want 40", v)
	}
}

func TestThenChaining(t *testing.T) {
	v, _ := P.New(func(resolve func(any), reject func(error)) {
		resolve(1)
	}).Then(func(value any) (any, error) {
		return value.(int) * 2, nil // 2
	}).Then(func(value any) (any, error) {
		return value.(int) + 5, nil // 7
	}).Await()

	if v != 7 {
		t.Errorf("chained value = %v; want 7", v)
	}
}
