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

func goroutineWithIndividualChannel() {
	c1 := make(chan string)
	c2 := make(chan string)
	go func() {
		time.Sleep(1 * time.Second)
		c1 <- "one"
	}()
	go func() {
		time.Sleep(2 * time.Second)
		c2 <- "two"
	}()
	/*
		Here whenever out of msg1 and msg2 comes it will print and exit
		select {
			case msg1 := <-c1:
				fmt.Println("received", msg1)
			case msg2 := <-c2:
				fmt.Println("received", msg2)
		}
	*/
	// It will wait for both. Normally select works whenever any case evaluates true
	// at first 1, any of msg1 and msg2 resolves and then in second iteration the other one would be resolved as the first one already consumed the channel
	for range 2 {
		select {
		case msg1 := <-c1:
			fmt.Println("received", msg1)
		case msg2 := <-c2:
			fmt.Println("received", msg2)
		}
	}
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
