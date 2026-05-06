package services

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/GordenArcher/lj-list-api/internal/config"
	"github.com/GordenArcher/lj-list-api/internal/models"
)

type stubSMSUserRepo struct {
	usersByID          map[string]*models.User
	usersByPhoneNumber map[string]*models.User
	users              []models.User
}

func (r *stubSMSUserRepo) FindByID(ctx context.Context, id string) (*models.User, error) {
	return r.usersByID[id], nil
}

func (r *stubSMSUserRepo) FindByPhoneNumber(ctx context.Context, phoneNumber string) (*models.User, error) {
	if user := r.usersByPhoneNumber[phoneNumber]; user != nil {
		return user, nil
	}
	if strings.HasPrefix(phoneNumber, "+") {
		return r.usersByPhoneNumber[strings.TrimPrefix(phoneNumber, "+")], nil
	}
	return r.usersByPhoneNumber["+"+phoneNumber], nil
}

func (r *stubSMSUserRepo) FindAll(ctx context.Context, role string, offset, limit int) ([]models.User, error) {
	return r.users, nil
}

func (r *stubSMSUserRepo) CountAll(ctx context.Context, role string) (int, error) {
	return len(r.users), nil
}

func TestSMSServiceSendBuildsHubtelQuickSendRequest(t *testing.T) {
	t.Parallel()

	var gotMethod string
	var gotPath string
	var gotQuery url.Values

	service := NewSMSService(config.Config{
		HubtelClientID:     "client-id",
		HubtelClientSecret: "client-secret",
		HubtelSMSURL:       "https://smsc.hubtel.com/v1/messages/send",
		HubtelSenderID:     "SendCore",
	}, nil)
	service.retryDelay = 0
	service.httpClient = &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			gotMethod = r.Method
			gotPath = r.URL.Path
			gotQuery = r.URL.Query()
			return &http.Response{
				StatusCode: http.StatusCreated,
				Body:       http.NoBody,
				Header:     make(http.Header),
			}, nil
		}),
	}

	err := service.send(context.Background(), "233546214293", "This Is A Test Message")
	if err != nil {
		t.Fatalf("send returned error: %v", err)
	}

	if gotMethod != http.MethodGet {
		t.Fatalf("expected GET request, got %s", gotMethod)
	}
	if gotPath != "/v1/messages/send" {
		t.Fatalf("expected path /v1/messages/send, got %s", gotPath)
	}
	if gotQuery.Get("clientid") != "client-id" {
		t.Fatalf("expected clientid query param to be set")
	}
	if gotQuery.Get("clientsecret") != "client-secret" {
		t.Fatalf("expected clientsecret query param to be set")
	}
	if gotQuery.Get("from") != "SendCore" {
		t.Fatalf("expected from query param to be set")
	}
	if gotQuery.Get("to") != "233546214293" {
		t.Fatalf("expected to query param to be set")
	}
	if gotQuery.Get("content") != "This Is A Test Message" {
		t.Fatalf("expected content query param to be set")
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func TestSMSServiceSendRequiresClientSecret(t *testing.T) {
	t.Parallel()

	service := NewSMSService(config.Config{
		HubtelClientID: "client-id",
		HubtelSMSURL:   "https://smsc.hubtel.com/v1/messages/send",
		HubtelSenderID: "SendCore",
	}, nil)

	err := service.send(context.Background(), "233546214293", "This Is A Test Message")
	if err == nil {
		t.Fatal("expected missing client secret error")
	}
}

func TestSMSServiceSendRetriesTransientFailures(t *testing.T) {
	t.Parallel()

	attempts := 0
	service := NewSMSService(config.Config{
		HubtelClientID:     "client-id",
		HubtelClientSecret: "client-secret",
		HubtelSMSURL:       "https://smsc.hubtel.com/v1/messages/send",
		HubtelSenderID:     "SendCore",
	}, nil)
	service.retryDelay = 0
	service.httpClient = &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			attempts++
			if attempts < 3 {
				return &http.Response{
					StatusCode: http.StatusBadGateway,
					Body:       io.NopCloser(strings.NewReader("temporary upstream failure")),
					Header:     make(http.Header),
				}, nil
			}
			return &http.Response{
				StatusCode: http.StatusCreated,
				Body:       http.NoBody,
				Header:     make(http.Header),
			}, nil
		}),
	}

	err := service.send(context.Background(), "233546214293", "This Is A Test Message")
	if err != nil {
		t.Fatalf("send returned error: %v", err)
	}
	if attempts != 3 {
		t.Fatalf("expected 3 attempts, got %d", attempts)
	}
}

func TestSMSServiceSendDoesNotRetryClientErrors(t *testing.T) {
	t.Parallel()

	attempts := 0
	service := NewSMSService(config.Config{
		HubtelClientID:     "client-id",
		HubtelClientSecret: "client-secret",
		HubtelSMSURL:       "https://smsc.hubtel.com/v1/messages/send",
		HubtelSenderID:     "SendCore",
	}, nil)
	service.retryDelay = 0
	service.httpClient = &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			attempts++
			return &http.Response{
				StatusCode: http.StatusBadRequest,
				Body:       io.NopCloser(strings.NewReader("invalid sender id")),
				Header:     make(http.Header),
			}, nil
		}),
	}

	err := service.send(context.Background(), "233546214293", "This Is A Test Message")
	if err == nil {
		t.Fatal("expected client error")
	}
	if attempts != 1 {
		t.Fatalf("expected 1 attempt, got %d", attempts)
	}
}

