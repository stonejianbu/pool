package pool

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"runtime"
	"sync"
	"testing"
	"time"
)

func handleFunc(ctx context.Context, msg interface{}) error {
	time.Sleep(time.Millisecond * 10)
	return nil
}

func errorHandleFunc(ctx context.Context, msg interface{}) error {
	time.Sleep(time.Millisecond * 10)
	if rand.Intn(100) == 10 {
		return errors.New("error")
	}
	return nil
}

func panicHandleFunc(ctx context.Context, msg interface{}) error {
	time.Sleep(time.Millisecond * 10)
	if rand.Intn(100) == 10 {
		panic("panic error")
	}
	return nil
}

func TestGo(t *testing.T) {
	startTime := time.Now()
	taskParams := make([]interface{}, 100000)
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, time.Second*30)
	defer cancel()
	err := Go(ctx, 10000, handleFunc, taskParams...)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Printf("耗时：%0.2f\n", time.Since(startTime).Seconds())
}

func TestGo_Error(t *testing.T) {
	startTime := time.Now()
	taskParams := make([]interface{}, 1000)
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, time.Second*30)
	defer cancel()
	err := Go(ctx, 10, errorHandleFunc, taskParams...)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Printf("耗时：%0.2f\n", time.Since(startTime).Seconds())
}

func TestGo_Panic(t *testing.T) {
	startTime := time.Now()
	taskParams := make([]interface{}, 1000)
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, time.Second*30)
	defer cancel()
	err := Go(ctx, 10, panicHandleFunc, taskParams...)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Printf("耗时：%0.2f\n", time.Since(startTime).Seconds())
}

func TestPool(t *testing.T) {
	ctx := context.Background()
	p := NewPool(ctx, WithSize(5), WithIgnoreErr())
	p.RegisterHandleFunc(handleFunc)
	p.Submit(ctx, []interface{}{"liming", "stone", "alice", "mery", "miss", "lucy", "foo", "nvd"}...)
	err := p.Wait()
	if err != nil {
		t.Errorf("err: %v", err)
		return
	}
}

func Benchmark_Pool_1(b *testing.B) {
	num := 1000000
	taskParams := make([]interface{}, num)
	ctx := context.Background()
	gNum := uint(num / 10)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Go(ctx, gNum, handleFunc, taskParams...)
	}
}

func Benchmark_Pool_2(b *testing.B) {
	num := 1000000
	var m runtime.MemStats
	taskParams := make([]interface{}, num)
	ctx := context.Background()
	gNum := uint(num / 10)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Go(ctx, gNum, handleFunc, taskParams...)
	}
	runtime.ReadMemStats(&m)
	afterAlloc := m.Alloc
	b.Logf("pool Memory allocated: %d bytes\n", afterAlloc)
}

func Benchmark_Goroutine(b *testing.B) {
	num := 1000000
	var m runtime.MemStats
	taskParams := make([]interface{}, num)
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		wg := sync.WaitGroup{}
		for _, param := range taskParams {
			wg.Add(1)
			go func(param interface{}) {
				defer wg.Done()
				handleFunc(ctx, param)
			}(param)
		}
	}
	runtime.ReadMemStats(&m)
	afterAlloc := m.Alloc
	b.Logf("goroutine Memory allocated: %d bytes\n", afterAlloc)
}

func ExampleGo() {
	taskParams := make([]interface{}, 100000)
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, time.Second*30)
	defer cancel()
	err := Go(ctx, 10000, func(ctx context.Context, msg interface{}) error {
		time.Sleep(time.Millisecond * 10)
		return nil
	}, taskParams...)
	if err != nil {
		fmt.Println(err)
	}
}

func ExampleNewPool() {
	ctx := context.Background()
	p := NewPool(ctx, WithSize(5), WithIgnoreErr())
	p.RegisterHandleFunc(func(ctx context.Context, msg interface{}) error {
		time.Sleep(time.Millisecond * 10)
		return nil
	})
	p.Submit(ctx, []interface{}{"liming", "stone", "alice", "mery", "miss", "lucy", "foo", "nvd"}...)
	err := p.Wait()
	if err != nil {
		return
	}
}
