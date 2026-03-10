package core

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestOAuthServiceAuthorizeURL(t *testing.T) {
	service := NewOAuthService(FeishuConfig{AppId: "cli_xxx", AppSecret: "secret"})
	got := service.AuthorizeURL("http://127.0.0.1:38080/auth/callback", "state123")
	if got == "" {
		t.Fatal("AuthorizeURL() returned empty string")
	}
	if want := "app_id=cli_xxx"; !contains(got, want) {
		t.Fatalf("AuthorizeURL() missing %q: %s", want, got)
	}
}

func TestRefreshingUserTokenProviderRefreshesExpiredToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case defaultAppTokenPath:
			_, _ = w.Write([]byte(`{"code":0,"msg":"ok","app_access_token":"app-token"}`))
		case defaultRefreshPath:
			_, _ = w.Write([]byte(`{"code":0,"msg":"ok","data":{"access_token":"new-access","expires_in":7200,"refresh_token":"new-refresh","refresh_expires_in":2592000,"user_id":"u_1","open_id":"ou_1","name":"tester"}}`))
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	config := FeishuConfig{
		AppId:                  "cli_xxx",
		AppSecret:              "secret",
		AuthType:               AuthTypeUser,
		UserAccessToken:        "old-access",
		UserRefreshToken:       "old-refresh",
		UserAccessTokenExpiry:  time.Now().Add(-time.Minute).UTC().Format(time.RFC3339),
		UserRefreshTokenExpiry: time.Now().Add(time.Hour).UTC().Format(time.RFC3339),
	}
	var persisted UserAuthState
	provider, err := NewRefreshingUserTokenProvider(config, func(state UserAuthState) error {
		persisted = state
		return nil
	})
	if err != nil {
		t.Fatalf("NewRefreshingUserTokenProvider() error = %v", err)
	}
	provider.oauth.baseURL = server.URL
	provider.oauth.httpClient = server.Client()

	token, err := provider.UserAccessToken(context.Background())
	if err != nil {
		t.Fatalf("UserAccessToken() error = %v", err)
	}
	if token != "new-access" {
		t.Fatalf("UserAccessToken() = %q, want %q", token, "new-access")
	}
	if persisted.RefreshToken != "new-refresh" {
		t.Fatalf("persisted refresh token = %q", persisted.RefreshToken)
	}
}

func contains(input, expect string) bool {
	return strings.Contains(input, expect)
}
