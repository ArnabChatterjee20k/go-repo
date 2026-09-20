package main

import (
	"fmt"
	"sync"
	"time"
)

type PromiseCallback func(resolve func(any), reject func(error))
type Callback func(any) (any, error)

type Promise struct {
	value any
	err   error
	// event signal
	fulfilled chan struct{}
	// guards settle so it runs exactly once (first settle wins)
	once sync.Once
}

type Promises struct{}

type Result struct {
	value any
	err   error
}

func (promises *Promises) New(cb PromiseCallback) *Promise {
	promise := &Promise{fulfilled: make(chan struct{})}
	go cb(
		// resolve
		func(v any) {
			promise.settle(v, nil)
		},
		// reject
		func(v error) {
			promise.settle(nil, v)
		},
	)
	return promise
}

func (promises *Promises) AllSettled(batch int, promiseFactory ...func() *Promise) *Promise {
	return promises.New(func(resolve func(any), reject func(error)) {
		results := make([]Result, len(promiseFactory))
		var wg sync.WaitGroup
		semaphore := make(chan struct{}, batch)
		wg.Add(len(promiseFactory))
		for i, promiseFunc := range promiseFactory {
			semaphore <- struct{}{}
			go func() {
				defer wg.Done()
				// defer can only be used with function call only
				defer func() { <-semaphore }()
				result, err := promiseFunc().Await()
				// no mutex needed as the results are created with the length of the promises only and get to their own index slot only
				results[i] = Result{value: result, err: err}
			}()
		}
		wg.Wait()
		// we dont need close as no one is reading it yet and wait group already settling the concurrency completion
		// close(semaphore)
		resolve(results)
	})
}

func (promises *Promises) Race(promiseArray ...*Promise) *Promise {
	return promises.New(func(resolve func(any), reject func(error)) {
		// resolve, reject already having the settle with once. so the goroutine calling either of them returns the promise
		for _, promiseFunc := range promiseArray {
			go func() {
				result, err := promiseFunc.Await()
				if err != nil {
					reject(err)
				} else {
					resolve(result)
				}
			}()
		}
	})
}

func (promise *Promise) Then(cb Callback) *Promise {
	promise.Await()
	result, err := cb(promise.value)
	if err == nil {
		promise.value = result
		return promise
	}
	return promise
}

func (promise *Promise) Catch(cb Callback) *Promise {
	result, err := cb(promise.value)
	if err != nil {
		promise.err = err
		return promise
	}
	promise.value = result
	return promise
}

/*
A settle method so that if both resolve and reject get called one after the other
then the close(promise.fulfilled) would try to close the fulfilled channel twice giving panic
sync.Once would maintain the lifecycle
*/
func (promise *Promise) settle(val any, err error) {
	promise.once.Do(func() {
		promise.value = val
		promise.err = err
		close(promise.fulfilled)
	})
}

func (promise *Promise) Await() (any, error) {
	<-promise.fulfilled
	return promise.value, promise.err
}

func (promise *Promise) Finally(cb Callback) Promise {
	result, err := cb(promise.value)
	return Promise{value: result, err: err}
}

func main() {
	promises := Promises{}
	result, _ := promises.New(func(resolve func(any), reject func(error)) {
		time.Sleep(2 * time.Second)
		resolve(30)
	}).Then(func(value any) (any, error) {
		switch v := value.(type) {
		case int:
			return v + 10, nil
		default:
			return nil, fmt.Errorf("unexpected type: %T", v)
		}
	}).Await()
	fmt.Println(result)
}
