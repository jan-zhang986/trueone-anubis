package model

// NodeStatus 节点执行状态
type NodeStatus string

const (
	NodeStatusPending NodeStatus = "PENDING"
	NodeStatusRunning NodeStatus = "RUNNING"
	NodeStatusSuccess NodeStatus = "SUCCESS"
	NodeStatusFailed  NodeStatus = "FAILED"
	NodeStatusSkipped NodeStatus = "SKIPPED"
)

// WorkflowStatus 工作流执行状态
type WorkflowStatus string

const (
	WorkflowStatusPending WorkflowStatus = "PENDING"
	WorkflowStatusRunning WorkflowStatus = "RUNNING"
	WorkflowStatusSuccess WorkflowStatus = "SUCCESS"
	WorkflowStatusFailed  WorkflowStatus = "FAILED"
)

// NodeType 节点类型
type NodeType string

const (
	NodeTypeHTTP       NodeType = "HTTP"        // HTTP 接口调用
	NodeTypeSQL        NodeType = "SQL"         // 数据库核算与状态断言
	NodeTypeScript     NodeType = "SCRIPT"      // 动态数据加工/表达式
	NodeTypeCondition  NodeType = "CONDITION"   // 分支判断
	NodeTypeQualityGate NodeType = "QUALITY_GATE" // 质量门禁与准出裁决
)

// WorkflowNode DAG 节点定义
type WorkflowNode struct {
	ID          string                 `json:"id" yaml:"id"`
	Name        string                 `json:"name" yaml:"name"`
	Type        NodeType               `json:"type" yaml:"type"`
	DependsOn   []string               `json:"dependsOn" yaml:"dependsOn"` // 依赖的前置节点 ID 列表
	Config      map[string]interface{} `json:"config" yaml:"config"`       // 节点执行配置 (URL, SQL, Script 等)
	RetryLimit  int                    `json:"retryLimit" yaml:"retryLimit"`
	TimeoutSec  int                    `json:"timeoutSec" yaml:"timeoutSec"`
}

// WorkflowGraph 完整有向无环图定义
type WorkflowGraph struct {
	ID          string                 `json:"id" yaml:"id"`
	Name        string                 `json:"name" yaml:"name"`
	Description string                 `json:"description" yaml:"description"`
	Priority    string                 `json:"priority" yaml:"priority"`
	Module      string                 `json:"module" yaml:"module"`
	Tags        []string               `json:"tags" yaml:"tags"`
	Params      map[string]interface{} `json:"params" yaml:"params"`
	Nodes       []WorkflowNode         `json:"nodes" yaml:"nodes"`
}

// NodeExecutionResult 单个节点的执行快照与证据
type NodeExecutionResult struct {
	NodeID     string                 `json:"nodeId"`
	NodeName   string                 `json:"nodeName"`
	Status     NodeStatus             `json:"status"`
	DurationMs int64                  `json:"durationMs"`
	Error      string                 `json:"error,omitempty"`
	Output     map[string]interface{} `json:"output,omitempty"` // 产出变量供下游节点消费
	Evidence   map[string]interface{} `json:"evidence,omitempty"` // 捕获的断言/DB快照
}

// WorkflowExecutionResult 整个 DAG 执行报告
type WorkflowExecutionResult struct {
	WorkflowID  string                         `json:"workflowId"`
	ExecutionID string                         `json:"executionId"`
	Status      WorkflowStatus                 `json:"status"`
	TotalNodes  int                            `json:"totalNodes"`
	DurationMs  int64                          `json:"durationMs"`
	NodeResults map[string]NodeExecutionResult `json:"nodeResults"`
	Context     map[string]interface{}         `json:"context"` // 全局上下文池 (共享变量)
	Error       string                         `json:"error,omitempty"`
}
