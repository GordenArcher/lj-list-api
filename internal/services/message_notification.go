package services

import (
	"context"
	"log"
	"time"

	"github.com/GordenArcher/lj-list-api/internal/config"
	"github.com/GordenArcher/lj-list-api/internal/repositories"
)

type messageNotificationRepository interface {
	RegisterCustomerMessageNotification(ctx context.Context, conversationID string, cooldown time.Duration) (bool, int, error)
	ResetMessageNotificationThrottle(ctx context.Context, conversationID string) error
}

type messageNotificationSMSService interface {
	NotifyAdminNewMessage(ctx context.Context, senderID, senderRole, content string)
	NotifyAdminMessageSummary(ctx context.Context, senderID string, messageCount int, latestContent string)
}

type MessageNotificationService struct {
	repo     messageNotificationRepository
	sms      messageNotificationSMSService
	cooldown time.Duration
}

func NewMessageNotificationService(repo *repositories.ConversationRepository, sms *SMSService, cfg config.Config) *MessageNotificationService {
	return &MessageNotificationService{
		repo:     repo,
		sms:      sms,
		cooldown: time.Duration(cfg.MessageSMSCooldownMinutes) * time.Minute,
	}
}

func (s *MessageNotificationService) NotifyMessage(ctx context.Context, conversationID, senderID, senderRole, content string) {
	if s == nil || s.repo == nil || s.sms == nil {
		return
	}

	if senderRole == "admin" {
		if err := s.repo.ResetMessageNotificationThrottle(ctx, conversationID); err != nil {
			log.Printf("failed to reset message notification throttle for conversation %s: %v", conversationID, err)
		}
		return
	}

	shouldNotify, messageCount, err := s.repo.RegisterCustomerMessageNotification(ctx, conversationID, s.cooldown)
	if err != nil {
		log.Printf("failed to register customer message notification for conversation %s: %v", conversationID, err)
		return
	}
	if !shouldNotify {
		return
	}

	if messageCount <= 1 {
		s.sms.NotifyAdminNewMessage(ctx, senderID, senderRole, content)
		return
	}
	s.sms.NotifyAdminMessageSummary(ctx, senderID, messageCount, content)
}
