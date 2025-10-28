package v5_refactored

import (
	"sync"
	"time"
)

// ============================================================================
// 事件实现
// ============================================================================

// BaseEvent 基础事件实现
type BaseEvent struct {
	// eventType 事件类型
	eventType EventType

	// context 验证上下文
	context *ValidationContext

	// timestamp 事件时间戳
	timestamp int64

	// data 事件数据
	data map[string]any
}

// NewBaseEvent 创建基础事件
func NewBaseEvent(eventType EventType, ctx *ValidationContext) *BaseEvent {
	return &BaseEvent{
		eventType: eventType,
		context:   ctx,
		timestamp: time.Now().UnixNano(),
		data:      make(map[string]any),
	}
}

// Type 事件类型
func (e *BaseEvent) Type() EventType {
	return e.eventType
}

// Context 获取验证上下文
func (e *BaseEvent) Context() *ValidationContext {
	return e.context
}

// Timestamp 事件时间戳
func (e *BaseEvent) Timestamp() int64 {
	return e.timestamp
}

// Data 事件数据
func (e *BaseEvent) Data() map[string]any {
	return e.data
}

// WithData 设置事件数据
func (e *BaseEvent) WithData(key string, value any) *BaseEvent {
	e.data[key] = value
	return e
}

// ============================================================================
// 同步事件总线
// ============================================================================

// SyncEventBus 同步事件总线
// 职责：同步发布订阅事件
// 设计原则：观察者模式，同步执行
type SyncEventBus struct {
	// listeners 监听器列表
	listeners []EventListener

	// mu 保护监听器列表
	mu sync.RWMutex
}

// NewSyncEventBus 创建同步事件总线
func NewSyncEventBus() *SyncEventBus {
	return &SyncEventBus{
		listeners: make([]EventListener, 0),
	}
}

// Subscribe 订阅事件
func (bus *SyncEventBus) Subscribe(listener EventListener) {
	if listener == nil {
		return
	}

	bus.mu.Lock()
	defer bus.mu.Unlock()

	bus.listeners = append(bus.listeners, listener)
}

// Unsubscribe 取消订阅
func (bus *SyncEventBus) Unsubscribe(listener EventListener) {
	if listener == nil {
		return
	}

	bus.mu.Lock()
	defer bus.mu.Unlock()

	for i, l := range bus.listeners {
		if l == listener {
			// 删除监听器
			bus.listeners = append(bus.listeners[:i], bus.listeners[i+1:]...)
			break
		}
	}
}

// Publish 发布事件
func (bus *SyncEventBus) Publish(event Event) {
	if event == nil {
		return
	}

	bus.mu.RLock()
	defer bus.mu.RUnlock()

	// 同步调用所有监听器
	for _, listener := range bus.listeners {
		// 检查监听器是否关心这个事件类型
		if bus.isInterestedIn(listener, event.Type()) {
			// 捕获 panic，避免单个监听器错误影响整体
			func() {
				defer func() {
					if r := recover(); r != nil {
						// 记录日志或忽略错误
					}
				}()
				listener.OnEvent(event)
			}()
		}
	}
}

// Clear 清空所有监听器
func (bus *SyncEventBus) Clear() {
	bus.mu.Lock()
	defer bus.mu.Unlock()

	bus.listeners = bus.listeners[:0]
}

// isInterestedIn 检查监听器是否关心此事件类型
func (bus *SyncEventBus) isInterestedIn(listener EventListener, eventType EventType) bool {
	types := listener.EventTypes()
	// 空列表表示关心所有事件
	if len(types) == 0 {
		return true
	}

	for _, t := range types {
		if t == eventType {
			return true
		}
	}
	return false
}

// ============================================================================
// 异步事件总线
// ============================================================================

// AsyncEventBus 异步事件总线
// 职责：异步发布订阅事件
// 设计原则：观察者模式，异步执行，不阻塞主流程
type AsyncEventBus struct {
	// listeners 监听器列表
	listeners []EventListener

	// eventChan 事件通道
	eventChan chan Event

	// workers 工作协程数
	workers int

	// mu 保护监听器列表
	mu sync.RWMutex

	// wg 等待工作协程退出
	wg sync.WaitGroup

	// stopChan 停止信号
	stopChan chan struct{}

	// started 是否已启动
	started bool
}

