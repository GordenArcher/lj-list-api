package handlers

import (
	"context"
	"net/http"
	"strings"

	"github.com/GordenArcher/lj-list-api/internal/models"
	"github.com/GordenArcher/lj-list-api/internal/services"
	"github.com/GordenArcher/lj-list-api/internal/utils"
	"github.com/gin-gonic/gin"
)

type conversationService interface {
	StartOrGet(ctx context.Context, customerID, initialMessage string) (*models.ConversationWithDetails, bool, error)
	GetUserConversations(ctx context.Context, userID string, offset, limit int) ([]models.ConversationWithDetails, error)
	GetUserConversationsCount(ctx context.Context, userID string) (int, error)
	GetAdminConversations(ctx context.Context, offset, limit int) ([]models.ConversationWithDetails, error)
	GetAdminConversationsCount(ctx context.Context) (int, error)
}

type conversationSMSService interface {
	NotifyAdminNewMessage(ctx context.Context, senderID, senderRole, content string)
}

type ConversationHandler struct {
	conversationService conversationService
	smsService          conversationSMSService
}

func NewConversationHandler(
	conversationService *services.ConversationService,
	smsService *services.SMSService,
) *ConversationHandler {
	return &ConversationHandler{
		conversationService: conversationService,
		smsService:          smsService,
	}
}

type startConversationRequest struct {
	Message string `json:"message"`
}

func (h *ConversationHandler) Create(c *gin.Context) {
	var req startConversationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusUnprocessableEntity, "INVALID_REQUEST", "Failed to parse request body", map[string][]string{
			"body": {err.Error()},
		})
		return
	}

	if strings.TrimSpace(req.Message) == "" {
		utils.Error(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "Message is required", map[string][]string{
			"message": {"cannot be empty"},
		})
		return
	}

	userID := utils.GetUserIDFromContext(c)

	conv, created, err := h.conversationService.StartOrGet(c.Request.Context(), userID, req.Message)
	if err != nil {
		utils.HandleError(c, err, "Failed to start conversation")
		return
	}

	if created {
		h.smsService.NotifyAdminNewMessage(c.Request.Context(), userID, utils.GetUserRoleFromContext(c), req.Message)
		utils.Success(c, http.StatusCreated, "Conversation started", conv)
		return
	}

	utils.Success(c, http.StatusOK, "Conversation already exists", conv)
}

func (h *ConversationHandler) List(c *gin.Context) {
	userID := utils.GetUserIDFromContext(c)
	userRole := utils.GetUserRoleFromContext(c)

	// Extract pagination parameters from query string.
	// Default: page=1, limit=20. Max limit is 100 to prevent abuse.
	pag := utils.ExtractPaginationParams(c)

	var (
		conversations []models.ConversationWithDetails
		total         int
		err           error
	)

	if userRole == "admin" {
		conversations, err = h.conversationService.GetAdminConversations(c.Request.Context(), pag.Offset, pag.Limit)
		if err != nil {
			utils.HandleError(c, err, "Failed to retrieve conversations")
			return
		}

		total, err = h.conversationService.GetAdminConversationsCount(c.Request.Context())
		if err != nil {
			utils.HandleError(c, err, "Failed to retrieve conversation count")
			return
		}
	} else {
		conversations, err = h.conversationService.GetUserConversations(c.Request.Context(), userID, pag.Offset, pag.Limit)
		if err != nil {
			utils.HandleError(c, err, "Failed to retrieve conversations")
			return
		}

		// Get the total count for pagination metadata.
		total, err = h.conversationService.GetUserConversationsCount(c.Request.Context(), userID)
		if err != nil {
			utils.HandleError(c, err, "Failed to retrieve conversation count")
			return
		}
	}

	utils.Success(c, http.StatusOK, "Conversations retrieved", gin.H{
		"conversations": conversations,
		"meta":          utils.BuildPaginationMeta(total, pag),
	})
}
