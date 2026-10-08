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
	Graph  wfModel.WorkflowGraph  `json:"graph" binding:"required"`
	Params map[string]interface{} `json:"params"`
}

// ExecuteGraph 运行工作流图
func (h *WorkflowHandler) ExecuteGraph(c *gin.Context) {
	var req ExecuteWorkflowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "参数解析失败: "+err.Error())
		return
	}

	result, err := h.svc.ExecuteGraph(&req.Graph, req.Params)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, "执行失败: "+err.Error())
		return
	}

	response.Success(c, result)
}
