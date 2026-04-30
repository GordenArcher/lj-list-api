package services

import (
	"context"
	"errors"

	"github.com/GordenArcher/lj-list-api/internal/apperrors"
	"github.com/GordenArcher/lj-list-api/internal/models"
	"github.com/GordenArcher/lj-list-api/internal/repositories"
	"github.com/jackc/pgx/v5"
)

type MessageService struct {
	messageRepo      messageRepository
	conversationRepo messageConversationRepository
}

type messageRepository interface {
	Create(ctx context.Context, conversationID, senderID, content string) (*models.Message, error)
	FindByConversationID(ctx context.Context, conversationID string, offset, limit int) ([]models.Message, error)
	CountByConversationID(ctx context.Context, conversationID string) (int, error)
	MarkAsRead(ctx context.Context, conversationID, readerID string) error
}

type messageConversationRepository interface {
	FindByID(ctx context.Context, id string) (*models.Conversation, error)
}

func NewMessageService(
	messageRepo *repositories.MessageRepository,
	conversationRepo *repositories.ConversationRepository,
) *MessageService {
	return &MessageService{
		messageRepo:      messageRepo,
		conversationRepo: conversationRepo,
	}
}

// Send adds a message to a conversation. Customers must be participants in the
// thread. Admins can reply from any authenticated admin account.
func (s *MessageService) Send(ctx context.Context, conversationID, senderID, senderRole, content string) (*models.Message, error) {
	if _, err := s.ensureParticipant(ctx, conversationID, senderID, senderRole); err != nil {
		return nil, err
	}

	msg, err := s.messageRepo.Create(ctx, conversationID, senderID, content)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.KindInternal, "Failed to send message", err)
	}

	return msg, nil
}

// GetMessages returns paginated messages in a conversation. Customers must be
// participants in the thread. Admins can open any conversation. Messages are
// marked as read for the caller after retrieval.
func (s *MessageService) GetMessages(ctx context.Context, conversationID, userID, userRole string, offset, limit int) ([]models.Message, error) {
	if _, err := s.ensureParticipant(ctx, conversationID, userID, userRole); err != nil {
		return nil, err
	}

	messages, err := s.messageRepo.FindByConversationID(ctx, conversationID, offset, limit)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.KindInternal, "Failed to retrieve messages", err)
	}

	// Mark messages from the other participant as read. This happens after
	// retrieval so the unread count is accurate at the moment of fetch.
	if err := s.messageRepo.MarkAsRead(ctx, conversationID, userID); err != nil {
		return nil, apperrors.Wrap(apperrors.KindInternal, "Failed to mark messages as read", err)
	}

	return messages, nil
}

// GetMessagesCount returns the total count of messages in a conversation.
func (s *MessageService) GetMessagesCount(ctx context.Context, conversationID, userID, userRole string) (int, error) {
	if _, err := s.ensureParticipant(ctx, conversationID, userID, userRole); err != nil {
		return 0, err
	}

	count, err := s.messageRepo.CountByConversationID(ctx, conversationID)
	if err != nil {
		return 0, apperrors.Wrap(apperrors.KindInternal, "Failed to retrieve message count", err)
	}

	return count, nil
}

func (s *MessageService) ensureParticipant(ctx context.Context, conversationID, userID, userRole string) (*models.Conversation, error) {
	conv, err := s.conversationRepo.FindByID(ctx, conversationID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.New(apperrors.KindNotFound, "Conversation not found", nil)
		}
		return nil, apperrors.Wrap(apperrors.KindInternal, "Failed to retrieve conversation", err)
	}

	if userRole == "admin" {
		return conv, nil
	}

	if conv.ParticipantOne != userID && conv.ParticipantTwo != userID {
		return nil, apperrors.New(apperrors.KindForbidden, "You do not have access to this conversation", map[string][]string{
			"conversation": {"user is not a participant"},
		})
	}

	return conv, nil
}
