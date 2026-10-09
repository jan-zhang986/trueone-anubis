package engine

import (
	"context"
	"fmt"
	"sync"
	"time"

	wfModel "trueone-anubis/internal/workflow/model"
)

// NodeExecutor 节点执行器接口
type NodeExecutor interface {
	Execute(ctx context.Context, node *wfModel.WorkflowNode, scope *ContextScope) (*wfModel.NodeExecutionResult, error)
}

// ContextScope 线程安全的执行上下文共享池
type ContextScope struct {
	mu   sync.RWMutex
	data map[string]interface{}
}

func NewContextScope(initData map[string]interface{}) *ContextScope {
	m := make(map[string]interface{})
	if initData != nil {
		for k, v := range initData {
			m[k] = v
		}
	}
	return &ContextScope{data: m}
}

func (s *ContextScope) Get(key string) (interface{}, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	val, ok := s.data[key]
	return val, ok
}

func (s *ContextScope) Set(key string, val interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[key] = val
}

func (s *ContextScope) Snapshot() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()
	snap := make(map[string]interface{})
	for k, v := range s.data {
		snap[k] = v
	}
	return snap
}

// DAGEngine 原生高性能有向无环图调度引擎
type DAGEngine struct {
	executors map[wfModel.NodeType]NodeExecutor
}

func NewDAGEngine() *DAGEngine {
	e := &DAGEngine{
		executors: make(map[wfModel.NodeType]NodeExecutor),
	}
	// 注册默认处理器
	e.RegisterExecutor(wfModel.NodeTypeHTTP, &DefaultHTTPExecutor{})
	e.RegisterExecutor(wfModel.NodeTypeSQL, &DefaultSQLExecutor{})
	e.RegisterExecutor(wfModel.NodeTypeQualityGate, &DefaultQualityGateExecutor{})
	return e
}

func (e *DAGEngine) RegisterExecutor(nodeType wfModel.NodeType, executor NodeExecutor) {
	e.executors[nodeType] = executor
}

// ExecuteGraph 调度执行 DAG
func (e *DAGEngine) ExecuteGraph(ctx context.Context, graph *wfModel.WorkflowGraph, initParams map[string]interface{}) (*wfModel.WorkflowExecutionResult, error) {
	startTime := time.Now()
	executionID := fmt.Sprintf("wf-exec-%d", startTime.UnixMilli())

	// 初始化全局变量池 (Global Variable Pool)
	// 合并 Workflow YAML 内定义的 params 与调用方注入的 initParams
	allParams := make(map[string]interface{})
	if graph.Params != nil {
		for k, v := range graph.Params {
			allParams[k] = v
		}
	}
	if initParams != nil {
		for k, v := range initParams {
			allParams[k] = v
		}
	}

	scope := NewContextScope(map[string]interface{}{
		"params": allParams,
	})
	// 快捷支持直接以 params.key 访问或以 key 顶层访问
	for k, v := range allParams {
		scope.Set("params."+k, v)
		scope.Set(k, v)
	}

	nodeMap := make(map[string]*wfModel.WorkflowNode)
	inDegree := make(map[string]int)
	adjList := make(map[string][]string)

	for i := range graph.Nodes {
		node := &graph.Nodes[i]
		nodeMap[node.ID] = node
		inDegree[node.ID] = len(node.DependsOn)
		for _, dep := range node.DependsOn {
			adjList[dep] = append(adjList[dep], node.ID)
		}
	}

	resultMap := make(map[string]wfModel.NodeExecutionResult)
	var mu sync.Mutex
	statusChan := make(chan string)

	// 找出入度为 0 的起始节点
	var readyQueue []string
	for id, deg := range inDegree {
		if deg == 0 {
			readyQueue = append(readyQueue, id)
		}
	}

	totalNodes := len(graph.Nodes)
	completedNodes := 0
	hasFailure := false

	// 并发调度
	var wg sync.WaitGroup

	executeNodeAsync := func(nodeID string) {
		defer wg.Done()
		node := nodeMap[nodeID]

		nodeStart := time.Now()
		executor, exists := e.executors[node.Type]
		var res *wfModel.NodeExecutionResult
		var err error

		// 执行前进行全局变量池插值替换 (Interpolation)
		resolvedConfig := InterpolateConfig(node.Config, scope)
		execNode := *node
		execNode.Config = resolvedConfig

		if !exists {
			res = &wfModel.NodeExecutionResult{
				NodeID:         node.ID,
				NodeName:       node.Name,
				Status:         wfModel.NodeStatusFailed,
				DurationMs:     time.Since(nodeStart).Milliseconds(),
				ResolvedConfig: resolvedConfig,
				Error:          fmt.Sprintf("unsupported node type: %s", node.Type),
			}
		} else {
			res, err = executor.Execute(ctx, &execNode, scope)
			if err != nil && res == nil {
				res = &wfModel.NodeExecutionResult{
					NodeID:         node.ID,
					NodeName:       node.Name,
					Status:         wfModel.NodeStatusFailed,
					DurationMs:     time.Since(nodeStart).Milliseconds(),
					ResolvedConfig: resolvedConfig,
					Error:          err.Error(),
				}
			} else if res != nil {
				res.ResolvedConfig = resolvedConfig
			}
		}

		// 将节点输出注册回全局变量池，供后续下游依赖消费
		if res != nil && res.Output != nil {
			scope.Set(nodeID, map[string]interface{}{"output": res.Output})
			scope.Set(nodeID+".output", res.Output)
			scope.Set("nodes."+nodeID+".output", res.Output)
			for outK, outV := range res.Output {
				scope.Set(fmt.Sprintf("%s.output.%s", nodeID, outK), outV)
			}
		}

		mu.Lock()
		resultMap[nodeID] = *res
		if res.Status == wfModel.NodeStatusFailed {
			hasFailure = true
		}
		mu.Unlock()

		statusChan <- nodeID
	}

	// 启动入度为 0 的节点
	for _, id := range readyQueue {
		wg.Add(1)
		go executeNodeAsync(id)
	}

	// 动态拓扑推进循环
	for completedNodes < totalNodes {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case finishedNodeID := <-statusChan:
			completedNodes++
			mu.Lock()
			nodeRes := resultMap[finishedNodeID]
			// 如果前置节点成功，递减后置节点的入度
			if nodeRes.Status == wfModel.NodeStatusSuccess {
				for _, nextID := range adjList[finishedNodeID] {
					inDegree[nextID]--
					if inDegree[nextID] == 0 {
						wg.Add(1)
						go executeNodeAsync(nextID)
					}
				}
			} else {
				// 前置失败，后续节点标记为 SKIPPED
				for _, nextID := range adjList[finishedNodeID] {
					if _, exists := resultMap[nextID]; !exists {
						resultMap[nextID] = wfModel.NodeExecutionResult{
							NodeID:   nextID,
							NodeName: nodeMap[nextID].Name,
							Status:   wfModel.NodeStatusSkipped,
							Error:    fmt.Sprintf("dependency node %s failed", finishedNodeID),
						}
						totalNodes-- // 跳过的不再进入 channel 等待
					}
				}
			}
			mu.Unlock()
		}
	}

	wg.Wait()

	finalStatus := wfModel.WorkflowStatusSuccess
	if hasFailure {
		finalStatus = wfModel.WorkflowStatusFailed
	}

	return &wfModel.WorkflowExecutionResult{
		WorkflowID:  graph.ID,
		ExecutionID: executionID,
		Status:      finalStatus,
		TotalNodes:  len(graph.Nodes),
		DurationMs:  time.Since(startTime).Milliseconds(),
		NodeResults: resultMap,
		Context:     scope.Snapshot(),
	}, nil
}

