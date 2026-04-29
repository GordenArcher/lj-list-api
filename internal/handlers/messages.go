package handlers

import (
	"net/http"
	"strings"

	"github.com/GordenArcher/lj-list-api/internal/services"
	"github.com/GordenArcher/lj-list-api/internal/utils"
	"github.com/gin-gonic/gin"
)

type MessageHandler struct {
	messageService *services.MessageService
	smsService     *services.SMSService
}

func NewMessageHandler(messageService *services.MessageService, smsService *services.SMSService) *MessageHandler {
	return &MessageHandler{
		messageService: messageService,
		smsService:     smsService,
	}
}

type sendMessageRequest struct {
	Content string `json:"content"`
}

func (h *MessageHandler) Send(c *gin.Context) {
	var req sendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusUnprocessableEntity, "INVALID_REQUEST", "Failed to parse request body", map[string][]string{
			"body": {err.Error()},
		})
		return
	}

	if strings.TrimSpace(req.Content) == "" {
		utils.Error(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "Message cannot be empty", map[string][]string{
			"content": {"cannot be empty"},
		})
		return
	}

	userID := utils.GetUserIDFromContext(c)
	conversationID := c.Param("id")

	msg, err := h.messageService.Send(c.Request.Context(), conversationID, userID, req.Content)
	if err != nil {
		utils.HandleError(c, err, "Failed to send message")
		return
	}

	h.smsService.NotifyAdminNewMessage(c.Request.Context(), userID, utils.GetUserRoleFromContext(c), req.Content)
	utils.Success(c, http.StatusCreated, "Message sent", msg)
}

func (h *MessageHandler) List(c *gin.Context) {
	userID := utils.GetUserIDFromContext(c)
	conversationID := c.Param("id")
	pag := utils.ExtractPaginationParams(c)

	messages, err := h.messageService.GetMessages(c.Request.Context(), conversationID, userID, pag.Offset, pag.Limit)
	if err != nil {
		utils.HandleError(c, err, "Failed to retrieve messages")
		return
	}

	total, err := h.messageService.GetMessagesCount(c.Request.Context(), conversationID, userID)
	if err != nil {
		utils.HandleError(c, err, "Failed to retrieve message count")
		return
	}

	utils.Success(c, http.StatusOK, "Messages retrieved", gin.H{
		"messages": messages,
		"meta":     utils.BuildPaginationMeta(total, pag),
	})
}
