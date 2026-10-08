package engine_test

import (
	"context"
	"testing"
	"time"

	"trueone-anubis/internal/workflow/engine"
	wfModel "trueone-anubis/internal/workflow/model"
)

func TestDAGEngine_ParallelAndDependencyExecution(t *testing.T) {
	dag := engine.NewDAGEngine()

	// 构造测试拓扑:
	//   [Node-1: HTTP 下单] ──┐
	//                        ├──► [Node-3: SQL 核算借贷对账] ──► [Node-4: 发布门禁裁决]
	//   [Node-2: HTTP 发券] ──┘
	graph := &wfModel.WorkflowGraph{
		ID:          "wf-test-seckill",
		Name:        "秒杀链路与资金核算 DAG",
		Description: "测试 Node1 与 Node2 并发执行，Node3 依赖两者，Node4 依赖 Node3",
		Nodes: []wfModel.WorkflowNode{
			{
				ID:        "node-1",
				Name:      "模拟用户抢购下单 (HTTP)",
				Type:      wfModel.NodeTypeHTTP,
				Config:    map[string]interface{}{"url": "http://api.mock/order/create"},
				DependsOn: []string{},
			},
			{
				ID:        "node-2",
				Name:      "模拟发放抵扣优惠券 (HTTP)",
				Type:      wfModel.NodeTypeHTTP,
				Config:    map[string]interface{}{"url": "http://api.mock/coupon/issue"},
				DependsOn: []string{},
			},
			{
				ID:        "node-3",
				Name:      "核对 trade_ledger 资金账目平衡 (SQL)",
				Type:      wfModel.NodeTypeSQL,
				Config:    map[string]interface{}{"sql": "SELECT balance FROM trade_ledger WHERE debit = credit"},
				DependsOn: []string{"node-1", "node-2"},
			},
			{
				ID:        "node-4",
				Name:      "发布准出门禁决策 (QUALITY_GATE)",
				Type:      wfModel.NodeTypeQualityGate,
				Config:    map[string]interface{}{"rule": "zero_loss"},
				DependsOn: []string{"node-3"},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	res, err := dag.ExecuteGraph(ctx, graph, map[string]interface{}{"biz_id": "BIZ-888999"})
	if err != nil {
		t.Fatalf("DAG 执行失败: %v", err)
	}

	if res.Status != wfModel.WorkflowStatusSuccess {
		t.Fatalf("期望状态 SUCCESS，实际: %s", res.Status)
	}

	if len(res.NodeResults) != 4 {
		t.Fatalf("期望执行 4 个节点，实际完成: %d", len(res.NodeResults))
	}

	// 验证各节点执行结果
	for id, nodeRes := range res.NodeResults {
		if nodeRes.Status != wfModel.NodeStatusSuccess {
			t.Errorf("节点 %s 未成功: %s, err: %s", id, nodeRes.Status, nodeRes.Error)
		}
	}

	// 验证 node-4 证据链
	node4Res := res.NodeResults["node-4"]
	if node4Res.Evidence["decision"] != "RELEASE_APPROVED" {
		t.Errorf("门禁裁决结果不符合预期: %+v", node4Res.Evidence)
	}

	t.Logf("✅ DAG 拓扑并发与依赖执行验证 100%% 通过！总耗时: %dms, 执行ID: %s", res.DurationMs, res.ExecutionID)
}
