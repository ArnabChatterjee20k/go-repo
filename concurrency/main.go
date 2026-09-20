package main

import (
	"fmt"
	"strconv"
	"time"
)

func process(id int, channel chan string) {
	fmt.Println("Processing ", id)
	time.Sleep(2 * time.Millisecond)
	channel <- strconv.Itoa(id)
}

func syncGoRoutinesWithChannels() {
	channel := make(chan string, 2)
	go process(1, channel)
	go process(2, channel)
	fmt.Println(<-channel)
	fmt.Println(<-channel)
}

// sendThenReport sends its id, then announces it finished sending.
// Whether "finished sending" prints early or late reveals if the send blocked.
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
	time.Sleep(10 * time.Millisecond) // let any late prints flush
}

// main function in Go runs in its own goroutine, which is known as the main goroutine
func main() {
	syncGoRoutinesWithChannels()

	fmt.Println()
	demoChannelBuffering(0) // unbuffered: "finished sending" appears late
	fmt.Println()
	demoChannelBuffering(2) // buffered: "finished sending" appears early
}
