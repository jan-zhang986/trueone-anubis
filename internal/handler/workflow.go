package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"trueone-anubis/internal/response"
	"trueone-anubis/internal/service"
	wfModel "trueone-anubis/internal/workflow/model"
)

type WorkflowHandler struct {
	svc *service.WorkflowService
}

func NewWorkflowHandler() *WorkflowHandler {
	return &WorkflowHandler{
		svc: service.NewWorkflowService(),
	}
}

type ExecuteWorkflowRequest struct {
	Graph       *wfModel.WorkflowGraph `json:"graph"`
	YAMLContent string                 `json:"yamlContent"`
	Params      map[string]interface{} `json:"params"`
}

// ExecuteGraph 运行工作流图 (支持接收 Graph 结构体或直接接收 YAML 文本)
func (h *WorkflowHandler) ExecuteGraph(c *gin.Context) {
	var req ExecuteWorkflowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "参数解析失败: "+err.Error())
		return
	}

	var result *wfModel.WorkflowExecutionResult
	var err error

	if req.YAMLContent != "" {
		result, err = h.svc.ExecuteYAML(req.YAMLContent, req.Params)
	} else if req.Graph != nil {
		result, err = h.svc.ExecuteGraph(req.Graph, req.Params)
	} else {
		response.Fail(c, http.StatusBadRequest, "必须提供 graph 或 yamlContent 参数")
		return
	}

	if err != nil {
		response.Fail(c, http.StatusInternalServerError, "执行失败: "+err.Error())
		return
	}

	response.Success(c, result)
}
