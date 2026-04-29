package handlers

import (
	"net/http/httptest"
	"testing"

	"github.com/GordenArcher/lj-list-api/internal/config"
	"github.com/GordenArcher/lj-list-api/internal/utils"
	"github.com/gin-gonic/gin"
)

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
