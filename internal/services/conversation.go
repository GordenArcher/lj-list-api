package services

import (
	"context"
	"errors"

	"github.com/GordenArcher/lj-list-api/internal/apperrors"
	"github.com/GordenArcher/lj-list-api/internal/models"
	"github.com/GordenArcher/lj-list-api/internal/repositories"
	"github.com/jackc/pgx/v5"
)

type ConversationService struct {
	conversationRepo conversationRepository
	userRepo         conversationUserRepository
}

type conversationRepository interface {
	FindOrCreateWithInitialMessage(ctx context.Context, userOne, userTwo, senderID, initialMessage string) (*models.Conversation, bool, error)
	FindAllByUser(ctx context.Context, userID string, offset, limit int) ([]models.ConversationWithDetails, error)
	CountByUser(ctx context.Context, userID string) (int, error)
	FindAllForAdmin(ctx context.Context, offset, limit int) ([]models.ConversationWithDetails, error)
	CountAllForAdmin(ctx context.Context) (int, error)
}

type conversationUserRepository interface {
	FindByID(ctx context.Context, id string) (*models.User, error)
	FindAll(ctx context.Context, role string, offset, limit int) ([]models.User, error)
}

func NewConversationService(
	conversationRepo *repositories.ConversationRepository,
	userRepo *repositories.UserRepository,
) *ConversationService {
	return &ConversationService{
		conversationRepo: conversationRepo,
		userRepo:         userRepo,
	}
}

// StartOrGet finds an existing conversation between the customer and the
// bootstrap admin, or creates one. The initial message is only inserted when
// the conversation is created for the first time, keeping the endpoint
// idempotent for retries or repeated "start conversation" actions. The bool
// return reports whether a new conversation was created.
func (s *ConversationService) StartOrGet(ctx context.Context, customerID, initialMessage string) (*models.ConversationWithDetails, bool, error) {
	admins, err := s.userRepo.FindAll(ctx, "admin", 0, 1)
	if err != nil {
		return nil, false, apperrors.Wrap(apperrors.KindInternal, "Failed to select admin recipient", err)
	}
	if len(admins) == 0 {
		return nil, false, apperrors.New(apperrors.KindNotFound, "Admin account not found", map[string][]string{
			"admin": {"no admin accounts are available"},
		})
	}

	adminID := admins[0].ID
	conv, created, err := s.conversationRepo.FindOrCreateWithInitialMessage(ctx, customerID, adminID, customerID, initialMessage)
	if err != nil {
		return nil, false, apperrors.Wrap(apperrors.KindInternal, "Failed to start conversation", err)
	}

	details, err := s.buildConversationDetails(ctx, conv, customerID)
	if err != nil {
		return nil, false, err
	}

	return details, created, nil
}

// GetUserConversations returns paginated conversations for a user with the other
// participant's profile and unread counts.
func (s *ConversationService) GetUserConversations(ctx context.Context, userID string, offset, limit int) ([]models.ConversationWithDetails, error) {
	conversations, err := s.conversationRepo.FindAllByUser(ctx, userID, offset, limit)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.KindInternal, "Failed to retrieve conversations", err)
	}
	return conversations, nil
}

// GetUserConversationsCount returns the total count of conversations for a user.
func (s *ConversationService) GetUserConversationsCount(ctx context.Context, userID string) (int, error) {
	count, err := s.conversationRepo.CountByUser(ctx, userID)
	if err != nil {
		return 0, apperrors.Wrap(apperrors.KindInternal, "Failed to retrieve conversation count", err)
	}
	return count, nil
}

// GetAdminConversations returns every customer conversation for the shared
// inbox so any admin can see and reply to any thread.
func (s *ConversationService) GetAdminConversations(ctx context.Context, offset, limit int) ([]models.ConversationWithDetails, error) {
	conversations, err := s.conversationRepo.FindAllForAdmin(ctx, offset, limit)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.KindInternal, "Failed to retrieve conversations", err)
	}
	return conversations, nil
}

// GetAdminConversationsCount returns the total number of customer threads in
// the shared admin inbox.
func (s *ConversationService) GetAdminConversationsCount(ctx context.Context) (int, error) {
	count, err := s.conversationRepo.CountAllForAdmin(ctx)
	if err != nil {
		return 0, apperrors.Wrap(apperrors.KindInternal, "Failed to retrieve conversation count", err)
	}
	return count, nil
}

// buildConversationDetails fetches the other user's profile and constructs
// a ConversationWithDetails. This is a shared helper, conversation list
// queries do this in SQL, but single conversation lookups do it here.
func (s *ConversationService) buildConversationDetails(ctx context.Context, conv *models.Conversation, currentUserID string) (*models.ConversationWithDetails, error) {
	otherUserID := conv.ParticipantOne
	if otherUserID == currentUserID {
		otherUserID = conv.ParticipantTwo
	}

	otherUser, err := s.userRepo.FindByID(ctx, otherUserID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.New(apperrors.KindNotFound, "Conversation participant not found", nil)
		}
		return nil, apperrors.Wrap(apperrors.KindInternal, "Failed to retrieve conversation participant", err)
	}

	return &models.ConversationWithDetails{
		ID: conv.ID,
		OtherUser: models.ConversationUser{
			ID:          otherUser.ID,
			DisplayName: otherUser.DisplayName,
			PhoneNumber: otherUser.PhoneNumber,
			Role:        otherUser.Role,
		},
		CreatedAt: conv.CreatedAt,
	}, nil
}
