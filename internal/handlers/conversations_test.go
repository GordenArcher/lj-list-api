package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/GordenArcher/lj-list-api/internal/config"
	"github.com/GordenArcher/lj-list-api/internal/models"
	"github.com/gin-gonic/gin"
)

type stubConversationHandlerService struct {
	conversation *models.ConversationWithDetails
	created      bool
	err          error
}

func (s *stubConversationHandlerService) StartOrGet(ctx context.Context, customerID, adminID, initialMessage string) (*models.ConversationWithDetails, bool, error) {
	if s.err != nil {
		return nil, false, s.err
	}
	return s.conversation, s.created, nil
}

func (s *stubConversationHandlerService) GetUserConversations(ctx context.Context, userID string, offset, limit int) ([]models.ConversationWithDetails, error) {
	return nil, nil
}

func (s *stubConversationHandlerService) GetUserConversationsCount(ctx context.Context, userID string) (int, error) {
	return 0, nil
}

type stubConversationHandlerUserRepo struct {
	user *models.User
	err  error
}

func (r *stubConversationHandlerUserRepo) FindByEmail(ctx context.Context, email string) (*models.User, error) {
	if r.err != nil {
		return nil, r.err
	}
	return r.user, nil
}

type stubConversationHandlerSMSService struct {
	calls int
}

func (s *stubConversationHandlerSMSService) NotifyAdminNewMessage(ctx context.Context, senderID, senderRole, content string) {
	s.calls++
}

func TestConversationCreateReturnsExistingConversationWithoutSendingSMS(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/conversations", strings.NewReader(`{"message":"hello"}`))
	ctx.Request.Header.Set("Content-Type", "application/json")
	ctx.Set("user_id", "customer-1")
	ctx.Set("user_role", "customer")

	smsService := &stubConversationHandlerSMSService{}
	handler := &ConversationHandler{
		conversationService: &stubConversationHandlerService{
			conversation: &models.ConversationWithDetails{
				ID:        "conv-1",
				CreatedAt: time.Unix(1, 0).UTC(),
			},
			created: false,
		},
		userRepo: &stubConversationHandlerUserRepo{
			user: &models.User{
				ID:          "admin-1",
				Email:       "admin@example.com",
				DisplayName: "Admin",
				Role:        "admin",
			},
		},
		smsService: smsService,
		cfg: config.Config{
			AdminEmail: "admin@example.com",
		},
	}

	handler.Create(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}
	if smsService.calls != 0 {
		t.Fatalf("expected no SMS notification for existing conversation, got %d", smsService.calls)
	}
}

func TestConversationCreateReturnsCreatedConversationAndSendsSMS(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/conversations", strings.NewReader(`{"message":"hello"}`))
	ctx.Request.Header.Set("Content-Type", "application/json")
	ctx.Set("user_id", "customer-1")
	ctx.Set("user_role", "customer")

	smsService := &stubConversationHandlerSMSService{}
	handler := &ConversationHandler{
		conversationService: &stubConversationHandlerService{
			conversation: &models.ConversationWithDetails{
				ID:        "conv-1",
				CreatedAt: time.Unix(1, 0).UTC(),
			},
			created: true,
		},
		userRepo: &stubConversationHandlerUserRepo{
			user: &models.User{
				ID:          "admin-1",
				Email:       "admin@example.com",
				DisplayName: "Admin",
				Role:        "admin",
			},
		},
		smsService: smsService,
		cfg: config.Config{
			AdminEmail: "admin@example.com",
		},
	}

	handler.Create(ctx)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", recorder.Code)
	}
	if smsService.calls != 1 {
		t.Fatalf("expected 1 SMS notification for new conversation, got %d", smsService.calls)
	}
}
