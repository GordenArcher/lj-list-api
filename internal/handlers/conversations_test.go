package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/GordenArcher/lj-list-api/internal/models"
	"github.com/gin-gonic/gin"
)

type stubConversationHandlerService struct {
	conversation *models.ConversationWithDetails
	created      bool
	err          error
	listUserID   string
	listResult   []models.ConversationWithDetails
	adminList    []models.ConversationWithDetails
	adminCount   int
	adminCalled  bool
}

func (s *stubConversationHandlerService) StartOrGet(ctx context.Context, customerID, initialMessage string) (*models.ConversationWithDetails, bool, error) {
	if s.err != nil {
		return nil, false, s.err
	}
	return s.conversation, s.created, nil
}

func (s *stubConversationHandlerService) GetUserConversations(ctx context.Context, userID string, offset, limit int) ([]models.ConversationWithDetails, error) {
	s.listUserID = userID
	return s.listResult, nil
}

func (s *stubConversationHandlerService) GetUserConversationsCount(ctx context.Context, userID string) (int, error) {
	return 0, nil
}

func (s *stubConversationHandlerService) GetAdminConversations(ctx context.Context, offset, limit int) ([]models.ConversationWithDetails, error) {
	s.adminCalled = true
	return s.adminList, nil
}

func (s *stubConversationHandlerService) GetAdminConversationsCount(ctx context.Context) (int, error) {
	return s.adminCount, nil
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
		smsService: smsService,
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
		smsService: smsService,
	}

	handler.Create(ctx)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", recorder.Code)
	}
	if smsService.calls != 1 {
		t.Fatalf("expected 1 SMS notification for new conversation, got %d", smsService.calls)
	}
}

func TestConversationListForAdminUsesSharedInbox(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/v1/admin/conversations?page=1&limit=20", nil)
	ctx.Set("user_id", "admin-1")
	ctx.Set("user_role", "admin")

	service := &stubConversationHandlerService{
		adminList: []models.ConversationWithDetails{
			{
				ID: "conv-1",
				OtherUser: models.ConversationUser{
					ID:          "customer-1",
					DisplayName: "Kwame Mensah",
					Role:        "customer",
				},
			},
		},
		adminCount: 1,
		listResult: []models.ConversationWithDetails{},
	}

	handler := &ConversationHandler{
		conversationService: service,
	}

	handler.List(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}
	if !service.adminCalled {
		t.Fatal("expected admin shared inbox lookup to be used")
	}
}