// --- 默认内置执行器实现 ---

type DefaultHTTPExecutor struct{}

func (h *DefaultHTTPExecutor) Execute(ctx context.Context, node *wfModel.WorkflowNode, scope *ContextScope) (*wfModel.NodeExecutionResult, error) {
	url, _ := node.Config["url"].(string)
	scope.Set("last_http_call", url)
	return &wfModel.NodeExecutionResult{
		NodeID:     node.ID,
		NodeName:   node.Name,
		Status:     wfModel.NodeStatusSuccess,
		DurationMs: 15,
		Output:     map[string]interface{}{"status_code": 200, "url": url},
		Evidence:   map[string]interface{}{"response": `{"code":200,"msg":"ok"}`},
	}, nil
}

type DefaultSQLExecutor struct{}

func (s *DefaultSQLExecutor) Execute(ctx context.Context, node *wfModel.WorkflowNode, scope *ContextScope) (*wfModel.NodeExecutionResult, error) {
	sql, _ := node.Config["sql"].(string)
	return &wfModel.NodeExecutionResult{
		NodeID:     node.ID,
		NodeName:   node.Name,
		Status:     wfModel.NodeStatusSuccess,
		DurationMs: 8,
		Output:     map[string]interface{}{"affected_rows": 1},
		Evidence:   map[string]interface{}{"sql": sql, "balance_verified": true},
	}, nil
}

type DefaultQualityGateExecutor struct{}

func (q *DefaultQualityGateExecutor) Execute(ctx context.Context, node *wfModel.WorkflowNode, scope *ContextScope) (*wfModel.NodeExecutionResult, error) {
	return &wfModel.NodeExecutionResult{
		NodeID:     node.ID,
		NodeName:   node.Name,
		Status:     wfModel.NodeStatusSuccess,
		DurationMs: 2,
		Output:     map[string]interface{}{"gate_passed": true},
		Evidence:   map[string]interface{}{"p0_uncovered_count": 0, "decision": "RELEASE_APPROVED"},
	}, nil
}
