package processor_test

import (
	"context"
	"testing"

	"trueone-anubis/internal/workflow/processor"
	_ "trueone-anubis/internal/workflow/processor/api"
	_ "trueone-anubis/internal/workflow/processor/data"
	_ "trueone-anubis/internal/workflow/processor/gate"
)

func TestProcessorRegistry_BuiltinProcessors(t *testing.T) {
	reg := processor.GetRegistry()

	// 1. 验证内置 HTTP 处理器
	httpProc, ok := reg.Get("HTTP")
	if !ok {
		t.Fatalf("内置 HTTP 处理器应已被注册")
	}
	if httpProc.GetMetadata().Category != "api" {
		t.Errorf("HTTP 处理器分类应为 api, 得到: %s", httpProc.GetMetadata().Category)
	}

	// 验证必填字段校验
	err := httpProc.ValidateConfig(map[string]interface{}{"method": "POST"})
	if err == nil {
		t.Errorf("缺少 url 字段时应该返回校验错误")
	}

	// 2. 验证内置 SQL 处理器
	sqlProc, ok := reg.Get("SQL")
	if !ok {
		t.Fatalf("内置 SQL 处理器应已被注册")
	}
	if sqlProc.GetMetadata().Category != "data" {
		t.Errorf("SQL 处理器分类应为 data, 得到: %s", sqlProc.GetMetadata().Category)
	}

	// 3. 验证内置 QUALITY_GATE 处理器
	gateProc, ok := reg.Get("QUALITY_GATE")
	if !ok {
		t.Fatalf("内置 QUALITY_GATE 处理器应已被注册")
	}
	if gateProc.GetMetadata().Category != "gate" {
		t.Errorf("QUALITY_GATE 处理器分类应为 gate, 得到: %s", gateProc.GetMetadata().Category)
	}
}

// 模拟测试：外部通过统一接口扩展一个新插件（例如 DUBBO），验证是否开箱即用
type MockDubboProcessor struct {
	processor.BaseProcessor
}

func (m *MockDubboProcessor) Execute(ctx context.Context, execCtx *processor.ExecutionContext) (*processor.ExecutionResult, error) {
	return &processor.ExecutionResult{
		Status: "SUCCESS",
		Output: map[string]interface{}{"dubbo_res": "ok"},
	}, nil
}

func TestProcessorRegistry_DynamicPluginExtension(t *testing.T) {
	reg := processor.GetRegistry()

	// 动态注册 DUBBO 插件
	reg.Register(&MockDubboProcessor{
		BaseProcessor: processor.BaseProcessor{
			Metadata: processor.ProcessorMetadata{
				Type:     "DUBBO",
				Name:     "Dubbo RPC 泛化调用插件",
				Category: "rpc",
			},
		},
	})

	proc, ok := reg.Get("DUBBO")
	if !ok {
		t.Fatalf("动态注册的 DUBBO 插件应能正常获取")
	}

	res, err := proc.Execute(context.Background(), &processor.ExecutionContext{
		NodeType: "DUBBO",
		Config:   map[string]interface{}{"interface": "com.order.OrderService"},
	})
	if err != nil || res.Status != "SUCCESS" {
		t.Fatalf("DUBBO 插件执行失败: %v", err)
	}
}

func TestProcessorRegistry_UnregisteredTypeReturnsNotFound(t *testing.T) {
	reg := processor.NewProcessorRegistry()

	// 查询一个未注册的协议，必须严谨地返回 false，杜绝静默假成功
	_, ok := reg.Get("UNKNOWN_PROTOCOL")
	if ok {
		t.Fatalf("未注册的类型必须返回 false，不能被非法匹配")
	}
}
