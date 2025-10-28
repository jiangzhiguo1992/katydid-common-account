package orchestrator

import (
	"sync"

	"katydid-common-account/pkg/validator/v6/core"
)

// EventDispatcherImpl 事件分发器实现
// 职责：分发验证事件给监听器
// 设计模式：观察者模式
type EventDispatcherImpl struct {
	listeners []core.ValidationListener
	mu        sync.RWMutex
}

// NewEventDispatcher 创建事件分发器
func NewEventDispatcher() core.EventDispatcher {
	return &EventDispatcherImpl{
		listeners: make([]core.ValidationListener, 0),
	}
}

// Dispatch 分发事件
func (d *EventDispatcherImpl) Dispatch(event core.ValidationEvent) {
	if event == nil {
		return
	}

	d.mu.RLock()
	defer d.mu.RUnlock()

	// 通知所有监听器
	for _, listener := range d.listeners {
		// 使用 goroutine 异步通知，避免阻塞
		// 注意：这里为了简单起见同步通知，实际可以改为异步
		listener.OnEvent(event)
	}
}

// Subscribe 订阅事件
func (d *EventDispatcherImpl) Subscribe(listener core.ValidationListener) {
	if listener == nil {
		return
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	d.listeners = append(d.listeners, listener)
}

// Unsubscribe 取消订阅
func (d *EventDispatcherImpl) Unsubscribe(listener core.ValidationListener) {
	if listener == nil {
		return
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	// 查找并移除
	for i, l := range d.listeners {
		// 注意：这里使用指针比较，可能需要更精确的比较方式
		if l == listener {
			d.listeners = append(d.listeners[:i], d.listeners[i+1:]...)
			break
		}
	}
}
