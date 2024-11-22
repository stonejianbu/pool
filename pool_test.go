package pool

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func handleFunc(ctx context.Context, msg interface{}) error {
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

func TestPool(t *testing.T) {
	ctx := context.Background()
	// 创建并初始化一个任务池
	p := NewPool(ctx, WithSize(5), WithIgnoreErr())
	// 注册默认的处理器函数到任务池
	p.RegisterHandleFunc(handleFunc)
	// 将参数加入到默认的生产者中，触发任务的生成
	p.Submit(ctx, []interface{}{"liming", "stone", "alice", "mery", "miss", "lucy", "foo", "nvd"}...)
	// 等待所有任务完成，并返回结果切片
	err := p.Wait()
	if err != nil {
		t.Errorf("err: %v", err)
		return
	}
}