// NewAsyncEventBus 创建异步事件总线
func NewAsyncEventBus(workers int, bufferSize int) *AsyncEventBus {
	if workers <= 0 {
		workers = 1
	}
	if bufferSize <= 0 {
		bufferSize = 100
	}

	bus := &AsyncEventBus{
		listeners: make([]EventListener, 0),
		eventChan: make(chan Event, bufferSize),
		workers:   workers,
		stopChan:  make(chan struct{}),
		started:   false,
	}

	return bus
}

// Start 启动事件总线
func (bus *AsyncEventBus) Start() {
	bus.mu.Lock()
	defer bus.mu.Unlock()

	if bus.started {
		return
	}

	// 启动工作协程
	for i := 0; i < bus.workers; i++ {
		bus.wg.Add(1)
		go bus.worker()
	}

	bus.started = true
}

// Stop 停止事件总线
func (bus *AsyncEventBus) Stop() {
	bus.mu.Lock()
	defer bus.mu.Unlock()

	if !bus.started {
		return
	}

	close(bus.stopChan)
	bus.wg.Wait()
	bus.started = false
}

// worker 工作协程
func (bus *AsyncEventBus) worker() {
	defer bus.wg.Done()

	for {
		select {
		case event := <-bus.eventChan:
			bus.handleEvent(event)
		case <-bus.stopChan:
			// 处理剩余事件
			for {
				select {
				case event := <-bus.eventChan:
					bus.handleEvent(event)
				default:
					return
				}
			}
		}
	}
}

// handleEvent 处理事件
func (bus *AsyncEventBus) handleEvent(event Event) {
	bus.mu.RLock()
	defer bus.mu.RUnlock()

	for _, listener := range bus.listeners {
		if bus.isInterestedIn(listener, event.Type()) {
			// 捕获 panic
			func() {
				defer func() {
					if r := recover(); r != nil {
						// 记录日志
					}
				}()
				listener.OnEvent(event)
			}()
		}
	}
}

// Subscribe 订阅事件
func (bus *AsyncEventBus) Subscribe(listener EventListener) {
	if listener == nil {
		return
	}

	bus.mu.Lock()
	defer bus.mu.Unlock()

	bus.listeners = append(bus.listeners, listener)
}

// Unsubscribe 取消订阅
func (bus *AsyncEventBus) Unsubscribe(listener EventListener) {
	if listener == nil {
		return
	}

	bus.mu.Lock()
	defer bus.mu.Unlock()

	for i, l := range bus.listeners {
		if l == listener {
			bus.listeners = append(bus.listeners[:i], bus.listeners[i+1:]...)
			break
		}
	}
}

// Publish 发布事件
func (bus *AsyncEventBus) Publish(event Event) {
	if event == nil {
		return
	}

	// 非阻塞发送
	select {
	case bus.eventChan <- event:
		// 成功发送
	default:
		// 通道已满，丢弃事件或记录日志
	}
}

// Clear 清空所有监听器
func (bus *AsyncEventBus) Clear() {
	bus.mu.Lock()
	defer bus.mu.Unlock()

	bus.listeners = bus.listeners[:0]
}

// isInterestedIn 检查监听器是否关心此事件类型
func (bus *AsyncEventBus) isInterestedIn(listener EventListener, eventType EventType) bool {
	types := listener.EventTypes()
	if len(types) == 0 {
		return true
	}

	for _, t := range types {
		if t == eventType {
			return true
		}
	}
	return false
}

// ============================================================================
// 空事件总线（用于禁用事件）
// ============================================================================

// NoOpEventBus 空事件总线
// 职责：什么都不做的事件总线，用于性能优化
type NoOpEventBus struct{}

// NewNoOpEventBus 创建空事件总线
func NewNoOpEventBus() *NoOpEventBus {
	return &NoOpEventBus{}
}

// Subscribe 订阅事件（什么都不做）
func (bus *NoOpEventBus) Subscribe(listener EventListener) {}

// Unsubscribe 取消订阅（什么都不做）
func (bus *NoOpEventBus) Unsubscribe(listener EventListener) {}

// Publish 发布事件（什么都不做）
func (bus *NoOpEventBus) Publish(event Event) {}

// Clear 清空（什么都不做）
func (bus *NoOpEventBus) Clear() {}
