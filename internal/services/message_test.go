package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/GordenArcher/lj-list-api/internal/apperrors"
	"github.com/GordenArcher/lj-list-api/internal/models"
	"github.com/jackc/pgx/v5"
)

type stubMessageConversationRepo struct {
	conversation *models.Conversation
	err          error
}

func (r *stubMessageConversationRepo) FindByID(ctx context.Context, id string) (*models.Conversation, error) {
	if r.err != nil {
		return nil, r.err
	}
	return r.conversation, nil
}

func TestMessageServiceAllowsAdminAccessToAnyConversation(t *testing.T) {
	t.Parallel()

	service := &MessageService{
		conversationRepo: &stubMessageConversationRepo{
			conversation: &models.Conversation{
				ID:             "conv-1",
				ParticipantOne: "customer-1",
				ParticipantTwo: "admin-1",
				CreatedAt:      time.Unix(1, 0).UTC(),
			},
		},
	}

	conv, err := service.ensureParticipant(context.Background(), "conv-1", "admin-2", "admin")
	if err != nil {
		t.Fatalf("ensureParticipant returned error: %v", err)
	}
	if conv == nil || conv.ID != "conv-1" {
		t.Fatalf("unexpected conversation returned: %#v", conv)
	}
}

func TestMessageServiceRejectsNonParticipantCustomer(t *testing.T) {
	t.Parallel()

	service := &MessageService{
		conversationRepo: &stubMessageConversationRepo{
			conversation: &models.Conversation{
				ID:             "conv-1",
				ParticipantOne: "customer-1",
				ParticipantTwo: "admin-1",
				CreatedAt:      time.Unix(1, 0).UTC(),
			},
		},
	}

	_, err := service.ensureParticipant(context.Background(), "conv-1", "customer-2", "customer")
	if err == nil {
		t.Fatal("expected forbidden error")
	}
	var appErr *apperrors.Error
	if !errors.As(err, &appErr) {
		t.Fatalf("expected app error, got %T", err)
	}
	if appErr.Kind != apperrors.KindForbidden {
		t.Fatalf("expected forbidden error, got %v", appErr.Kind)
	}
	if errors.Is(err, pgx.ErrNoRows) {
		t.Fatal("expected access error, not missing conversation")
	}
}
