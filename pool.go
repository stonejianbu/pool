package pool

import (
	"context"
	"fmt"
	"sync"
)

type Pool struct {
	handlerFunc  func(ctx context.Context, msg interface{}) (interface{}, error)
	justDoItFunc func(ctx context.Context, msg interface{})
	msgChan      chan interface{}
	msgWg        sync.WaitGroup
	num          uint
	resultChan   chan interface{}
	resultList   []interface{}
	resultWg     sync.WaitGroup
	err          error
	errMux       sync.RWMutex
	ignoreErr    bool
}

// NewPool 创建一个新的任务处理池。
// 该函数接收一个context和一个表示池中并发工作器数量的uint。
// 返回值是一个指向Pool的指针。
func NewPool(ctx context.Context, num uint) *Pool {
	// 初始化Pool结构体实例。
	instance := &Pool{
		msgChan:    make(chan interface{}, 2*num), // 创建一个固定容量的消息通道。
		msgWg:      sync.WaitGroup{},              // 初始化用于同步消息处理的WaitGroup。
		num:        num,                           // 设置工作器数量。
		resultChan: make(chan interface{}, 2*num), // 创建一个固定容量的结果通道。
		resultList: make([]interface{}, 0),        // 初始化存储结果的切片。
		resultWg:   sync.WaitGroup{},              // 初始化用于同步结果处理的WaitGroup。
		errMux:     sync.RWMutex{},
	}
	// 启动消息消费者协程，负责从消息通道中消费消息并进行处理。
	instance.consumer(ctx)
	// 启动结果获取协程，负责从结果通道中获取并处理结果。
	instance.getResult()
	// 返回池实例。
	return instance
}

// Producer 向消息池中发送消息。
// 该方法接收一个可变数量的Msg参数，并将它们逐个发送到消息通道中。
// 参数：
//   msgs ...interface{}: 要发送到消息池中的消息列表。
// 注意：
//   1. 该方法通过循环遍历传入的消息列表，并使用channel发送操作（<-）将每个消息发送到消息通道中。
//   2. 调用该方法前，需要确保消息池已经被正确初始化并且消息通道（msgChan）是可写入的。
func (that *Pool) Producer(msgs ...interface{}) {
	// 遍历传入的所有消息，并将它们逐个发送到消息通道中。
	for _, msg := range msgs {
		if that.getError() != nil {
			return
		}
		that.msgChan <- msg
	}
}

// Wait 等待所有消息处理完成并返回结果列表。
// 该方法首先关闭消息通道（msgChan），确保所有未处理的消息被处理。
// 然后，它等待与消息处理相关的等待群（msgWg）完成。
// 接着，关闭结果通道（resultChan），确保所有结果都被取出。
// 最后，等待与结果处理相关的等待群（resultWg）完成。
// 返回值是解析后的响应消息列表。
func (that *Pool) Wait() ([]interface{}, error) {
	// 关闭消息通道，确保所有消息发送完毕
	close(that.msgChan)
	// 等待所有消息被处理完成
	that.msgWg.Wait()
	// 所有消息被处理完成时，也同时表明所有结果已进入保存结果的队列，关闭结果通道
	close(that.resultChan)
	// 等待所有结果处理完成
	that.resultWg.Wait()
	// 返回解析后的结果列表
	return that.resultList, that.err
}

// getResult 从 resultChan 中接收结果消息，并将其添加到 resultList 中
// 本函数通过 resultWg 进行协程同步，确保所有结果都被正确处理
func (that *Pool) getResult() {
	// resultWg.Add 表示有一个新的协程将参与同步
	that.resultWg.Add(1)
	// 启动一个新的匿名协程来处理结果接收逻辑
	go func() {
		// 在协程结束时，调用 resultWg.Done 表示此协程的处理完毕
		defer that.resultWg.Done()
		// 不断从 resultChan 中接收消息，直到 resultChan 关闭或无新消息
		for ret := range that.resultChan {
			// 将接收到的消息追加到 resultList 中
			that.resultList = append(that.resultList, ret)
		}
	}()
}

// consumer 是 Pool 类型的消费者方法，负责处理消息队列中的消息。
// 它接受一个 context 参数，用于取消操作和传递额外的请求信息。
// 该方法在 Pool 类型的实例上运行。
func (that *Pool) consumer(ctx context.Context) {
	for i := 0; i < int(that.num); i++ {
		that.msgWg.Add(1)
		go func() {
			defer that.msgWg.Done()
			defer func() {
				if err := recover(); err != nil {
					that.setError(fmt.Errorf("panic recover err: %v", err))
					return
				}
			}()
			for msg := range that.msgChan {
				select {
				case <-ctx.Done(): // 超时判断
					// 超时设置处理异常，其它协程捕获到异常时会退出
					that.setError(ctx.Err())
					return
				default:
					if that.ignoreErr {
						// 只是执行处理（除panic异常外，不捕获异常）
						that.justDoItFunc(ctx, msg)
					} else {
						// 如果捕获到处理异常则退出
						if that.getError() != nil {
							return
						}
						// 执行处理逻辑
						resp, err := that.handlerFunc(ctx, msg)
						if err != nil {
							that.setError(err)
							return
						}
						that.resultChan <- resp
					}
				}

			}
		}()
	}
}

// getError 获取处理错误
func (that *Pool) getError() error {
	that.errMux.RLock()
	defer that.errMux.RUnlock()
	return that.err
}

// setError 赋值处理错误
func (that *Pool) setError(err error) {
	that.errMux.Lock()
	defer that.errMux.Unlock()
	that.err = err
	return
}

// Do 执行特定任务的函数，使用任务池进行管理并行处理
// 该函数接收一个上下文context、一个无符号整数、一个处理器函数和一组可变参数
// 当处理过程出现异常会继续执行处理，直至任务处理完成
func Do(ctx context.Context, num uint, handler func(context.Context, interface{}), params ...interface{}) {
	// 创建并初始化一个任务池
	p := NewPool(ctx, num)
	// 忽视处理错误，不返回结果列表，just do it
	p.ignoreErr = true
	// 注册默认的处理器函数到任务池
	p.justDoItFunc = handler
	// 将参数加入到默认的生产者中，触发任务的生成
	p.Producer(params...)
	// 关闭消息通道，确保所有消息发送完毕
	close(p.msgChan)
	// 等待所有消息被处理完成
	p.msgWg.Wait()
}

// Go 执行特定任务的函数，使用任务池进行管理并行处理
// 该函数接收一个上下文context、一个无符号整数、一个处理器函数和一组可变参数
// 当处理过程出现异常则停止处理并返回错误
// 返回一个响应消息切片和错误信息
func Go(ctx context.Context, num uint, handler func(context.Context, interface{}) (interface{}, error), params ...interface{}) ([]interface{}, error) {
	// 创建并初始化一个任务池
	p := NewPool(ctx, num)
	// 注册默认的处理器函数到任务池
	p.handlerFunc = handler
	// 将参数加入到默认的生产者中，触发任务的生成
	p.Producer(params...)
	// 等待所有任务完成，并返回结果切片
	return p.Wait()
}
