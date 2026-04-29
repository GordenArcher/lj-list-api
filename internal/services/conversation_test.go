package services

import (
	"context"
	"testing"
	"time"

	"github.com/GordenArcher/lj-list-api/internal/models"
)

type stubConversationRepo struct {
	conversation *models.Conversation
	created      bool
	err          error
}

func (r *stubConversationRepo) FindOrCreateWithInitialMessage(ctx context.Context, userOne, userTwo, senderID, initialMessage string) (*models.Conversation, bool, error) {
	if r.err != nil {
		return nil, false, r.err
	}
	return r.conversation, r.created, nil
}

func (r *stubConversationRepo) FindAllByUser(ctx context.Context, userID string, offset, limit int) ([]models.ConversationWithDetails, error) {
	return nil, nil
}

func (r *stubConversationRepo) CountByUser(ctx context.Context, userID string) (int, error) {
	return 0, nil
}

type stubConversationUserRepo struct {
	user *models.User
	err  error
}

func (r *stubConversationUserRepo) FindByID(ctx context.Context, id string) (*models.User, error) {
	if r.err != nil {
		return nil, r.err
	}
	return r.user, nil
}

func TestConversationServiceStartOrGetReturnsCreatedFlag(t *testing.T) {
	t.Parallel()

	service := &ConversationService{
		conversationRepo: &stubConversationRepo{
			conversation: &models.Conversation{
				ID:             "conv-1",
				ParticipantOne: "customer-1",
				ParticipantTwo: "admin-1",
				CreatedAt:      time.Unix(1, 0).UTC(),
			},
			created: true,
		},
		userRepo: &stubConversationUserRepo{
			user: &models.User{
				ID:          "admin-1",
				Email:       "admin@example.com",
				DisplayName: "Admin",
				Role:        "admin",
			},
		},
	}

	conv, created, err := service.StartOrGet(context.Background(), "customer-1", "admin-1", "hello")
	if err != nil {
		t.Fatalf("StartOrGet returned error: %v", err)
	}
	if !created {
		t.Fatal("expected conversation to be reported as newly created")
	}
	if conv == nil || conv.ID != "conv-1" {
		t.Fatalf("unexpected conversation returned: %#v", conv)
	}
}

func TestConversationServiceStartOrGetReturnsExistingFlag(t *testing.T) {
	t.Parallel()

	service := &ConversationService{
		conversationRepo: &stubConversationRepo{
			conversation: &models.Conversation{
				ID:             "conv-1",
				ParticipantOne: "customer-1",
				ParticipantTwo: "admin-1",
				CreatedAt:      time.Unix(1, 0).UTC(),
			},
			created: false,
		},
		userRepo: &stubConversationUserRepo{
			user: &models.User{
				ID:          "admin-1",
				Email:       "admin@example.com",
				DisplayName: "Admin",
				Role:        "admin",
			},
		},
	}

	_, created, err := service.StartOrGet(context.Background(), "customer-1", "admin-1", "hello")
	if err != nil {
		t.Fatalf("StartOrGet returned error: %v", err)
	}
	if created {
		t.Fatal("expected existing conversation to return created=false")
	}
}
