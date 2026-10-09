package engine

import (
	"context"
	"fmt"
	"sync"
	"time"

	wfModel "trueone-anubis/internal/workflow/model"
	"trueone-anubis/internal/workflow/processor"
	_ "trueone-anubis/internal/workflow/processor/api"
	_ "trueone-anubis/internal/workflow/processor/data"
	_ "trueone-anubis/internal/workflow/processor/gate"
	"trueone-anubis/internal/workflow/processor/remote"
)

// NodeExecutor 节点执行器接口（保留以兼容历史代码）
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

// DAGEngine 插件化、高性能有向无环图调度引擎（微内核架构）
type DAGEngine struct {
	registry *processor.ProcessorRegistry
}

// NewDAGEngine 创建并初始化 DAG 引擎，默认绑定全局插件注册中心
func NewDAGEngine() *DAGEngine {
	reg := processor.GetRegistry()
	// 设置远端 Runner 代理处理器作为默认兜底（对齐 aegis-runner 插件扩展能力）
	reg.SetFallback(remote.NewRunnerProxyProcessor("http://127.0.0.1:8000"))
	return &DAGEngine{
		registry: reg,
	}
}

// NewDAGEngineWithRegistry 允许注入自定义注册中心
func NewDAGEngineWithRegistry(reg *processor.ProcessorRegistry) *DAGEngine {
	return &DAGEngine{
		registry: reg,
	}
}

// RegisterProcessor 注册新节点处理器插件（对齐 Runner 动态插件注册）
func (e *DAGEngine) RegisterProcessor(p processor.ProcessorInterface) {
	e.registry.Register(p)
}

// RegisterExecutor 兼容旧版适配器注册方法
func (e *DAGEngine) RegisterExecutor(nodeType wfModel.NodeType, executor NodeExecutor) {
	e.registry.Register(&legacyExecutorAdapter{nodeType: string(nodeType), executor: executor})
}

// legacyExecutorAdapter 旧版 NodeExecutor 兼容适配器
type legacyExecutorAdapter struct {
	processor.BaseProcessor
	nodeType string
	executor NodeExecutor
}

func (l *legacyExecutorAdapter) GetType() string {
	return l.nodeType
}

func (l *legacyExecutorAdapter) Execute(ctx context.Context, execCtx *processor.ExecutionContext) (*processor.ExecutionResult, error) {
	node := &wfModel.WorkflowNode{
		ID:     execCtx.NodeID,
		Name:   execCtx.NodeName,
		Type:   wfModel.NodeType(execCtx.NodeType),
		Config: execCtx.Config,
	}
	scope := NewContextScope(nil)
	res, err := l.executor.Execute(ctx, node, scope)
	if err != nil {
		return nil, err
	}
	return &processor.ExecutionResult{
		Status:     string(res.Status),
		DurationMs: res.DurationMs,
		Output:     res.Output,
		Evidence:   res.Evidence,
		Error:      res.Error,
	}, nil
}

// ExecuteGraph 调度执行 DAG
func (e *DAGEngine) ExecuteGraph(ctx context.Context, graph *wfModel.WorkflowGraph, initParams map[string]interface{}) (*wfModel.WorkflowExecutionResult, error) {
	startTime := time.Now()
	executionID := fmt.Sprintf("wf-exec-%d", startTime.UnixMilli())

	// 初始化全局变量池 (Global Variable Pool)
	// 合并 Workflow YAML 内定义的 variables (兼容 params) 与调用方注入的入参
	allVars := make(map[string]interface{})
	if graph.Variables != nil {
		for k, v := range graph.Variables {
			allVars[k] = v
		}
	}
	if graph.Params != nil {
		for k, v := range graph.Params {
			if _, exists := allVars[k]; !exists {
				allVars[k] = v
			}
		}
	}
	if initParams != nil {
		for k, v := range initParams {
			allVars[k] = v
		}
	}

	scope := NewContextScope(map[string]interface{}{
		"variables": allVars,
		"vars":      allVars,
		"params":    allVars, // 向下兼容
	})
	// 支持以 variables.key / vars.key / key 原生访问
	for k, v := range allVars {
		scope.Set("variables."+k, v)
		scope.Set("vars."+k, v)
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
		var res *wfModel.NodeExecutionResult

		// 执行前进行全局变量池插值替换 (Interpolation)
		resolvedConfig := InterpolateConfig(node.Config, scope)

		// 从注册中心动态获取匹配的处理器插件 (对标 aegis-runner ProcessorRegistry.get_processor)
		proc, exists := e.registry.Get(string(node.Type))
		if !exists {
			res = &wfModel.NodeExecutionResult{
				NodeID:         node.ID,
				NodeName:       node.Name,
				Status:         wfModel.NodeStatusFailed,
				DurationMs:     time.Since(nodeStart).Milliseconds(),
				ResolvedConfig: resolvedConfig,
				Error:          fmt.Sprintf("no processor plugin registered for type: %s", node.Type),
			}
		} else {
			// 配置有效性静态校验 (对标 validate_config)
			if valErr := proc.ValidateConfig(resolvedConfig); valErr != nil {
				res = &wfModel.NodeExecutionResult{
					NodeID:         node.ID,
					NodeName:       node.Name,
					Status:         wfModel.NodeStatusFailed,
					DurationMs:     time.Since(nodeStart).Milliseconds(),
					ResolvedConfig: resolvedConfig,
					Error:          valErr.Error(),
				}
			} else {
				// 构建标准化执行上下文
				execCtx := &processor.ExecutionContext{
					NodeID:             node.ID,
					NodeName:           node.Name,
					NodeType:           string(node.Type),
					Config:             resolvedConfig,
					PredecessorResults: make(map[string]interface{}),
					ScopeGetter:        scope.Get,
					ScopeSetter:        scope.Set,
				}
				for _, depID := range node.DependsOn {
					if depRes, ok := resultMap[depID]; ok {
						execCtx.PredecessorResults[depID] = depRes.Output
					}
				}

				procRes, err := proc.Execute(ctx, execCtx)
				if err != nil && procRes == nil {
					res = &wfModel.NodeExecutionResult{
						NodeID:         node.ID,
						NodeName:       node.Name,
						Status:         wfModel.NodeStatusFailed,
						DurationMs:     time.Since(nodeStart).Milliseconds(),
						ResolvedConfig: resolvedConfig,
						Error:          err.Error(),
					}
				} else if procRes != nil {
					nodeStatus := wfModel.NodeStatusSuccess
					if procRes.Status == "FAILED" {
						nodeStatus = wfModel.NodeStatusFailed
					}
					res = &wfModel.NodeExecutionResult{
						NodeID:         node.ID,
						NodeName:       node.Name,
						Status:         nodeStatus,
						DurationMs:     procRes.DurationMs,
						ResolvedConfig: resolvedConfig,
						Output:         procRes.Output,
						Evidence:       procRes.Evidence,
						Error:          procRes.Error,
					}
				}
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
