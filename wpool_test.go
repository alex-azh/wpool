package wpool

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// Test ensures that the err response for the number 3 will be executed by only one goroutine
func Test(t *testing.T) {
	ctx, _ := context.WithTimeout(context.Background(), time.Second*4)

	wasErr := new(atomic.Bool)
	wasErr.Store(false)
	mu := new(sync.Mutex)
	arr := []int{} // values of parallel workers

	wp := New(
		ctx,
		2,
		func(v int) error {
			fmt.Println("value:", v)
			if v == 3 && wasErr.Load() == false {
				wasErr.Store(true)
				return errors.New("...")
			}
			mu.Lock()
			arr = append(arr, v)
			mu.Unlock()
			return nil
		},
		func() {
			time.Sleep(time.Second)
		},
	)
	wp.Start()

	// 1
	go func() {
		for {
			wp.Push(1)
			time.Sleep(time.Second)
		}
	}()

	// 2
	go func() {
		for {
			wp.Push(2)
			time.Sleep(time.Second)
		}
	}()

	time.Sleep(time.Second)
	wp.Push(3) // must be error

	<-ctx.Done()

	threeCount := 0
	for _, v := range arr {
		if v == 3 {
			threeCount++
		}
	}
	// 3 must be one invoke from errResolver func
	if threeCount != 1 {
		fmt.Println(threeCount)
		t.Error("3 must be 1 count")
	}
}
