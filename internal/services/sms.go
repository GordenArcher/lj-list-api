package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/GordenArcher/lj-list-api/internal/config"
)

type SMSService struct {
	cfg        config.Config
	httpClient *http.Client
}

func NewSMSService(cfg config.Config) *SMSService {
	return &SMSService{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// SendNotification fires an SMS asynchronously. It does not block the
// caller, if Hubtel is down, the API still responds quickly. Failures
// are logged but not surfaced to the user because SMS delivery is not
// critical to the application flow. The admin can always check the
// dashboard for new applications and messages.
func (s *SMSService) SendNotification(phone, message string) {
	go func() {
		if err := s.send(context.Background(), phone, message); err != nil {
			log.Printf("SMS send failed to %s: %v", phone, err)
		}
	}()
}

func (s *SMSService) send(ctx context.Context, phone, message string) error {
	// Hubtel Messaging API expects this exact shape. The sender ID must
	// be pre-registered with Hubtel or it falls back to a shortcode.
	payload := map[string]interface{}{
		"from":    s.cfg.HubtelSenderID,
		"to":      phone,
		"content": message,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal sms payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://sms.hubtel.com/v1/messages/send", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create sms request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Basic "+s.cfg.HubtelAPIKey)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send sms request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("hubtel returned status %d", resp.StatusCode)
	}

	log.Printf("SMS sent to %s", phone)
	return nil
}
