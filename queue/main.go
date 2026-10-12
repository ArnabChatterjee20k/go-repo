package main

import (
	"fmt"
	"sync"
	"time"
)

type Queue struct {
	items             []int
	size              int
	lock              *sync.Mutex
	notEmptyCondition *sync.Cond
	notFullCondition  *sync.Cond
}

func CreateQueue(size int) Queue {
	lock := sync.Mutex{}
	// two condition otherwise a producer can wake up a producer instead of a consumer and vice versa
	notEmptyCondition := sync.NewCond(&lock)
	notFullCondition := sync.NewCond(&lock)
	return Queue{items: []int{}, size: size, lock: &lock, notEmptyCondition: notEmptyCondition, notFullCondition: notFullCondition}
}

func (queue *Queue) produce(item int) {
	queue.lock.Lock()
	for {
		if len(queue.items) != queue.size {
			break
		}
		// will release the above queue lock before sleeping
		queue.notFullCondition.Wait()
	}
	queue.items = append(queue.items, item)
	// doing manual unlocks instead of defer so that we broadcast outside the lock
	queue.lock.Unlock()
	// we dont need to do it inside lock as singalling is independent of the lock
	// using signal instesad of the broadcast so that one consumer works instead of multiple consumers and checking and going back to sleep
	queue.notEmptyCondition.Signal()
}

func (queue *Queue) consume() int {
	queue.lock.Lock()
	for {
		if len(queue.items) != 0 {
			break
		}
		queue.notEmptyCondition.Wait()
	}
	item := queue.items[0]
	queue.items = queue.items[1:]
	queue.lock.Unlock()
	queue.notFullCondition.Signal()
	return item
}

func main() {
	queue := CreateQueue(10)
	go func() {
		fmt.Println(queue.consume())
	}()

	queue.produce(12)

	time.Sleep(2 * time.Second)
}
