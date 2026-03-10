package core

import (
	"testing"
	"time"
)

func TestFeishuConfigValidateAllowsUserAuthBeforeLogin(t *testing.T) {
	config := FeishuConfig{
		AppId:     "cli_xxx",
		AppSecret: "secret",
		AuthType:  AuthTypeUser,
	}
	if err := config.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestFeishuConfigUserAuthRoundTrip(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	state := UserAuthState{
		AccessToken:        "access",
		RefreshToken:       "refresh",
		AccessTokenExpiry:  now.Add(time.Hour),
		RefreshTokenExpiry: now.Add(24 * time.Hour),
		UserID:             "ou_xxx",
		OpenID:             "open_xxx",
		Name:               "tester",
	}
	config := FeishuConfig{}
	config.SetUserAuthState(state)

	parsed, err := config.UserAuthState()
	if err != nil {
		t.Fatalf("UserAuthState() error = %v", err)
	}
	if parsed.AccessToken != state.AccessToken || parsed.RefreshToken != state.RefreshToken {
		t.Fatalf("parsed tokens mismatch: %+v", parsed)
	}
	if !parsed.AccessTokenExpiry.Equal(state.AccessTokenExpiry) {
		t.Fatalf("access expiry mismatch: got %v want %v", parsed.AccessTokenExpiry, state.AccessTokenExpiry)
	}
	if !parsed.RefreshTokenExpiry.Equal(state.RefreshTokenExpiry) {
		t.Fatalf("refresh expiry mismatch: got %v want %v", parsed.RefreshTokenExpiry, state.RefreshTokenExpiry)
	}
}
