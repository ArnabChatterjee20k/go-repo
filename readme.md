## Go: Values, References & Slices

Go is **always pass-by-value**. Function arguments are copied, but the copied value may point to shared underlying data.

```text
                 Function argument
                       │
                       ▼
                   COPY IT
                       │
          ┌────────────┼────────────┐
          │            │            │
       int/struct    pointer    slice/map
          │            │            │
      independent   shared data  shared data
```

### Slices and `make`

A slice is a small header containing a pointer to an underlying array, its `len`, and `cap`.

```go
s := make([]int, 3, 10)
```

```text
s
┌──────────────┐
│ ptr ─────────┼──────► [0 0 0 _ _ _ _ _ _ _]
│ len = 3      │
│ cap = 10     │
└──────────────┘
```

* `len` = number of elements currently in the slice.
* `cap` = how many elements can fit in the underlying array before a new array may be allocated.
* Passing a slice copies the **slice header**, not the underlying array.
* Therefore, two slices can share the same backing array.

```go
s := make([]int, 3, 10)
t := s

t[0] = 100
// s[0] is also 100
```

### `append`

`append` always returns a **slice**, but it may or may not allocate a new backing array.

```go
s = append(s, 100)
```

* If `cap(s)` is sufficient → the backing array can be reused.
* If capacity is insufficient → a new backing array is allocated and the data is copied.

This is why a slice argument can mutate the caller's data, while `append` can cause the function's slice to point to a completely different backing array.


## Go: Strings, Bytes & Runes

* `string` is an **immutable sequence of bytes**, usually UTF-8 encoded.
* `byte` is an alias for `uint8` → represents one raw byte.
* `rune` is an alias for `int32` → represents a Unicode **code point**.

```text
string
  │
  ▼
UTF-8 encoded bytes
  │
  ▼
Unicode code points (runes)
```

### Example

```go
s := "你好"

len(s)           // 6 → number of bytes
len([]rune(s))   // 2 → number of Unicode code points
```

Use `range` to iterate over runes:

```go
for _, r := range s {
    fmt.Printf("%c\n", r)
}
```

While:

```go
for i := 0; i < len(s); i++ {
    // s[i] is a byte, not a rune
}
```

**Remember:**

```text
string → bytes
byte   → uint8
rune   → int32 / Unicode code point
```

A rune isn't necessarily a user-perceived "character"; a single visible character can consist of multiple Unicode code points.

## Property accessibility in structs
We dont have private/public
We have exported and unexported
```
package main

import "myapp/trie"

func main() {
    t := trie.New()
    t.Remove("foo") // ✅ exported
    t.remove("foo") // ❌ unexported
}
```
lowercase → unexported/private to package
Uppercase → exported/public

Same for attributes as well

## Go: Channels (Buffered vs Unbuffered)

In unbuffered channels they are adding to the channel only if there is a receiver, as cap stays zero (no slot to store in — the value is handed straight to the receiver). Buffered channels have slots, so a sender can drop the value and continue.

```go
func sendThenReport(id int, channel chan string) {
	channel <- strconv.Itoa(id)
	fmt.Println("  process", id, "finished sending")
}

func demoChannelBuffering(capacity int) {
	fmt.Printf("--- channel capacity %d ---\n", capacity)
	channel := make(chan string, capacity)
	go sendThenReport(1, channel)
	go sendThenReport(2, channel)

	time.Sleep(500 * time.Millisecond) // deliberately do NOT receive yet
	fmt.Println("  main woke up, now receiving")
	fmt.Println("  got:", <-channel)
	fmt.Println("  got:", <-channel)
}

demoChannelBuffering(0) // unbuffered
demoChannelBuffering(2) // buffered
```

Output:

```text
--- channel capacity 0 ---
  main woke up, now receiving   ← receives happen FIRST
  got: 2
  got: 1
  process 1 finished sending    ← senders only finish AFTER a receiver arrives
  process 2 finished sending

--- channel capacity 2 ---
  process 2 finished sending    ← senders finish IMMEDIATELY (dropped in buffer)
  process 1 finished sending
  main woke up, now receiving
  got: 2
  got: 1
```

* **cap 0 (unbuffered)** → meeting point; send waits for a receiver.
* **cap > 0 (buffered)** → mailbox with N slots; send drops the value and continues until full.
* A **closed** channel never blocks a receiver — `close(done)` broadcasts "done" to all waiters (the basis of a settled Promise/Future).

## If else
```
func (node *Node) hasNode(topic string) bool {
	/*
		if <statement>; <condition> {
			...
		}
		node.nodes[topic] gives val, existenceBool
		so if exists
	*/
	if _, ok := node.nodes[topic]; ok {
		return true
	}
	return false
}
```

## Channel direction (<-chan vs chan<-)
```
func pong(pings <-chan string, pongs chan<- string) {
    msg := <-pings
    pongs <- msg
}
```
The pong function accepts one channel for receives (pings) and a second for sends (pongs).