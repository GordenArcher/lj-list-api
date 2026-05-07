package services

import (
	"context"
	"testing"
	"time"

	"github.com/GordenArcher/lj-list-api/internal/config"
)

type stubMessageNotificationRepo struct {
	shouldNotify bool
	messageCount int
	registers    int
	resets       int
	cooldown     time.Duration
}

func (r *stubMessageNotificationRepo) RegisterCustomerMessageNotification(ctx context.Context, conversationID string, cooldown time.Duration) (bool, int, error) {
	r.registers++
	r.cooldown = cooldown
	return r.shouldNotify, r.messageCount, nil
}

func (r *stubMessageNotificationRepo) ResetMessageNotificationThrottle(ctx context.Context, conversationID string) error {
	r.resets++
	return nil
}

type stubMessageNotificationSMS struct {
	newMessages int
	summaries   int
	lastCount   int
}

func (s *stubMessageNotificationSMS) NotifyAdminNewMessage(ctx context.Context, senderID, senderRole, content string) {
	s.newMessages++
}

func (s *stubMessageNotificationSMS) NotifyAdminMessageSummary(ctx context.Context, senderID string, messageCount int, latestContent string) {
	s.summaries++
	s.lastCount = messageCount
}

func TestMessageNotificationServiceSendsFirstCustomerMessage(t *testing.T) {
	t.Parallel()

	repo := &stubMessageNotificationRepo{shouldNotify: true, messageCount: 1}
	sms := &stubMessageNotificationSMS{}
	service := &MessageNotificationService{
		repo:     repo,
		sms:      sms,
		cooldown: 30 * time.Minute,
	}

	service.NotifyMessage(context.Background(), "conv-1", "customer-1", "customer", "hello")

	if repo.registers != 1 {
		t.Fatalf("expected customer message to be registered, got %d", repo.registers)
	}
	if repo.cooldown != 30*time.Minute {
		t.Fatalf("expected 30 minute cooldown, got %s", repo.cooldown)
	}
	if sms.newMessages != 1 || sms.summaries != 0 {
		t.Fatalf("expected one direct SMS, got new=%d summary=%d", sms.newMessages, sms.summaries)
	}
}

func TestMessageNotificationServiceSuppressesCustomerMessageInsideCooldown(t *testing.T) {
	t.Parallel()

	repo := &stubMessageNotificationRepo{shouldNotify: false}
	sms := &stubMessageNotificationSMS{}
	service := &MessageNotificationService{
		repo:     repo,
		sms:      sms,
		cooldown: 30 * time.Minute,
	}

	service.NotifyMessage(context.Background(), "conv-1", "customer-1", "customer", "hello again")

	if repo.registers != 1 {
		t.Fatalf("expected customer message to be registered, got %d", repo.registers)
	}
	if sms.newMessages != 0 || sms.summaries != 0 {
		t.Fatalf("expected no SMS inside cooldown, got new=%d summary=%d", sms.newMessages, sms.summaries)
	}
}

func TestMessageNotificationServiceSendsSummaryAfterCooldown(t *testing.T) {
	t.Parallel()

	repo := &stubMessageNotificationRepo{shouldNotify: true, messageCount: 8}
	sms := &stubMessageNotificationSMS{}
	service := &MessageNotificationService{
		repo:     repo,
		sms:      sms,
		cooldown: 30 * time.Minute,
	}

	service.NotifyMessage(context.Background(), "conv-1", "customer-1", "customer", "latest")

	if sms.newMessages != 0 || sms.summaries != 1 {
		t.Fatalf("expected one summary SMS, got new=%d summary=%d", sms.newMessages, sms.summaries)
	}
	if sms.lastCount != 8 {
		t.Fatalf("expected summary count 8, got %d", sms.lastCount)
	}
}

func TestMessageNotificationServiceResetsWhenAdminReplies(t *testing.T) {
	t.Parallel()

	repo := &stubMessageNotificationRepo{}
	sms := &stubMessageNotificationSMS{}
	service := &MessageNotificationService{
		repo:     repo,
		sms:      sms,
		cooldown: 30 * time.Minute,
	}

	service.NotifyMessage(context.Background(), "conv-1", "admin-1", "admin", "reply")

	if repo.resets != 1 {
		t.Fatalf("expected admin reply to reset throttle, got %d", repo.resets)
	}
	if repo.registers != 0 {
		t.Fatalf("expected admin reply not to register customer notification, got %d", repo.registers)
	}
	if sms.newMessages != 0 || sms.summaries != 0 {
		t.Fatalf("expected no admin SMS for admin reply, got new=%d summary=%d", sms.newMessages, sms.summaries)
	}
}

func TestNewMessageNotificationServiceUsesConfigCooldown(t *testing.T) {
	t.Parallel()

	repo := &stubMessageNotificationRepo{}
	sms := &stubMessageNotificationSMS{}
	service := NewMessageNotificationServiceForTest(repo, sms, config.Config{MessageSMSCooldownMinutes: 30})

	if service.cooldown != 30*time.Minute {
		t.Fatalf("expected configured cooldown, got %s", service.cooldown)
	}
}

func NewMessageNotificationServiceForTest(repo messageNotificationRepository, sms messageNotificationSMSService, cfg config.Config) *MessageNotificationService {
	return &MessageNotificationService{
		repo:     repo,
		sms:      sms,
		cooldown: time.Duration(cfg.MessageSMSCooldownMinutes) * time.Minute,
	}
}
