package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/GordenArcher/lj-list-api/internal/config"
	"github.com/GordenArcher/lj-list-api/internal/models"
	"github.com/GordenArcher/lj-list-api/internal/services"
	"github.com/GordenArcher/lj-list-api/internal/utils"
	"github.com/gin-gonic/gin"
)

type stubAuthHandlerService struct {
	signupInput   services.SignupInput
	signupCalled  bool
	verifyPhone   string
	verifyOTP     string
	verifyCalled  bool
	resendPhone   string
	resendCalled  bool
	loginPhone    string
	loginCalled   bool
	signupUser    *models.User
	verifyUser    *models.User
	loginUser     *models.User
	adminUser     *models.User
	tokenPair     *utils.TokenPair
	signupErr     error
	verifyErr     error
	resendErr     error
	loginErr      error
	adminLoginErr error
}

func (s *stubAuthHandlerService) Signup(ctx context.Context, input services.SignupInput) (*models.User, error) {
	s.signupCalled = true
	s.signupInput = input
	if s.signupErr != nil {
		return nil, s.signupErr
	}
	return s.signupUser, nil
}

func (s *stubAuthHandlerService) VerifyOTP(ctx context.Context, phoneNumber, otp string) (*models.User, *utils.TokenPair, error) {
	s.verifyCalled = true
	s.verifyPhone = phoneNumber
	s.verifyOTP = otp
	if s.verifyErr != nil {
		return nil, nil, s.verifyErr
	}
	return s.verifyUser, s.tokenPair, nil
}

func (s *stubAuthHandlerService) ResendOTP(ctx context.Context, phoneNumber string) error {
	s.resendCalled = true
	s.resendPhone = phoneNumber
	return s.resendErr
}

func (s *stubAuthHandlerService) Login(ctx context.Context, phoneNumber, password string) (*models.User, *utils.TokenPair, error) {
	s.loginPhone = phoneNumber
	s.loginCalled = true
	if s.loginErr != nil {
		return nil, nil, s.loginErr
	}
	return s.loginUser, s.tokenPair, nil
}

func (s *stubAuthHandlerService) LoginAdmin(ctx context.Context, phoneNumber, password string) (*models.User, *utils.TokenPair, error) {
	s.loginPhone = phoneNumber
	s.loginCalled = true
	if s.adminLoginErr != nil {
		return nil, nil, s.adminLoginErr
	}
	if s.adminUser != nil {
		return s.adminUser, s.tokenPair, nil
	}
	return s.loginUser, s.tokenPair, nil
}

func (s *stubAuthHandlerService) RefreshTokens(ctx context.Context, refreshToken string) (*utils.TokenPair, error) {
	return s.tokenPair, nil
}

func TestSetAuthCookiesSetsRefreshCookieOnAuthPath(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)

	handler := &AuthHandler{
		cfg: config.Config{
			CookieSameSite: "Lax",
		},
	}

	handler.setAuthCookies(ctx, &utils.TokenPair{
		AccessToken:  "access-token-value",
		RefreshToken: "refresh-token-value",
	})

	response := recorder.Result()
	defer response.Body.Close()

	cookies := response.Cookies()
	if len(cookies) != 2 {
		t.Fatalf("expected 2 cookies, got %d", len(cookies))
	}

	var accessCookieFound bool
	var refreshCookieFound bool

	for _, cookie := range cookies {
		switch cookie.Name {
		case "access_token":
			accessCookieFound = true
			if cookie.Value != "access-token-value" {
				t.Fatalf("unexpected access token cookie value: %q", cookie.Value)
			}
			if cookie.Path != "/" {
				t.Fatalf("unexpected access token cookie path: %q", cookie.Path)
			}
			if !cookie.HttpOnly {
				t.Fatal("expected access token cookie to be httpOnly")
			}
		case "refresh_token":
			refreshCookieFound = true
			if cookie.Value != "refresh-token-value" {
				t.Fatalf("unexpected refresh token cookie value: %q", cookie.Value)
			}
			if cookie.Path != refreshTokenCookiePath {
				t.Fatalf("unexpected refresh token cookie path: %q", cookie.Path)
			}
			if !cookie.HttpOnly {
				t.Fatal("expected refresh token cookie to be httpOnly")
			}
		}
	}

	if !accessCookieFound {
		t.Fatal("expected access_token cookie to be set")
	}
	if !refreshCookieFound {
		t.Fatal("expected refresh_token cookie to be set")
	}
}

