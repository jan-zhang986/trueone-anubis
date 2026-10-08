package service

import (
	"context"
	"fmt"
	"time"

	"trueone-anubis/internal/workflow/engine"
	wfModel "trueone-anubis/internal/workflow/model"
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

	return s.dagEngine.ExecuteGraph(ctx, graph, params)
}
