# Promise — Concurrency Interview Study Sheet

A JS-style Promise built on Go primitives. Every question below maps to real code
in `main.go` / the `*_test.go` files. Verify any claim with:

```sh
go test -race ./promise
```

The through-line: **a channel makes a value safe to *read* (happens-before); a
`sync.Once` makes a settle safe to *write once*.** Different jobs, different tools.

---

## Channels & blocking

### 1. `nil` channel vs `closed` channel — send / receive / close?

|          | send            | receive                  | close       |
| -------- | --------------- | ------------------------ | ----------- |
| `nil`    | blocks forever  | blocks forever           | **panics**  |
| `closed` | **panics**      | zero value, instantly    | **panics**  |

We hit `close of nil channel` because `&Promise{}` left `fulfilled` as `nil` —
channels must be `make()`'d before use.

**Curious follow-up:** when is "block forever on a nil channel" useful?
→ Setting a `select` case's channel to `nil` disables that case dynamically.

### 2. Why doesn't receiving from a closed channel block — and how do we use it?

`Await()` does `<-fulfilled`. `settle` calls `close(fulfilled)`. A closed channel
never blocks a receiver, so **every** waiting `Await()` unblocks at once, and any
later `Await()` returns instantly. `close` = a one-time broadcast of "settled".

### 3. Unbuffered vs buffered — when does a *send* block?

- **Unbuffered:** send blocks until a receiver is *simultaneously* ready (rendezvous).
- **Buffered:** send blocks only when the buffer is full.

The semaphore in `AllSettled` is a buffered channel used as a token pool.

---

## The memory model (the deep ones)

### 4. Write a field, then `close(done)`; reader reads the field after `<-done`. Race?

**No.** The write happens-before the `close`, and the `close` happens-before the
receive — so the write is visible to the reader. This is the ordering rule that
makes the whole design safe.

**The bug we actually fixed:** the first version did

```go
close(fulfilled)   // wakes Await FIRST
value = v          // ...writes AFTER → data race
```

`-race` flagged it. Correct order is **write first, close last**:

```go
value = v
err = e
close(fulfilled)   // barrier: everything before is now visible to receivers
```

### 5. Does `sync.Once` give a happens-before guarantee, or just mutual exclusion?

**Both.** Everything inside the first `Do(f)` happens-before any later `Do`
returns. That's why `settle` can write `value`/`err` inside `once.Do` and readers
(after `<-fulfilled`) are guaranteed to see them.

---

## sync vs channels

### 6. We already have `close(done)` for signaling. Why *also* `sync.Once`?

They solve different problems:
- **channel** → makes the value safe to *read* (happens-before barrier).
- **`Once`** → makes the settle safe to *write exactly once* — a channel can't
  express "first writer wins, drop the rest."

Without `Once`, a second settle = second `close` = `close of closed channel` panic.
`TestDoubleSettleFirstWins` proves the first settle wins and the rest are no-ops.

### 7. The classic `WaitGroup` bug — reproduce it.

```go
for _, p := range ps {
    go func() {
        wg.Add(1)      // ✗ Wait() may run before any goroutine starts → sees 0 → returns early
        defer wg.Done()
    }()
}
wg.Wait()
```

Rule: **`Add` before you launch (with the known count), `Done` inside with `defer`.**
`AllSettled` does `wg.Add(len(...))` up front — correct.

### 8. Two goroutines write `results[i]` and `results[j]` (`i≠j`). Race?

**No** — distinct memory locations, no mutex needed. This is why `AllSettled`
pre-sizes the slice and gives each goroutine its own index. `append` from multiple
goroutines *would* race (it read/writes the shared header).

---

## Goroutine lifecycle & leaks

### 9. In `Race`, losers keep running after the winner settles. Leak?

The loser goroutines finish on their own (their `Await` returns, their settle is a
no-op). But if a promise **never** settles, its `Await` goroutine blocks forever —
a leak. Fix direction: **`context.Context`** or a cancel channel in a `select`, so
`Await` can give up. (Natural next feature.)

### 10. The pre-Go-1.22 loop-variable capture bug — is it in our code?

```go
for i, p := range ps { go func(){ use(i) }() }
```

Pre-1.22: all goroutines saw the **last** `i`. Go 1.22+: per-iteration, so it's
safe. Our `go.mod` is 1.27, so we're fine — but know the fix (`i := i`, or 1.22).

---

## Design / semantics

### 11. Why does `AllSettled` take **factories** but `Race` takes **live promises**?

Rate limiting must throttle *starts* — and a live `*Promise` already started at
`New()`. So to cap concurrency you need `func() *Promise` (deferred start) and call
it under a semaphore. `Race` wants **maximum** concurrency (everyone races from the
gun), so live promises are correct there.

| | starts work at | to rate-limit? |
| --- | --- | --- |
| live `*Promise` | `New()`, immediately | impossible (already running) |
| `func() *Promise` | when the factory is **called** | yes — call under a semaphore |

`TestAllSettledRespectsLimit` proves `max concurrent <= limit` — only possible
because factories defer the start.

### 12. Map each JS combinator to a Go primitive.

| JS | settles when | Go primitive |
| --- | --- | --- |
| `all` | first error, else all done | `Once` + success counter |
| `allSettled` | everyone finished | **`WaitGroup`** |
| `race` | first to *settle* (value or error) | **`Once`** |
| `any` | first *success* | `Once` + error counter |

`race` vs `any`: `race` accepts the first *error* as the result; `any` ignores
errors until a success.

---

## The one to be able to answer cold

> "`Await` does `<-fulfilled` then reads the field. On the Go memory model, why is a
> reader on a *different* goroutine guaranteed to see the value the writer set — and
> what one-line change breaks it?"

**Answer:** `close(fulfilled)` happens-before the `<-fulfilled` receive, and the
field write happens-before the close, so the value is visible. **Breaks it:** move
the `close` *before* the field write — now the receiver can read a stale/zero value,
a data race. We found exactly this with `-race` and fixed it by writing before
closing.
