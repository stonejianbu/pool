package pool

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestDo(t *testing.T) {
	// 假如有一万任务待执行，每个任务耗时0.5s
	// 如果使用循环处理顺序执行，那耗时肯定很长
	// 如果是一个任务启用一个协程处理，有一万个就启用一万个，可能程序会崩溃掉
	// 为了避免程序崩溃，又让程序快点执行完成，我们需要限定开启协程的数量
	taskParams := make([]interface{}, 10000)
	startTime := time.Now()
	ctx := context.Background()
	Do(ctx, 500, func(ctx context.Context, msg interface{}) {
		fmt.Printf("msg:%+v, do something\n", msg)
		// 这里可能执行调用第三方接口或数据库读写操作
		time.Sleep(time.Millisecond * 500)
		return
	}, taskParams...)
	fmt.Printf("耗时：%0.2f\n", time.Since(startTime).Seconds())
}

func TestGo(t *testing.T) {
	taskParams := []interface{}{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20}
	startTime := time.Now()
	ctx := context.Background()
	rets, err := Go(ctx, 3, func(ctx context.Context, msg interface{}) (interface{}, error) {
		fmt.Printf("msg:%+v, do something\n", msg)
		// 这里可能执行调用第三方接口或数据库读写操作
		time.Sleep(time.Second * 1)
		m := msg.(int)
		if m == 10 {
			return nil, fmt.Errorf("msg信息异常")
		}
		return msg, nil
	}, taskParams...)
	if err != nil {
		fmt.Println(err)
	}
	for _, ret := range rets {
		fmt.Printf("ret: %v\n", ret)
	}
	fmt.Printf("耗时：%0.2f\n", time.Since(startTime).Seconds())
}

func TestGo_WithTimeout(t *testing.T) {
	taskParams := []interface{}{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20}
	startTime := time.Now()
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, time.Second*3)
	defer cancel()
	rets, err := Go(ctx, 3, func(ctx context.Context, msg interface{}) (interface{}, error) {
		fmt.Printf("msg:%+v, do something\n", msg)
		// 这里可能执行调用第三方接口或数据库读写操作
		time.Sleep(time.Second * 2)
		return msg, nil
	}, taskParams...)
	if err != nil {
		fmt.Println(err)
	}
	for _, ret := range rets {
		fmt.Printf("ret: %v\n", ret)
	}
	fmt.Printf("耗时：%0.2f\n", time.Since(startTime).Seconds())
}