func TestSignupUsesNameAliasAndNormalizesPhoneNumber(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	service := &stubAuthHandlerService{
		signupUser: &models.User{
			ID:          "user-1",
			DisplayName: "Kwame Mensah",
			PhoneNumber: "+233240000000",
			Role:        "customer",
		},
	}

	handler := &AuthHandler{
		authService: service,
		cfg: config.Config{
			CookieSameSite: "Lax",
		},
	}

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("request_id", "req-test")
		c.Next()
	})
	router.POST("/signup", handler.Signup)

	body := `{
		"phone_number":"+233 24-000-0000",
		"staff_number":" GES-2024-0018 ",
		"institution":" Ghana Education Service ",
		"ghana_card_number":" GHA-123456789-0 ",
		"password":"password123",
		"name":" Kwame Mensah "
	}`

	req := httptest.NewRequest(http.MethodPost, "/signup", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusAccepted {
		t.Fatalf("expected status 202, got %d", recorder.Code)
	}
	if !service.signupCalled {
		t.Fatal("expected signup service to be called")
	}
	if service.signupInput.DisplayName != "Kwame Mensah" {
		t.Fatalf("expected display name from name alias, got %q", service.signupInput.DisplayName)
	}
	if service.signupInput.PhoneNumber != "+233240000000" {
		t.Fatalf("expected normalized phone number, got %q", service.signupInput.PhoneNumber)
	}
	if service.signupInput.StaffNumber != "GES-2024-0018" {
		t.Fatalf("expected trimmed staff number, got %q", service.signupInput.StaffNumber)
	}
}

func TestSignupValidatesRequiredRegistrationFields(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	service := &stubAuthHandlerService{}
	handler := &AuthHandler{authService: service}

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("request_id", "req-test")
		c.Next()
	})
	router.POST("/signup", handler.Signup)

	body := `{
		"phone_number":"invalid",
		"password":"short",
		"display_name":" "
	}`

	req := httptest.NewRequest(http.MethodPost, "/signup", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected status 422, got %d", recorder.Code)
	}
	if service.signupCalled {
		t.Fatal("expected signup service not to be called on validation failure")
	}
}

func TestVerifyOTPValidatesPhoneNumberAndOTP(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	service := &stubAuthHandlerService{}
	handler := &AuthHandler{authService: service}

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("request_id", "req-test")
		c.Next()
	})
	router.POST("/verify-otp", handler.VerifyOTP)

	req := httptest.NewRequest(http.MethodPost, "/verify-otp", bytes.NewBufferString(`{"phone_number":"abc","otp":" "}`))
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected status 422, got %d", recorder.Code)
	}
	if service.verifyCalled {
		t.Fatal("expected verify service not to be called on validation failure")
	}
}

func TestResendOTPUsesNormalizedPhoneNumber(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	service := &stubAuthHandlerService{}
	handler := &AuthHandler{authService: service}

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("request_id", "req-test")
		c.Next()
	})
	router.POST("/resend-otp", handler.ResendOTP)

	req := httptest.NewRequest(http.MethodPost, "/resend-otp", bytes.NewBufferString(`{"phone_number":"+233 24-000-0000"}`))
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}
	if !service.resendCalled {
		t.Fatal("expected resend service to be called")
	}
	if service.resendPhone != "+233240000000" {
		t.Fatalf("expected normalized phone number, got %q", service.resendPhone)
	}
}

func TestLoginValidatesPhoneNumberAndPassword(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	service := &stubAuthHandlerService{}
	handler := &AuthHandler{authService: service}

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("request_id", "req-test")
		c.Next()
	})
	router.POST("/login", handler.Login)

	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBufferString(`{"phone_number":"abc","password":" "}`))
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected status 422, got %d", recorder.Code)
	}
	if service.loginCalled {
		t.Fatal("expected login service not to be called on validation failure")
	}

	var response models.APIResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if response.Code != "VALIDATION_ERROR" {
		t.Fatalf("expected VALIDATION_ERROR code, got %q", response.Code)
	}
}

func TestAdminLoginUsesDedicatedEndpoint(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	service := &stubAuthHandlerService{
		adminUser: &models.User{
			ID:          "admin-1",
			DisplayName: "Admin User",
			PhoneNumber: "+233240000000",
			Role:        "admin",
		},
		tokenPair: &utils.TokenPair{
			AccessToken:  "access-token",
			RefreshToken: "refresh-token",
		},
	}
	handler := &AuthHandler{authService: service}

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("request_id", "req-test")
		c.Next()
	})
	router.POST("/admin/auth/login", handler.AdminLogin)

	req := httptest.NewRequest(http.MethodPost, "/admin/auth/login", bytes.NewBufferString(`{"phone_number":"+233 24-000-0000","password":"password123"}`))
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}
	if !service.loginCalled {
		t.Fatal("expected admin login service to be called")
	}
	var response models.APIResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if response.Message != "Admin login successful" {
		t.Fatalf("unexpected message: %q", response.Message)
	}
}
