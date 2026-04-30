package services

import (
	"context"
	"testing"
	"time"

	"github.com/GordenArcher/lj-list-api/internal/models"
)

type stubConversationRepo struct {
	conversation       *models.Conversation
	created            bool
	err                error
	adminConversations []models.ConversationWithDetails
	adminCount         int
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

func (r *stubConversationRepo) FindAllForAdmin(ctx context.Context, offset, limit int) ([]models.ConversationWithDetails, error) {
	return r.adminConversations, nil
}

func (r *stubConversationRepo) CountAllForAdmin(ctx context.Context) (int, error) {
	return r.adminCount, nil
}

type stubConversationUserRepo struct {
	user  *models.User
	err   error
	users []models.User
}

func (r *stubConversationUserRepo) FindByID(ctx context.Context, id string) (*models.User, error) {
	if r.err != nil {
		return nil, r.err
	}
	return r.user, nil
}

func (r *stubConversationUserRepo) FindAll(ctx context.Context, role string, offset, limit int) ([]models.User, error) {
	if len(r.users) > 0 {
		return r.users, nil
	}
	if r.user != nil {
		return []models.User{*r.user}, nil
	}
	return nil, nil
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
				PhoneNumber: "+233500000001",
				DisplayName: "Admin",
				Role:        "admin",
			},
		},
	}

	conv, created, err := service.StartOrGet(context.Background(), "customer-1", "hello")
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
				PhoneNumber: "+233500000001",
				DisplayName: "Admin",
				Role:        "admin",
			},
		},
	}

	_, created, err := service.StartOrGet(context.Background(), "customer-1", "hello")
	if err != nil {
		t.Fatalf("StartOrGet returned error: %v", err)
	}
	if created {
		t.Fatal("expected existing conversation to return created=false")
	}
}

func TestConversationServiceStartOrGetSelectsFirstAdmin(t *testing.T) {
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
				PhoneNumber: "+233500000001",
				DisplayName: "Admin One",
				Role:        "admin",
			},
			users: []models.User{
				{
					ID:          "admin-1",
					PhoneNumber: "+233500000001",
					DisplayName: "Admin One",
					Role:        "admin",
				},
			},
		},
	}

	conv, created, err := service.StartOrGet(context.Background(), "customer-1", "hello")
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
