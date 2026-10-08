package parser

import (
	"testing"
)

func TestParseWorkflowYAML(t *testing.T) {
	yamlData := []byte(`
id: wf-test-checkout
name: "端到端秒杀下单与对账"
priority: "P0"
params:
  userId: "U1001"
nodes:
  - id: step-1
    name: "创建订单"
    type: HTTP
    config:
      url: "/api/order/create"
  - id: step-2
    name: "校验库存与流水"
    type: SQL
    dependsOn: ["step-1"]
    config:
      sql: "SELECT balance FROM trade_ledger WHERE id=1"
`)

	graph, err := ParseWorkflowYAML(yamlData)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if graph.ID != "wf-test-checkout" {
		t.Errorf("expected ID wf-test-checkout, got %s", graph.ID)
	}
	if len(graph.Nodes) != 2 {
		t.Errorf("expected 2 nodes, got %d", len(graph.Nodes))
	}
	if len(graph.Nodes[1].DependsOn) != 1 || graph.Nodes[1].DependsOn[0] != "step-1" {
		t.Errorf("expected step-2 dependsOn step-1")
	}
}
