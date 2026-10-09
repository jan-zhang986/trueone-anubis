package processor

import (
	"context"
)

// ProcessorMetadata 处理器元数据描述（对标 aegis-runner ProcessorMetadata）
type ProcessorMetadata struct {
	Type               string   `json:"type"`               // 处理器唯一类型标识 (如 HTTP, SQL, QUALITY_GATE, DUBBO 等)
	Name               string   `json:"name"`               // 可读名称
	Description        string   `json:"description"`        // 描述信息
	Category           string   `json:"category"`           // 分类: "api", "data", "gate", "remote", "job", "workflow"
	Version            string   `json:"version"`            // 插件版本
	Author             string   `json:"author"`             // 作者
	RequiredConfigKeys []string `json:"requiredConfigKeys"` // 必填配置项键列表
	OptionalConfigKeys []string `json:"optionalConfigKeys"` // 选填配置项键列表
}

// ExecutionContext 节点执行上下文（对标 aegis-runner ExecutionContext）
type ExecutionContext struct {
	NodeID             string                 // 当前执行节点 ID
	NodeName           string                 // 节点名称
	NodeType           string                 // 节点类型
	Config             map[string]interface{} // 已完成变量插值的配置字典
	PredecessorResults map[string]interface{} // 前置上游依赖节点的执行结果
	ScopeGetter        func(key string) (interface{}, bool)
	ScopeSetter        func(key string, val interface{})
}

// ExecutionResult 统一执行结果
type ExecutionResult struct {
	Status     string                 `json:"status"`     // SUCCESS, FAILED, SKIPPED
	DurationMs int64                  `json:"durationMs"` // 执行耗时 (ms)
	Output     map[string]interface{} `json:"output"`     // 输出上下文数据 (用于 extract 注入 downstream)
	Evidence   map[string]interface{} `json:"evidence"`   // 执行证据快照 (用于审计与追溯)
	Error      string                 `json:"error"`      // 异常或错误信息
}

// ProcessorInterface 统一处理器接口（所有 DAG 节点能力插件均需实现此接口）
type ProcessorInterface interface {
	// GetType 返回处理器类型标识
	GetType() string
	// GetMetadata 返回处理器的元数据规范
	GetMetadata() ProcessorMetadata
	// ValidateConfig 校验节点配置合法性
	ValidateConfig(config map[string]interface{}) error
	// Execute 执行处理器核心业务逻辑
	Execute(ctx context.Context, execCtx *ExecutionContext) (*ExecutionResult, error)
}

// BaseProcessor 基础抽象结构体，提供默认实现
type BaseProcessor struct {
	Metadata ProcessorMetadata
}

func (b *BaseProcessor) GetType() string {
	return b.Metadata.Type
}

func (b *BaseProcessor) GetMetadata() ProcessorMetadata {
	return b.Metadata
}

func (b *BaseProcessor) ValidateConfig(config map[string]interface{}) error {
	for _, reqKey := range b.Metadata.RequiredConfigKeys {
		if _, ok := config[reqKey]; !ok {
			return &ValidationError{
				ProcessorType: b.Metadata.Type,
				MissingField:  reqKey,
			}
		}
	}
	return nil
}

// ValidationError 配置校验错误
type ValidationError struct {
	ProcessorType string
	MissingField  string
}

func (e *ValidationError) Error() string {
	return "[" + e.ProcessorType + "] missing required config field: " + e.MissingField
}
