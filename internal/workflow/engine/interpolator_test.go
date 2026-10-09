package engine

import (
	"reflect"
	"testing"
)

func TestInterpolateValue(t *testing.T) {
	scope := NewContextScope(map[string]interface{}{
		"userId": "USR-998811",
		"params": map[string]interface{}{
			"skuId":       "SKU-554422",
			"orderAmount": 199.5,
			"active":      true,
		},
		"step-1": map[string]interface{}{
			"output": map[string]interface{}{
				"orderId": "ORD-20261009-001",
			},
		},
	})

	// 测试 1：全局 params 点路径获取
	res1 := InterpolateValue("{{ params.skuId }}", scope)
	if res1 != "SKU-554422" {
		t.Fatalf("expected SKU-554422, got %v", res1)
	}

	// 测试 2：保留原始非字符串类型 (float64)
	res2 := InterpolateValue("{{ params.orderAmount }}", scope)
	if !reflect.DeepEqual(res2, 199.5) {
		t.Fatalf("expected 199.5 (float), got %v (type %T)", res2, res2)
	}

	// 测试 3：SQL 文本与内嵌变量替换
	sqlTpl := "SELECT * FROM trade_ledger WHERE user_id = '{{ userId }}' AND order_id = '{{ step-1.output.orderId }}';"
	res3 := InterpolateValue(sqlTpl, scope)
	expectedSQL := "SELECT * FROM trade_ledger WHERE user_id = 'USR-998811' AND order_id = 'ORD-20261009-001';"
	if res3 != expectedSQL {
		t.Fatalf("expected %s, got %s", expectedSQL, res3)
	}

	// 测试 4：系统内置动态变量
	resSys := InterpolateValue("trace-{{ sys.date }}", scope)
	if str, ok := resSys.(string); !ok || len(str) < 15 {
		t.Fatalf("unexpected sys date replacement: %v", resSys)
	}

	// 测试 5：Map 嵌套结构深度替换
	body := map[string]interface{}{
		"user": "{{ userId }}",
		"sku":  "{{ params.skuId }}",
		"details": map[string]interface{}{
			"orderId": "{{ step-1.output.orderId }}",
		},
	}
	resMap := InterpolateConfig(body, scope)
	userVal := resMap["user"]
	if userVal != "USR-998811" {
		t.Fatalf("expected USR-998811, got %v", userVal)
	}
	details := resMap["details"].(map[string]interface{})
	if details["orderId"] != "ORD-20261009-001" {
		t.Fatalf("expected ORD-20261009-001, got %v", details["orderId"])
	}
}
