# sync.Cond and its Mutex

Notes on how a condition variable uses the lock passed into it, and why that lock must be held around `Wait()`.

## 1. A Cond holds no state

```go
var mu sync.Mutex
var ready bool              // the actual "condition" lives here, in your variable
cond := sync.NewCond(&mu)   // cond = just a waiting room of parked goroutines
```

`cond` knows nothing about `ready`. Your variables hold the state and the mutex guards them. The Cond takes `&mu` only so that `Wait()` can release and re-take it.

## 2. What `Wait()` does

Go's implementation, slightly trimmed:

```go
func (c *Cond) Wait() {
    t := runtime_notifyListAdd(&c.notify) // 1. register in the waiting room
    c.L.Unlock()                          // 2. release the lock
    runtime_notifyListWait(&c.notify, t)  // 3. sleep until signalled
    c.L.Lock()                            // 4. re-take the lock before returning
}
```

- You must hold the lock when calling `Wait()`, or step 2 fails with `fatal error: sync: unlock of unlocked mutex`.
- It **registers first, then unlocks**. That ordering is what prevents a missed signal.
- It sleeps without the lock, so other goroutines can change the state.
- It returns holding the lock, so you can safely re-check the state.

`Signal()` / `Broadcast()` never touch the lock. They only wake registered waiters.

## 3. Why lock before Wait

**We take the lock before `Wait()` because we first need to check the shared state (a slice, map, queue, flag, ...) to decide whether to wait at all.** That shared state is read and modified by other goroutines, so reading it safely requires the lock that guards it. `Wait()` is then called while still holding that lock, so the check and the sleep happen as one step.

Its not like wait() and signal() (signal doesn't depend on the lock at all) need to sync rather its for the correctness that wait() is called after checking the shared state.
So we take lock so now other external can't mutate. Then we do wait() so that the goroutine can go to sleep. wait() releases the lock before sleeping so that other worker can check it

Like this almost
```
consumer
lock
-> check shared state 
+ wait()
-> wait() releases the lock
```

```
producer
lock
-> to check the state
but get locked as the shared is lock
-> when lock released
it calls signal
```

From here only we can connect why and how condition sharing the same mutex works
Its solely for the correctness of the shared state and wait() always release the lock
So if conditions are for the same shared state so the same single lock would be taken right so a single mutex for both the condition
so checks and wait are done with on the same lock only

```go
mu.Lock()                 // lock to read the shared state safely
for len(items) == 0 {     // check the shared state (slice/map/queue)
    cond.Wait()           // not ready: register, release the lock, sleep
}
// state is ready and we still hold the lock, so use it
mu.Unlock()
```

### The lost wakeup

Without the lock, "check the state" and "register in the waiting room" are two separate steps with a gap between them:

```
G1 (waiter):  checks ready → false
G2 (changer): ready = true; Signal()   ← nobody registered yet, so the signal is lost
G1 (waiter):  registers, sleeps        ← waits forever
```

The state is fine (`ready` is true), but G1 never notices and hangs.

With the lock:

```go
// Waiter
mu.Lock()
for !ready {
    cond.Wait()   // registers, THEN unlocks
}
// ready is true here and we hold mu
mu.Unlock()

// Changer
mu.Lock()
ready = true
mu.Unlock()
cond.Signal()
```

The changer can only write `ready` while holding `mu`, and the waiter holds `mu` from its check until it is registered. So the write happens either:

| Case | Result |
|---|---|
| before the waiter checks | the waiter sees `true` and never sleeps |
| after the waiter has registered | the signal reaches the waiter |

The lock makes "check + register" (waiter) and "change state" (changer) mutually exclusive, so the state can't change in the gap.

**Correction to a common belief:** you don't have to hold the lock when calling `Signal()` (Go docs: "allowed but not required"). What must happen under the lock is the **state change**. A wakeup is lost when the state is changed without the lock, not when `Signal` is called without it.

## 4. Why `for`, not `if`

Between being woken (step 3) and getting the lock back (step 4), another goroutine may take the lock and change the state again. With `Broadcast`, several waiters compete for it. A wakeup means "something changed, go check", not "your condition is true". So re-check in a loop.

## 5. It must be the same lock that guards the state

Could `ready` be guarded by a different `mu2` while the Cond uses `mu`? No:

- **Hold `mu2` while waiting:** `Wait` releases `mu`, not `mu2`. The waiter sleeps holding `mu2`, the changer blocks on `mu2` forever, and you get a deadlock.
- **Release `mu2` before waiting:** the gap between check and register is back, and so is the lost wakeup.

`Wait()` can only release the lock it was given. **Rule: the lock passed to the Cond must be the lock that guards the state the condition reads.**

## 6. Two Conds, one mutex (notEmpty / notFull)

```go
lock     := &sync.Mutex{}
notEmpty := sync.NewCond(lock)  // consumers wait here
notFull  := sync.NewCond(lock)  // producers wait here

func produce(item int) {
    lock.Lock()
    for len(items) == size {
        notFull.Wait()
    }
    items = append(items, item)
    lock.Unlock()
    notEmpty.Signal()           // added an item, so wake a CONSUMER
}

func consume() int {
    lock.Lock()
    for len(items) == 0 {
        notEmpty.Wait()
    }
    item := items[0]
    items = items[1:]
    lock.Unlock()
    notFull.Signal()            // freed a slot, so wake a PRODUCER
    return item
}
```

- **Sharing a mutex does not connect the Conds.** Each Cond has its own waiting room. `notFull.Signal()` only wakes `notFull` waiters. The cross-signalling comes from your code.
- **Same mutex because it's the same state.** Both conditions read `len(items)`, so both Conds must use the lock that guards `items`. This is a correctness requirement, not a convenience. With separate locks, a producer and a consumer could both modify `items` at once (a data race), and checks and changes would no longer be under the same lock (lost wakeups).
- **Two Conds instead of one** so that each `Signal()` wakes the right side. With one Cond, `Signal()` might wake another consumer instead of a producer, and things can stall. A single-Cond design has to use `Broadcast()`, which wakes everyone.

## 7. What goes wrong without the lock

| Problem | Cause | Effect |
|---|---|---|
| Lost wakeup | the state changes in the gap between check and register | the waiter sleeps forever |
| Data race | two goroutines modify shared data at once | corrupted slice, wrong values |

## 8. The full flow

```
CONSUMER                                  PRODUCER
lock()
check state: empty
Wait():
  register in the waiting room
  release the lock  ─────────────────►    lock() was blocked, now gets it
  sleep                                   check state: has space
                                          modify state (append item)
                                          unlock()
                                          Signal()
  wake up  ◄─────────────────────────────
  re-take the lock
loop: check state again → has an item
take the item
unlock()
```

The lock moves back and forth between the two sides. `Wait()` hands it over while the waiter sleeps and gets it back on wakeup.

## Summary

| Who | Must do under the lock |
|---|---|
| Waiter | lock to check the shared state (slice/map/queue) → `Wait()` → re-check the state |
| Changer | modify the state (`Signal` can go inside or outside) |
| `Wait()` itself | register → **unlock** → sleep → **re-lock** |
