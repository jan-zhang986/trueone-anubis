package service

import (
	"context"
	"fmt"
	"time"

	"github.com/vanguard-platform/aegis-sdk-go/workflow"
	wfModel "github.com/vanguard-platform/aegis-sdk-go/workflow/model"
)

type WorkflowService struct{}

func NewWorkflowService() *WorkflowService {
	return &WorkflowService{}
}

// ExecuteGraph 接收并执行工作流 DAG（委托给共享的 trueone-sdk 内核）
func (s *WorkflowService) ExecuteGraph(graph *wfModel.WorkflowGraph, params map[string]interface{}) (*wfModel.WorkflowExecutionResult, error) {
	if graph == nil || len(graph.Nodes) == 0 {
		return nil, fmt.Errorf("workflow graph nodes cannot be empty")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	return workflow.RunGraph(ctx, graph, params)
}

// ExecuteYAML 解析 YAML 并调度执行 DAG（委托给共享的 trueone-sdk 内核）
func (s *WorkflowService) ExecuteYAML(yamlContent string, params map[string]interface{}) (*wfModel.WorkflowExecutionResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	return workflow.RunYAML(ctx, []byte(yamlContent), params)
}
