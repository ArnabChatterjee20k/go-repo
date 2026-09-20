package main

import (
	"errors"
	"testing"
	"time"
)

// delayedResolve returns a promise that resolves with v after d.
func delayedResolve(P *Promises, d time.Duration, v any) *Promise {
	return P.New(func(resolve func(any), reject func(error)) {
		time.Sleep(d)
		resolve(v)
	})
}

// delayedReject returns a promise that rejects with err after d.
func delayedReject(P *Promises, d time.Duration, err error) *Promise {
	return P.New(func(resolve func(any), reject func(error)) {
		time.Sleep(d)
		reject(err)
	})
}

// The fastest promise wins, even though slower ones also settle afterwards.
func TestRaceFastestWins(t *testing.T) {
	var P Promises
	p := P.Race(
		delayedResolve(&P, 60*time.Millisecond, "slow"),
		delayedResolve(&P, 10*time.Millisecond, "fast"),
		delayedResolve(&P, 40*time.Millisecond, "medium"),
	)
	v, err := p.Await()
	if err != nil {
		t.Fatalf("err = %v; want nil", err)
	}
	if v != "fast" {
		t.Errorf("Race value = %v; want fast", v)
	}
}

// Race settles on the first to FINISH — even if that first one is an error
// (this is Promise.race semantics, not Promise.any).
func TestRaceFirstIsError(t *testing.T) {
	var P Promises
	boom := errors.New("boom")
	p := P.Race(
		delayedResolve(&P, 50*time.Millisecond, "would-succeed"),
		delayedReject(&P, 10*time.Millisecond, boom), // finishes first
	)
	v, err := p.Await()
	if !errors.Is(err, boom) {
		t.Errorf("err = %v; want %v", err, boom)
	}
	if v != nil {
		t.Errorf("value = %v; want nil when the first settle is an error", v)
	}
}

// The slower promises settling after the winner must not change the result
// (settle's Once drops them) and must not panic under -race.
func TestRaceLaterSettlersAreDropped(t *testing.T) {
	var P Promises
	p := P.Race(
		delayedResolve(&P, 10*time.Millisecond, 1), // winner
		delayedResolve(&P, 20*time.Millisecond, 2),
		delayedReject(&P, 30*time.Millisecond, errors.New("late")),
	)
	v, err := p.Await()
	if v != 1 || err != nil {
		t.Fatalf("Race = %v, %v; want 1, nil", v, err)
	}
	// give the losers time to settle-and-be-dropped; must stay 1 and not panic
	time.Sleep(50 * time.Millisecond)
	v2, err2 := p.Await()
	if v2 != 1 || err2 != nil {
		t.Errorf("after losers settled: %v, %v; want 1, nil", v2, err2)
	}
}
