// Copyright (c) 2026 DYCloud J.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to
// use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
// the Software, and to permit persons to whom the Software is furnished to do so,
// subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
// FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
// COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
// IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
// CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

package asset

import (
	"fmt"
	"reflect"
	"time"

	pb "github.com/ydcloud-dy/opshub/pkg/agentproto"
)

// AgentHubAdapter 适配器，将真实的 AgentHub 适配为接口
type AgentHubAdapter struct {
	realHub interface{}
}

// NewAgentHubAdapter 创建适配器
func NewAgentHubAdapter(realHub interface{}) AgentHubInterfaceV2 {
	return &AgentHubAdapter{
		realHub: realHub,
	}
}

func (a *AgentHubAdapter) IsOnline(hostID uint) bool {
	// 使用反射调用 IsOnline 方法
	type onlineChecker interface {
		IsOnline(uint) bool
	}
	return a.realHub.(onlineChecker).IsOnline(hostID)
}

func (a *AgentHubAdapter) GetByHostID(hostID uint) (AgentStreamInterface, bool) {
	// 使用反射调用 GetByHostID 方法
	// AgentHub.GetByHostID 返回 (*AgentStream, bool)
	type hostGetter interface {
		GetByHostID(uint) (*struct{}, bool)
	}

	// 使用 interface{} 来接收任意类型的返回值
	type anyHostGetter interface {
		GetByHostID(uint) (interface{}, bool)
	}

	// 尝试直接调用方法（使用反射）
	// 由于 Go 的类型系统限制，我们需要使用更灵活的方式
	result := reflect.ValueOf(a.realHub).MethodByName("GetByHostID").Call([]reflect.Value{
		reflect.ValueOf(hostID),
	})

	if len(result) != 2 {
		return nil, false
	}

	ok := result[1].Bool()
	if !ok {
		return nil, false
	}

	as := result[0].Interface()
	return &AgentStreamAdapter{realStream: as}, true
}

func (a *AgentHubAdapter) WaitResponse(as AgentStreamInterface, requestID string, timeout time.Duration) (interface{}, error) {
	// 使用反射调用 WaitResponse 方法
	adapter := as.(*AgentStreamAdapter)

	result := reflect.ValueOf(a.realHub).MethodByName("WaitResponse").Call([]reflect.Value{
		reflect.ValueOf(adapter.realStream),
		reflect.ValueOf(requestID),
		reflect.ValueOf(timeout),
	})

	if len(result) != 2 {
		return nil, fmt.Errorf("WaitResponse 返回值数量错误")
	}

	if !result[1].IsNil() {
		return nil, result[1].Interface().(error)
	}

	return result[0].Interface(), nil
}

// AgentStreamAdapter 适配器，将真实的 AgentStream 适配为接口
type AgentStreamAdapter struct {
	realStream interface{}
}

func (a *AgentStreamAdapter) Send(msg *pb.ServerMessage) error {
	// 使用反射调用 Send 方法
	result := reflect.ValueOf(a.realStream).MethodByName("Send").Call([]reflect.Value{
		reflect.ValueOf(msg),
	})

	if len(result) != 1 {
		return fmt.Errorf("Send 返回值数量错误")
	}

	if result[0].IsNil() {
		return nil
	}

	return result[0].Interface().(error)
}
