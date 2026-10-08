package handler

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"trueone-anubis/config"
	"trueone-anubis/internal/model"
	"trueone-anubis/internal/response"
)

type MessageHandler struct{}

func NewMessageHandler() *MessageHandler {
	return &MessageHandler{}
}

func (h *MessageHandler) MessageList(c *gin.Context) {
	// Returns message task list for the project
	response.Success(c, []interface{}{})
}

func (h *MessageHandler) MessageSave(c *gin.Context) {
	response.Success(c, "保存成功")
}

func (h *MessageHandler) MessageUserList(c *gin.Context) {
	var users []model.User
	config.DB.Where("deleted = 0").Find(&users)

	type Receiver struct {
		ID    string `json:"id"`
		Name  string `json:"name"`
		Email string `json:"email"`
	}
	res := make([]Receiver, 0, len(users))
	for _, u := range users {
		res = append(res, Receiver{
			ID:    u.ID,
			Name:  u.Name,
			Email: u.Email,
		})
	}
	response.Success(c, res)
}

func (h *MessageHandler) MessageDetail(c *gin.Context) {
	response.Success(c, gin.H{})
}

func (h *MessageHandler) MessageFields(c *gin.Context) {
	response.Success(c, []interface{}{})
}

// Robots
func (h *MessageHandler) RobotList(c *gin.Context) {
	response.Success(c, []interface{}{})
}

func (h *MessageHandler) RobotAdd(c *gin.Context) {
	var body map[string]interface{}
	_ = c.ShouldBindJSON(&body)
	if body == nil {
		body = make(map[string]interface{})
	}
	body["id"] = uuid.New().String()
	body["createdAt"] = time.Now().UnixMilli()
	response.Success(c, body)
}

func (h *MessageHandler) RobotUpdate(c *gin.Context) {
	response.Success(c, "更新成功")
}

func (h *MessageHandler) RobotDelete(c *gin.Context) {
	response.Success(c, "删除成功")
}

func (h *MessageHandler) RobotToggle(c *gin.Context) {
	response.Success(c, "更新成功")
}
