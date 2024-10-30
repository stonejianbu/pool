package pool

import (
	"context"
	"fmt"
	"testing"
)

func TestNewPool(t *testing.T) {
	ctx := context.Background()
	p := NewPool(ctx, 5)
	handler := func(ctx context.Context, msg Msg) *RespMsg {
		fmt.Printf("msg: %+v\n", msg)
		return &RespMsg{
			Content: msg.Content,
			Type:    msg.Type,
		}
	}
	p.RegisterHandlerFunc(DefaultMsgType, handler)
	p.RegisterHandlerFunc(AMsgType, handler)
	p.RegisterHandlerFunc(BMsgType, handler)
	msgList := NewMsgList(DefaultMsgType, []interface{}{"hello", "world", "pool"})
	msgList = append(msgList, NewMsgList(AMsgType, []interface{}{1, 2, 3, 4, 5})...)
	msgList = append(msgList, NewMsgList(BMsgType, []interface{}{"1", "2", "3", "4", "5"})...)
	p.Producer(msgList...)
	results := p.Wait()
	for _, result := range results {
		fmt.Printf("result: %+v\n", result)
	}
	if len(results) != len(msgList) {
		t.Errorf("results length is not equal to params length")
	}
}

func TestNewPool_Default(t *testing.T) {
	ctx := context.Background()
	p := NewPool(ctx, 10)
	p.RegisterDefaultHandlerFunc(func(ctx context.Context, msg Msg) *RespMsg {
		fmt.Printf("default msg: %+v\n", msg)
		return &RespMsg{
			Content: msg.Content,
			Type:    msg.Type,
		}
	})
	params := []interface{}{"hello", "world"}
	p.DefaultProducer(params...)
	results := p.Wait()
	for _, result := range results {
		fmt.Printf("result: %+v\n", result)
	}
	if len(results) != len(params) {
		t.Errorf("results length is not equal to params length")
	}

}
