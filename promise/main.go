package main

import (
	"fmt"
	"sync"
	"time"
)

// TODO: A promise like js with options for running multiple promises with allsettled, race with rate limiting on the number of executions

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
