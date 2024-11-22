package pool

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestGo(t *testing.T) {
	taskParams := make([]interface{}, 10000)
	startTime := time.Now()
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()
	err := Go(ctx, 100, func(ctx context.Context, msg interface{}) error {
		fmt.Printf("msg:%+v, do something\n", msg)
		return nil
	}, taskParams...)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Printf("耗时：%0.2f\n", time.Since(startTime).Seconds())
}

func TestPool(t *testing.T) {
	ctx := context.Background()
	// 创建并初始化一个任务池
	p := NewPool(ctx, WithSize(5), WithIgnoreErr())
	// 注册默认的处理器函数到任务池
	p.RegisterHandleFunc(func(ctx context.Context, msg interface{}) error {
		fmt.Printf("msg:%+v, do something\n", msg)
		// 这里可能执行调用第三方接口或数据库读写操作
		time.Sleep(time.Second * 1)
		return nil
	})
	// 将参数加入到默认的生产者中，触发任务的生成
	p.Submit(ctx, []interface{}{"liming", "stone", "alice", "mery", "miss", "lucy", "foo", "nvd"}...)
	// 等待所有任务完成，并返回结果切片
	err := p.Wait()
	if err != nil {
		t.Errorf("err: %v", err)
		return
	}
}
