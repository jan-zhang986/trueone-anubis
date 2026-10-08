package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Result struct {
	Code    int         `json:"code"`
	Data    interface{} `json:"data"`
	Message string      `json:"message"`
}

type PageResult struct {
	List       interface{} `json:"list"`
	Total      int64       `json:"total"`
	Current    int         `json:"current,omitempty"`
	PageSize   int         `json:"pageSize,omitempty"`
	TotalPages int64       `json:"totalPages,omitempty"`
}

func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Result{
		Code:    200,
		Data:    data,
		Message: "success",
	})
}

func SuccessWithMsg(c *gin.Context, data interface{}, message string) {
	c.JSON(http.StatusOK, Result{
		Code:    200,
		Data:    data,
		Message: message,
	})
}

func SuccessPage(c *gin.Context, list interface{}, total int64, current int, pageSize int) {
	var totalPages int64 = 0
	if pageSize > 0 {
		totalPages = (total + int64(pageSize) - 1) / int64(pageSize)
	}
	c.JSON(http.StatusOK, Result{
		Code: 200,
		Data: PageResult{
			List:       list,
			Total:      total,
			Current:    current,
			PageSize:   pageSize,
			TotalPages: totalPages,
		},
		Message: "success",
	})
}

func Fail(c *gin.Context, code int, message string) {
	c.JSON(http.StatusOK, Result{
		Code:    code,
		Data:    nil,
		Message: message,
	})
}

func Unauthorized(c *gin.Context, message string) {
	c.JSON(http.StatusUnauthorized, Result{
		Code:    100401,
		Data:    nil,
		Message: message,
	})
}
