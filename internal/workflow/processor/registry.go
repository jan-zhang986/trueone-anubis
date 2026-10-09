package processor

import (
	"fmt"
	"strings"
	"sync"
)

// ProcessorRegistry 统一处理器注册中心（对标 aegis-runner ProcessorRegistry）
type ProcessorRegistry struct {
	mu         sync.RWMutex
	processors map[string]ProcessorInterface
	fallback   ProcessorInterface
}

var (
	defaultRegistry *ProcessorRegistry
	once            sync.Once
)

// GetRegistry 获取全局单例注册中心
func GetRegistry() *ProcessorRegistry {
	once.Do(func() {
		defaultRegistry = NewProcessorRegistry()
	})
	return defaultRegistry
}

// NewProcessorRegistry 创建一个隔离的注册中心实例
func NewProcessorRegistry() *ProcessorRegistry {
	return &ProcessorRegistry{
		processors: make(map[string]ProcessorInterface),
	}
}

// Register 注册处理器
func (r *ProcessorRegistry) Register(p ProcessorInterface) {
	if p == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	normType := strings.ToUpper(strings.TrimSpace(p.GetType()))
	r.processors[normType] = p
}

// Get 获取指定类型的处理器（若本地未注册且配置了 fallback 则返回 fallback）
func (r *ProcessorRegistry) Get(procType string) (ProcessorInterface, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	normType := strings.ToUpper(strings.TrimSpace(procType))
	p, ok := r.processors[normType]
	if ok {
		return p, true
	}
	if r.fallback != nil {
		return r.fallback, true
	}
	return nil, false
}

// SetFallback 设置兜底处理器（例如用于向 aegis-runner 远程派发的代理执行器）
func (r *ProcessorRegistry) SetFallback(p ProcessorInterface) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.fallback = p
}

// List 列出所有已注册处理器的元数据列表
func (r *ProcessorRegistry) List() []ProcessorMetadata {
	r.mu.RLock()
	defer r.mu.RUnlock()

	list := make([]ProcessorMetadata, 0, len(r.processors))
	for _, p := range r.processors {
		list = append(list, p.GetMetadata())
	}
	return list
}

// RegisterProcessor 全局便捷注册函数
func RegisterProcessor(p ProcessorInterface) {
	GetRegistry().Register(p)
}

// GetProcessor 全局便捷查询函数
func GetProcessor(procType string) (ProcessorInterface, bool) {
	return GetRegistry().Get(procType)
}

// ErrProcessorNotFound 处理器未找到错误
func ErrProcessorNotFound(procType string) error {
	return fmt.Errorf("processor not registered for type: %s", procType)
}
