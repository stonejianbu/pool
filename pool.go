package pool

import (
	"context"
	"fmt"
	"sync"
)

type MsgType int

const (
	DefaultMsgType MsgType = iota
	AMsgType
	BMsgType
	CMsgType
	DMsgType
	EMsgType
	FMsgType
	GMsgType
)

type Msg struct {
	Type    MsgType
	Content interface{}
}

type RespMsg struct {
	Type    MsgType
	Error   error
	Content interface{}
}

func NewMsgList(msgType MsgType, items []interface{}) []Msg {
	msgList := make([]Msg, 0, len(items))
	for _, item := range items {
		msgList = append(msgList, Msg{Type: msgType, Content: item})
	}
	return msgList
}

type Pool struct {
	handlerFuncMap map[MsgType]func(context.Context, Msg) *RespMsg
	msgChan        chan Msg
	msgWg          sync.WaitGroup
	num            uint
	resultChan     chan *RespMsg
	resultList     []*RespMsg
	mux            sync.Mutex
	resultWg       sync.WaitGroup
}

// NewPool 创建一个新的任务处理池。
// 该函数接收一个context和一个表示池中并发工作器数量的uint。
// 返回值是一个指向Pool的指针。
func NewPool(ctx context.Context, num uint) *Pool {
	// 初始化Pool结构体实例。
	instance := &Pool{
		msgChan:        make(chan Msg, 2*num),                             // 创建一个固定容量的消息通道。
		msgWg:          sync.WaitGroup{},                                  // 初始化用于同步消息处理的WaitGroup。
		num:            num,                                               // 设置工作器数量。
		handlerFuncMap: map[MsgType]func(context.Context, Msg) *RespMsg{}, // 初始化消息处理函数映射。
		resultChan:     make(chan *RespMsg, 2*num),                        // 创建一个固定容量的结果通道。
		resultList:     make([]*RespMsg, 0),                               // 初始化存储结果的切片。
		resultWg:       sync.WaitGroup{},                                  // 初始化用于同步结果处理的WaitGroup。
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
//   msgs ...Msg: 要发送到消息池中的消息列表。
// 注意：
//   1. 该方法通过循环遍历传入的消息列表，并使用channel发送操作（<-）将每个消息发送到消息通道中。
//   2. 调用该方法前，需要确保消息池已经被正确初始化并且消息通道（msgChan）是可写入的。
func (that *Pool) Producer(msgs ...Msg) {
	// 遍历传入的所有消息，并将它们逐个发送到消息通道中。
	for _, msg := range msgs {
		that.msgChan <- msg
	}
}

// Wait 等待所有消息处理完成并返回结果列表。
// 该方法首先关闭消息通道（msgChan），确保所有未处理的消息被处理。
// 然后，它等待与消息处理相关的等待群（msgWg）完成。
// 接着，关闭结果通道（resultChan），确保所有结果都被取出。
// 最后，等待与结果处理相关的等待群（resultWg）完成。
// 返回值是解析后的响应消息列表。
func (that *Pool) Wait() []*RespMsg {
	// 关闭消息通道，确保所有消息发送完毕
	close(that.msgChan)
	// 等待所有消息被处理
	that.msgWg.Wait()
	// 关闭结果通道，确保所有结果都被接收
	close(that.resultChan)
	// 等待所有结果处理完成
	that.resultWg.Wait()
	// 返回解析后的结果列表
	return that.resultList
}

// RegisterHandlerFunc 为特定消息类型注册处理函数。
//
// msgType: 指定需要注册处理函数的消息类型。
// handler: 处理函数，接收一个context和一个消息，返回一个响应消息。
//
// 该方法允许消息池为特定的消息类型关联一个处理函数，当接收到已注册消息类型的消息时，
// 消息池会调用对应的处理函数来处理消息。
func (that *Pool) RegisterHandlerFunc(msgType MsgType, handler func(context.Context, Msg) *RespMsg) {
	that.handlerFuncMap[msgType] = handler
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
			for msg := range that.msgChan {
				if handler := that.handlerFuncMap[msg.Type]; handler != nil {
					that.resultChan <- handler(ctx, msg)
				} else {
					panic(fmt.Sprintf("unknown msgType: %d", msg.Type))
				}
			}
		}()
	}
}

// DefaultProducer 是一个方法，用于创建并发送默认类型的MsgList。
// 该方法属于Pool结构。
// 参数 params 用于初始化MsgList的内容。
// 该方法通过调用Pool的Producer方法，将创建的MsgList作为参数传递。
//
// 该方法的主要目的是简化消息生产的过程，
// 通过使用默认的消息类型和提供的参数来创建消息列表（MsgList），
// 并触发这些消息的处理。
func (that *Pool) DefaultProducer(params ...interface{}) {
	that.Producer(NewMsgList(DefaultMsgType, params)...)
}

// RegisterDefaultHandlerFunc registers a default message handler.
//
// This function allows the user to specify a handler for default messages, enhancing the flexibility and usability of the message processing system.
// Parameters:
// - handler: A function that processes the default message. It accepts a context and a message object, and returns a response message object.
//
// Note: This method does not return any value, but modifies the state of the current Pool instance by registering the handler function under the default message type.
func (that *Pool) RegisterDefaultHandlerFunc(handler func(context.Context, Msg) *RespMsg) {
	that.RegisterHandlerFunc(DefaultMsgType, handler)
}

// Do 执行特定任务的函数，使用任务池进行管理并行处理
// 该函数接收一个上下文、一个无符号整数、一个处理器函数和一组可变参数
// 返回一个响应消息切片
func Do(ctx context.Context, num uint, handler func(context.Context, Msg) *RespMsg, params ...interface{}) []*RespMsg {
	// 创建并初始化一个任务池
	p := NewPool(ctx, num)
	// 注册默认的处理器函数到任务池
	p.RegisterDefaultHandlerFunc(handler)
	// 将参数加入到默认的生产者中，触发任务的生成
	p.DefaultProducer(params...)
	// 等待所有任务完成，并返回结果切片
	return p.Wait()
}

// FilterResults 根据指定消息类型筛选响应消息列表。
// 该函数遍历响应消息列表，寻找与指定消息类型相匹配的消息，并收集这些消息的内容。
// 如果某条匹配的消息带有错误信息，则立即返回错误并附带之前收集的所有内容。
// 参数:
//   t - 要筛选的消息类型。
//   respMsgList - 响应消息列表，包含不同类型的消息。
// 返回值:
//   []interface{} - 收集到的消息内容，具体类型根据实际情况而定。
//   error - 如果有消息携带错误信息，则返回该错误，否则返回nil。
func FilterResults(t MsgType, respMsgList []*RespMsg) ([]interface{}, error) {
	// 初始化收集到的消息内容列表。
	rets := make([]interface{}, 0)
	// 初始化错误变量，用于记录遇到的第一个错误。
	var err error
	// 遍历响应消息列表。
	for _, msg := range respMsgList {
		// 检查消息类型是否与目标类型匹配。
		if msg.Type == t {
			// 检查消息是否带有错误信息，如果有，则记录错误并提前返回。
			if msg.Error != nil {
				err = msg.Error
				return rets, err
			}
			// 将匹配消息的内容添加到结果列表中。
			rets = append(rets, msg.Content)
		}
	}
	// 如果没有遇到错误，返回收集到的消息内容列表和nil错误。
	return rets, nil
}