func TestNotifyAdminNewApplicationBuildsReadableMessage(t *testing.T) {
	t.Parallel()

	var gotQuery url.Values
	done := make(chan struct{}, 1)
	service := NewSMSService(config.Config{
		HubtelClientID:     "client-id",
		HubtelClientSecret: "client-secret",
		HubtelSMSURL:       "https://smsc.hubtel.com/v1/messages/send",
		HubtelSenderID:     "SendCore",
	}, &stubSMSUserRepo{
		usersByID: map[string]*models.User{
			"customer-1": {
				ID:          "customer-1",
				DisplayName: "Kwame Mensah",
				PhoneNumber: "+233240000000",
				Role:        "customer",
			},
		},
		users: []models.User{
			{
				ID:          "admin-1",
				DisplayName: "Archer",
				PhoneNumber: "+233500000999",
				Role:        "admin",
			},
		},
	})
	service.retryDelay = 0
	service.httpClient = &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			gotQuery = r.URL.Query()
			done <- struct{}{}
			return &http.Response{
				StatusCode: http.StatusCreated,
				Body:       http.NoBody,
				Header:     make(http.Header),
			}, nil
		}),
	}

	service.NotifyAdminNewApplication(context.Background(), "customer-1", &models.Application{
		ID:            "app-123",
		PackageName:   "Abusua Asomdwee",
		TotalAmount:   569,
		MonthlyAmount: 190,
		Institution:   "GRA",
	})

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for sms send")
	}

	if gotQuery.Get("to") != "+233500000999" {
		t.Fatalf("expected admin number destination, got %q", gotQuery.Get("to"))
	}
	content := gotQuery.Get("content")
	if !strings.Contains(content, "Hello Archer") {
		t.Fatalf("expected admin greeting in sms content, got %q", content)
	}
	if !strings.Contains(content, "Kwame Mensah") {
		t.Fatalf("expected customer name in sms content, got %q", content)
	}
	if !strings.Contains(content, "Abusua Asomdwee") {
		t.Fatalf("expected package summary in sms content, got %q", content)
	}
	if strings.Contains(content, "app-123") {
		t.Fatalf("did not expect raw application id in sms content, got %q", content)
	}
}

func TestNotifyAdminNewMessageSkipsAdminSender(t *testing.T) {
	t.Parallel()

	attempts := 0
	service := NewSMSService(config.Config{
		HubtelClientID:     "client-id",
		HubtelClientSecret: "client-secret",
		HubtelSMSURL:       "https://smsc.hubtel.com/v1/messages/send",
		HubtelSenderID:     "SendCore",
	}, nil)
	service.retryDelay = 0
	service.httpClient = &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			attempts++
			return &http.Response{
				StatusCode: http.StatusCreated,
				Body:       http.NoBody,
				Header:     make(http.Header),
			}, nil
		}),
	}

	service.NotifyAdminNewMessage(context.Background(), "admin-1", "admin", "hello")

	if attempts != 0 {
		t.Fatalf("expected no sms send for admin-authored message, got %d attempts", attempts)
	}
}

func TestNotifyAdminNewMessageBuildsReadableMessage(t *testing.T) {
	t.Parallel()

	var gotQuery url.Values
	done := make(chan struct{}, 1)
	service := NewSMSService(config.Config{
		HubtelClientID:     "client-id",
		HubtelClientSecret: "client-secret",
		HubtelSMSURL:       "https://smsc.hubtel.com/v1/messages/send",
		HubtelSenderID:     "SendCore",
	}, &stubSMSUserRepo{
		usersByID: map[string]*models.User{
			"customer-1": {
				ID:          "customer-1",
				DisplayName: "Kwame Mensah",
				PhoneNumber: "+233240000000",
				Role:        "customer",
			},
		},
		users: []models.User{
			{
				ID:          "admin-1",
				DisplayName: "Archer",
				PhoneNumber: "+233500000999",
				Role:        "admin",
			},
		},
	})
	service.retryDelay = 0
	service.httpClient = &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			gotQuery = r.URL.Query()
			done <- struct{}{}
			return &http.Response{
				StatusCode: http.StatusCreated,
				Body:       http.NoBody,
				Header:     make(http.Header),
			}, nil
		}),
	}

	service.NotifyAdminNewMessage(context.Background(), "customer-1", "customer", "Please help me with my application status")

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for sms send")
	}

	content := gotQuery.Get("content")
	if !strings.Contains(content, "Hello Archer") {
		t.Fatalf("expected admin greeting in sms content, got %q", content)
	}
	if !strings.Contains(content, "Kwame Mensah") {
		t.Fatalf("expected customer name in sms content, got %q", content)
	}
	if !strings.Contains(content, "Please help me with my application status") {
		t.Fatalf("expected message preview in sms content, got %q", content)
	}
	if strings.Contains(content, "conversation") {
		t.Fatalf("did not expect internal conversation wording in sms content, got %q", content)
	}
}
