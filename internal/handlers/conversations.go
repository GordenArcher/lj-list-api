package handlers

import (
	"errors"
	"net/http"
	"strings"

	"github.com/GordenArcher/lj-list-api/internal/apperrors"
	"github.com/GordenArcher/lj-list-api/internal/config"
	"github.com/GordenArcher/lj-list-api/internal/repositories"
	"github.com/GordenArcher/lj-list-api/internal/services"
	"github.com/GordenArcher/lj-list-api/internal/utils"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

type ConversationHandler struct {
	conversationService *services.ConversationService
	userRepo            *repositories.UserRepository
	cfg                 config.Config
}

func NewConversationHandler(
	conversationService *services.ConversationService,
	userRepo *repositories.UserRepository,
	cfg config.Config,
) *ConversationHandler {
	return &ConversationHandler{
		conversationService: conversationService,
		userRepo:            userRepo,
		cfg:                 cfg,
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

	// Find the admin user — conversations always include the admin as the
	// other participant. We look up by the configured admin email rather
	// than hardcoding a UUID.
	admin, err := h.userRepo.FindByEmail(c.Request.Context(), h.cfg.AdminEmail)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			utils.HandleError(c, apperrors.New(
				apperrors.KindNotFound,
				"Admin account not found",
				map[string][]string{"admin": {"configured admin account does not exist"}},
			), "")
			return
		}
		utils.HandleError(c, apperrors.Wrap(apperrors.KindInternal, "Failed to find admin user", err), "")
		return
	}

	conv, err := h.conversationService.StartOrGet(c.Request.Context(), userID, admin.ID, req.Message)
	if err != nil {
		utils.HandleError(c, err, "Failed to start conversation")
		return
	}

	utils.Success(c, http.StatusCreated, "Conversation started", conv)
}

func (h *ConversationHandler) List(c *gin.Context) {
	userID := utils.GetUserIDFromContext(c)

	// Extract pagination parameters from query string.
	// Default: page=1, limit=20. Max limit is 100 to prevent abuse.
	pag := utils.ExtractPaginationParams(c)

	conversations, err := h.conversationService.GetUserConversations(c.Request.Context(), userID, pag.Offset, pag.Limit)
	if err != nil {
		utils.HandleError(c, err, "Failed to retrieve conversations")
		return
	}

	// Get the total count for pagination metadata.
	total, err := h.conversationService.GetUserConversationsCount(c.Request.Context(), userID)
	if err != nil {
		utils.HandleError(c, err, "Failed to retrieve conversation count")
		return
	}

	utils.Success(c, http.StatusOK, "Conversations retrieved", gin.H{
		"conversations": conversations,
		"meta":          utils.BuildPaginationMeta(total, pag),
	})
}
