# pool
goroutine pool

## Introduction

`pool` implements a goroutine pool with fixed capacity, allowing developers to limit the number of goroutines in your concurrent programs.

## Features:

- Simple to new a pool goroutines in your concurrent programs.

## Quick start

### Get it

```shell
go get github.com/stonejianbu/pool@latest
```

### Use it

```go
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/stonejianbu/pool"
)

func main() {
	taskParams := make([]interface{}, 100000)
	taskHandleFunc := func(ctx context.Context, msg interface{}) error {
		time.Sleep(time.Millisecond * 10)
		return nil
	}
	var poolSize uint = 10000
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, time.Second*30)
	defer cancel()
	err := pool.Go(ctx, poolSize, taskHandleFunc, taskParams...)
	if err != nil {
		fmt.Println(err)
	}
}

```

## 📄 License

The source code in `pool` is available under the [MIT License](/LICENSE).