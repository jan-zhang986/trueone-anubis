package service

import (
	"context"
	"fmt"
	"time"

	"trueone-anubis/internal/workflow/engine"
	wfModel "trueone-anubis/internal/workflow/model"
	"trueone-anubis/internal/workflow/parser"
)

type WorkflowService struct {
	dagEngine *engine.DAGEngine
}

func NewWorkflowService() *WorkflowService {
	return &WorkflowService{
		dagEngine: engine.NewDAGEngine(),
	}
}

// ExecuteGraph 接收并执行工作流 DAG
func (s *WorkflowService) ExecuteGraph(graph *wfModel.WorkflowGraph, params map[string]interface{}) (*wfModel.WorkflowExecutionResult, error) {
	if graph == nil || len(graph.Nodes) == 0 {
		return nil, fmt.Errorf("workflow graph nodes cannot be empty")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	combinedParams := make(map[string]interface{})
	if graph.Params != nil {
		for k, v := range graph.Params {
			combinedParams[k] = v
		}
	}
	if params != nil {
		for k, v := range params {
			combinedParams[k] = v
		}
	}

	return s.dagEngine.ExecuteGraph(ctx, graph, combinedParams)
}

// ExecuteYAML 解析 YAML 并调度执行 DAG
func (s *WorkflowService) ExecuteYAML(yamlContent string, params map[string]interface{}) (*wfModel.WorkflowExecutionResult, error) {
	graph, err := parser.ParseWorkflowYAML([]byte(yamlContent))
	if err != nil {
		return nil, fmt.Errorf("解析 YAML 工作流错误: %w", err)
	}
	return s.ExecuteGraph(graph, params)
}
