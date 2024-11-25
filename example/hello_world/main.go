package main

import (
	"context"
	"fmt"
	"time"

	"github.com/stonejianbu/pool"
)

func main() {
	taskParams := []interface{}{"liming", "stone", "alice", "mery", "miss", "lucy", "foo", "nvd"}
	taskHandleFunc := func(ctx context.Context, msg interface{}) error {
		fmt.Printf("do %v\n", msg)
		time.Sleep(time.Millisecond * 10)
		return nil
	}
	var poolSize uint = 10
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, time.Second*30)
	defer cancel()
	err := pool.Go(ctx, poolSize, taskHandleFunc, taskParams...)
	if err != nil {
		fmt.Println(err)
	}
}
