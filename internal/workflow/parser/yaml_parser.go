package parser

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
	"trueone-anubis/internal/workflow/model"
)

// ParseWorkflowYAML 将 YAML 字节切片反序列化为 WorkflowGraph 对象并校验合法性
func ParseWorkflowYAML(content []byte) (*model.WorkflowGraph, error) {
	var graph model.WorkflowGraph
	if err := yaml.Unmarshal(content, &graph); err != nil {
		return nil, fmt.Errorf("解析 YAML 失败: %w", err)
	}

	if graph.ID == "" {
		if graph.Name != "" {
			graph.ID = "wf-" + strings.ToLower(strings.ReplaceAll(graph.Name, " ", "-"))
		} else {
			return nil, fmt.Errorf("工作流 YAML 必须包含 id 或 name")
		}
	}

	if graph.Priority == "" {
		graph.Priority = "P0"
	}

	// 统一归一化为 variables 字段
	if graph.Variables == nil && graph.Params != nil {
		graph.Variables = graph.Params
	} else if graph.Variables != nil && graph.Params != nil {
		for k, v := range graph.Params {
			if _, exists := graph.Variables[k]; !exists {
				graph.Variables[k] = v
			}
		}
	}

	if len(graph.Nodes) == 0 {
		return nil, fmt.Errorf("工作流必须至少包含一个节点")
	}

	// 校验节点 ID 唯一性与依赖合法性
	nodeIDs := make(map[string]bool)
	for _, n := range graph.Nodes {
		if n.ID == "" {
			return nil, fmt.Errorf("节点必须指定有效 id")
		}
		if nodeIDs[n.ID] {
			return nil, fmt.Errorf("存在重复的节点 id: %s", n.ID)
		}
		nodeIDs[n.ID] = true
	}

	// 校验依赖是否存在
	for _, n := range graph.Nodes {
		for _, dep := range n.DependsOn {
			if !nodeIDs[dep] {
				return nil, fmt.Errorf("节点 %s 依赖了不存在的节点 %s", n.ID, dep)
			}
			if dep == n.ID {
				return nil, fmt.Errorf("节点 %s 不能自我依赖", n.ID)
			}
		}
	}

	return &graph, nil
}
