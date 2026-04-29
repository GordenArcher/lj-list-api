package services

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/GordenArcher/lj-list-api/internal/config"
	"github.com/GordenArcher/lj-list-api/internal/models"
)

const (
	// hubtelMaxRetries is the number of retries after the initial send
	// attempt. A value of 3 means the service will try at most 4 times total.
	hubtelMaxRetries = 3

	// hubtelRetryDelay is the base wait between retries. Each retry uses a
	// linear backoff derived from this base to avoid hammering Hubtel when the
	// failure is transient, such as a temporary upstream outage or rate limit.
	hubtelRetryDelay = 250 * time.Millisecond
)

type SMSService struct {
	cfg        config.Config
	userRepo   smsUserRepository
	httpClient *http.Client
	retryDelay time.Duration
	maxRetries int
}

type smsUserRepository interface {
	FindByID(ctx context.Context, id string) (*models.User, error)
	FindByEmail(ctx context.Context, email string) (*models.User, error)
}

func NewSMSService(cfg config.Config, userRepo smsUserRepository) *SMSService {
	return &SMSService{
		cfg:      cfg,
		userRepo: userRepo,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		retryDelay: hubtelRetryDelay,
		maxRetries: hubtelMaxRetries,
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

// NotifyAdminNewApplication sends a readable operational alert to the
// configured admin number when a customer submits a new application. The
// copy intentionally avoids raw internal IDs and instead summarizes the
// customer name, requested package/order, and repayment context.
func (s *SMSService) NotifyAdminNewApplication(ctx context.Context, customerID string, app *models.Application) {
	adminNumber := strings.TrimSpace(s.cfg.AdminNumber)
	if adminNumber == "" {
		log.Printf("SMS skipped for application %s: ADMIN_NUMBER is not configured", app.ID)
		return
	}

	adminName := s.resolveAdminName(ctx)
	customerName := s.resolveUserName(ctx, customerID, "A customer")
	requestSummary := summarizeApplicationRequest(app)
	institution := strings.TrimSpace(app.Institution)

	message := fmt.Sprintf(
		"Hello %s, %s submitted a new LJ-List application for %s. Total: GHC %d. Monthly repayment: GHC %d. Institution: %s. Please review it in the dashboard.",
		adminName,
		customerName,
		requestSummary,
		app.TotalAmount,
		app.MonthlyAmount,
		valueOrFallback(institution, "not provided"),
	)
	s.SendNotification(adminNumber, message)
}

// NotifyAdminNewMessage alerts the configured admin number about new customer
// chat activity. Messages sent by the admin are intentionally ignored so the
// admin does not receive an SMS for their own outgoing replies. The SMS copy
// is phrased for business use, using names and a short preview instead of an
// internal conversation UUID.
func (s *SMSService) NotifyAdminNewMessage(ctx context.Context, senderID, senderRole, content string) {
	if senderRole == "admin" {
		return
	}

	adminNumber := strings.TrimSpace(s.cfg.AdminNumber)
	if adminNumber == "" {
		log.Printf("SMS skipped for customer message from %s: ADMIN_NUMBER is not configured", senderID)
		return
	}

	adminName := s.resolveAdminName(ctx)
	customerName := s.resolveUserName(ctx, senderID, "A customer")
	message := fmt.Sprintf(
		"Hello %s, %s sent you a new message on LJ-List: \"%s\". Please reply in the dashboard.",
		adminName,
		customerName,
		smsPreview(content, 120),
	)
	s.SendNotification(adminNumber, message)
}

// send validates configuration, prepares the exact Hubtel quick-send URL,
// and then runs a bounded retry loop around the actual HTTP request.
//
// Retries are intentionally conservative. Hubtel quick-send does not expose
// a client-supplied idempotency key, so retrying every failure would increase
// the chance of duplicate messages if Hubtel accepted a request but the
// network failed before we saw the response. We therefore retry only the
// failure modes that are commonly transient: transport errors, 408, 429,
// and 5xx responses.
func (s *SMSService) send(ctx context.Context, phone, message string) error {
	if s.cfg.HubtelClientID == "" {
		return fmt.Errorf("hubtel client id is not configured")
	}
	if s.cfg.HubtelClientSecret == "" {
		return fmt.Errorf("hubtel client secret is not configured")
	}

	// Build the URL once so every retry represents the same logical request.
	endpointURL, err := s.buildQuickSendURL(phone, message)
	if err != nil {
		return err
	}

	for attempt := 0; attempt <= s.maxRetries; attempt++ {
		retryable, err := s.sendOnce(ctx, endpointURL)
		if err == nil {
			log.Printf("SMS sent to %s", phone)
			return nil
		}

		if !retryable || attempt == s.maxRetries {
			return err
		}

		delay := s.retryDelay * time.Duration(attempt+1)
		log.Printf(
			"retrying SMS to %s after attempt %d/%d: %v",
			phone,
			attempt+1,
			s.maxRetries+1,
			err,
		)
		if err := sleepWithContext(ctx, delay); err != nil {
			return fmt.Errorf("wait before sms retry: %w", err)
		}
	}

	return fmt.Errorf("hubtel sms send exhausted retries")
}

// buildQuickSendURL translates our config and message fields into Hubtel's
// quick-send contract. This endpoint expects all credentials and message data
// in the query string, so we keep that mapping in one helper instead of
// duplicating it across the send path.
func (s *SMSService) buildQuickSendURL(phone, message string) (string, error) {
	endpoint, err := url.Parse(strings.TrimSpace(s.cfg.HubtelSMSURL))
	if err != nil {
		return "", fmt.Errorf("parse hubtel sms url: %w", err)
	}

	query := endpoint.Query()
	query.Set("clientid", s.cfg.HubtelClientID)
	query.Set("clientsecret", s.cfg.HubtelClientSecret)
	query.Set("from", s.cfg.HubtelSenderID)
	query.Set("to", phone)
	query.Set("content", message)
	endpoint.RawQuery = query.Encode()

	return endpoint.String(), nil
}

// sendOnce performs one HTTP attempt and classifies the result as retryable or
// final. The retry loop above does not need to know whether the failure came
// from the transport layer or from Hubtel's HTTP status.
func (s *SMSService) sendOnce(ctx context.Context, endpointURL string) (bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpointURL, nil)
	if err != nil {
		return false, fmt.Errorf("create sms request: %w", err)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return false, fmt.Errorf("send sms request: %w", ctx.Err())
		}
		return true, fmt.Errorf("send sms request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return false, nil
	}

	body, readErr := io.ReadAll(io.LimitReader(resp.Body, 2048))
	if readErr != nil {
		return shouldRetryStatus(resp.StatusCode), fmt.Errorf("hubtel returned status %d", resp.StatusCode)
	}

	return shouldRetryStatus(resp.StatusCode), fmt.Errorf(
		"hubtel returned status %d: %s",
		resp.StatusCode,
		strings.TrimSpace(string(body)),
	)
}

// shouldRetryStatus restricts retries to statuses that usually mean "try
// later". We intentionally do not retry general 4xx responses because they
// usually indicate bad credentials, invalid sender IDs, or malformed numbers.
func shouldRetryStatus(statusCode int) bool {
	switch statusCode {
	case http.StatusRequestTimeout, http.StatusTooManyRequests:
		return true
	default:
		return statusCode >= 500
	}
}

// sleepWithContext lets retries back off without making shutdown or request
// cancellation wait for the full timer duration.
func sleepWithContext(ctx context.Context, delay time.Duration) error {
	if delay <= 0 {
		return nil
	}

	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func smsPreview(content string, maxLen int) string {
	trimmed := strings.Join(strings.Fields(content), " ")
	if maxLen <= 0 || len(trimmed) <= maxLen {
		return trimmed
	}
	if maxLen <= 3 {
		return trimmed[:maxLen]
	}
	return trimmed[:maxLen-3] + "..."
}

func (s *SMSService) resolveAdminName(ctx context.Context) string {
	return s.resolveUserNameByEmail(ctx, s.cfg.AdminEmail, "Admin")
}

func (s *SMSService) resolveUserName(ctx context.Context, userID, fallback string) string {
	if s.userRepo == nil {
		return fallback
	}

	normalizedID := strings.TrimSpace(userID)
	if normalizedID == "" {
		return fallback
	}

	user, err := s.userRepo.FindByID(smsLookupContext(ctx), normalizedID)
	if err != nil {
		return fallback
	}

	return bestDisplayName(user, fallback)
}

func (s *SMSService) resolveUserNameByEmail(ctx context.Context, email, fallback string) string {
	if s.userRepo == nil {
		return fallback
	}

	normalizedEmail := strings.TrimSpace(email)
	if normalizedEmail == "" {
		return fallback
	}

	user, err := s.userRepo.FindByEmail(smsLookupContext(ctx), normalizedEmail)
	if err != nil {
		return fallback
	}

	return bestDisplayName(user, fallback)
}

func smsLookupContext(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return context.WithoutCancel(ctx)
}

func bestDisplayName(user *models.User, fallback string) string {
	if user == nil {
		return fallback
	}
	if trimmed := strings.TrimSpace(user.DisplayName); trimmed != "" {
		return trimmed
	}
	if trimmed := strings.TrimSpace(user.Email); trimmed != "" {
		return trimmed
	}
	return fallback
}

func summarizeApplicationRequest(app *models.Application) string {
	if app == nil {
		return "an application"
	}

	if packageName := strings.TrimSpace(app.PackageName); packageName != "" {
		return packageName
	}

	if strings.EqualFold(strings.TrimSpace(app.PackageType), "custom") {
		if len(app.CartItems) == 0 {
			return "a custom grocery order"
		}

		itemSummaries := make([]string, 0, len(app.CartItems))
		for i, item := range app.CartItems {
			if i == 3 {
				break
			}

			name := strings.TrimSpace(item.Name)
			if name == "" {
				name = "item"
			}
			itemSummaries = append(itemSummaries, fmt.Sprintf("%s x%d", name, item.Quantity))
		}

		summary := "a custom grocery order"
		if len(itemSummaries) > 0 {
			summary += " (" + strings.Join(itemSummaries, ", ")
			if len(app.CartItems) > len(itemSummaries) {
				summary += ", and more"
			}
			summary += ")"
		}

		return summary
	}

	if packageType := strings.TrimSpace(app.PackageType); packageType != "" {
		return packageType + " application"
	}

	return "an application"
}

func valueOrFallback(value, fallback string) string {
	if trimmed := strings.TrimSpace(value); trimmed != "" {
		return trimmed
	}
	return fallback
}
