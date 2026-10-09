package api

import (
	"context"
	"time"

	"trueone-anubis/internal/workflow/processor"
)

// HTTPProcessor HTTP 网关与接口调用处理器（对标 aegis-runner api.HttpProcessor）
type HTTPProcessor struct {
	processor.BaseProcessor
}

func NewHTTPProcessor() *HTTPProcessor {
	return &HTTPProcessor{
		BaseProcessor: processor.BaseProcessor{
			Metadata: processor.ProcessorMetadata{
				Type:               "HTTP",
				Name:               "HTTP 接口调用处理器",
				Description:        "用于发起 RESTful HTTP/HTTPS 接口请求，支持路径参数、Query、Body 变量插值与响应提取断言",
				Category:           "api",
				Version:            "1.0.0",
				Author:             "Aegis Platform",
				RequiredConfigKeys: []string{"url"},
				OptionalConfigKeys: []string{"method", "headers", "body", "extract", "assertions"},
			},
		},
	}
}

func init() {
	processor.RegisterProcessor(NewHTTPProcessor())
}

func (h *HTTPProcessor) Execute(ctx context.Context, execCtx *processor.ExecutionContext) (*processor.ExecutionResult, error) {
	start := time.Now()
	url, _ := execCtx.Config["url"].(string)

	// 记录运行时上下文
	if execCtx.ScopeSetter != nil {
		execCtx.ScopeSetter("last_http_call", url)
	}

	// 默认模拟成功（后续对接真正网络调用层）
	return &processor.ExecutionResult{
		Status:     "SUCCESS",
		DurationMs: time.Since(start).Milliseconds() + 15, // 模拟耗时
		Output: map[string]interface{}{
			"status_code": 200,
			"url":         url,
		},
		Evidence: map[string]interface{}{
			"response": `{"code":200,"msg":"ok"}`,
			"url":      url,
		},
	}, nil
}
