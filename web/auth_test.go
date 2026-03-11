package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Wsine/feishu2md/core"
	"github.com/gin-gonic/gin"
)

func TestPersistWebConfigDoesNotStoreUserAuth(t *testing.T) {
	restore := snapshotWebAuthState()
	defer restore()

	webConfigPath = filepath.Join(t.TempDir(), "web_config.json")
	webFeishuConfig = core.FeishuConfig{
		AppId:            "cli_xxx",
		AppSecret:        "secret",
		AuthType:         core.AuthTypeUser,
		OAuthRedirectURL: "http://127.0.0.1:8080/auth/callback",
	}
	webFeishuConfig.SetUserAuthState(testUserAuthState())

	if err := persistWebConfig(); err != nil {
		t.Fatalf("persistWebConfig() error = %v", err)
	}

	config, err := core.ReadConfigFromFile(webConfigPath)
	if err != nil {
		t.Fatalf("ReadConfigFromFile() error = %v", err)
	}
	if config.Feishu.HasUserSession() {
		t.Fatal("persisted web config unexpectedly contains user session")
	}
	if config.Feishu.UserName != "" || config.Feishu.UserID != "" || config.Feishu.UserOpenID != "" {
		t.Fatalf("persisted user identity should be empty: %+v", config.Feishu)
	}
}

func TestLoadWebConfigFromFileIgnoresPersistedUserAuth(t *testing.T) {
	restore := snapshotWebAuthState()
	defer restore()

	webConfigPath = filepath.Join(t.TempDir(), "web_config.json")
	config := core.NewConfig("cli_xxx", "secret")
	config.Feishu.AuthType = core.AuthTypeUser
	config.Feishu.OAuthRedirectURL = "http://127.0.0.1:8080/auth/callback"
	config.Feishu.SetUserAuthState(testUserAuthState())
	if err := config.WriteConfig2File(webConfigPath); err != nil {
		t.Fatalf("WriteConfig2File() error = %v", err)
	}

	webFeishuConfig = core.FeishuConfig{}
	if err := loadWebConfigFromFile(); err != nil {
		t.Fatalf("loadWebConfigFromFile() error = %v", err)
	}

	if webFeishuConfig.AppId != "cli_xxx" || webFeishuConfig.AppSecret != "secret" {
		t.Fatalf("loadWebConfigFromFile() did not load app config: %+v", webFeishuConfig)
	}
	if webFeishuConfig.HasUserSession() {
		t.Fatal("loadWebConfigFromFile() should ignore persisted user session")
	}
}

func TestNewWebClientDoesNotFallbackToGlobalUserAuth(t *testing.T) {
	restore := snapshotWebAuthState()
	defer restore()

	gin.SetMode(gin.TestMode)
	webFeishuConfig = core.FeishuConfig{
		AppId:     "cli_xxx",
		AppSecret: "secret",
		AuthType:  core.AuthTypeUser,
	}
	webFeishuConfig.SetUserAuthState(testUserAuthState())
	webSessions = &sessionStore{sessions: map[string]*sessionData{}}

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	request := httptest.NewRequest(http.MethodGet, "/download", nil)
	request.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "session-a"})
	context.Request = request

	_, err := newWebClient(context)
	if !errors.Is(err, core.ErrUserLoginRequired) {
		t.Fatalf("newWebClient() error = %v, want %v", err, core.ErrUserLoginRequired)
	}
}

func TestAuthLogoutOnlyClearsCurrentSession(t *testing.T) {
	restore := snapshotWebAuthState()
	defer restore()

	gin.SetMode(gin.TestMode)
	webSessions = &sessionStore{
		sessions: map[string]*sessionData{
			"session-a": {UserAuth: testUserAuthState()},
			"session-b": {UserAuth: testUserAuthState()},
		},
	}

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	request := httptest.NewRequest(http.MethodGet, "/auth/logout", nil)
	request.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "session-a"})
	context.Request = request

	authLogoutHandler(context)

	if _, ok := webSessions.get("session-a"); ok {
		t.Fatal("logout should clear current session")
	}
	if _, ok := webSessions.get("session-b"); !ok {
		t.Fatal("logout should not clear other sessions")
	}
	if recorder.Code != http.StatusFound {
		t.Fatalf("authLogoutHandler() status = %d, want %d", recorder.Code, http.StatusFound)
	}
}

func snapshotWebAuthState() func() {
	oldConfig := webFeishuConfig
	oldOAuth := webOAuth
	oldSessions := webSessions
	oldPath := webConfigPath

	return func() {
		webFeishuConfig = oldConfig
		webOAuth = oldOAuth
		webSessions = oldSessions
		webConfigPath = oldPath
	}
}

func testUserAuthState() core.UserAuthState {
	return core.UserAuthState{
		AccessToken:  "access-token",
		RefreshToken: "refresh-token",
		UserID:       "u_123",
		OpenID:       "ou_123",
		Name:         "tester",
	}
}

func TestPersistedWebConfigJSONOmitsUserAuthFields(t *testing.T) {
	restore := snapshotWebAuthState()
	defer restore()

	webConfigPath = filepath.Join(t.TempDir(), "web_config.json")
	webFeishuConfig = core.FeishuConfig{
		AppId:            "cli_xxx",
		AppSecret:        "secret",
		AuthType:         core.AuthTypeUser,
		OAuthRedirectURL: "http://127.0.0.1:8080/auth/callback",
	}
	webFeishuConfig.SetUserAuthState(testUserAuthState())

	if err := persistWebConfig(); err != nil {
		t.Fatalf("persistWebConfig() error = %v", err)
	}

	raw, err := os.ReadFile(webConfigPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	var persisted map[string]any
	if err = json.Unmarshal(raw, &persisted); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	feishu, ok := persisted["feishu"].(map[string]any)
	if !ok {
		t.Fatalf("missing feishu config in persisted json: %s", string(raw))
	}
	if _, ok = feishu["user_access_token"]; ok {
		t.Fatalf("persisted json should not contain user_access_token: %s", string(raw))
	}
	if _, ok = feishu["user_refresh_token"]; ok {
		t.Fatalf("persisted json should not contain user_refresh_token: %s", string(raw))
	}
}

func TestInitWebAuthLoadsPersistedAppSettings(t *testing.T) {
	restore := snapshotWebAuthState()
	defer restore()

	configPath := filepath.Join(t.TempDir(), "web_config.json")
	config := core.NewConfig("cli_xxx", "secret")
	config.Feishu.AuthType = core.AuthTypeUser
	config.Feishu.OAuthRedirectURL = "https://example.com/auth/callback"
	config.Feishu.SetUserAuthState(testUserAuthState())
	if err := config.WriteConfig2File(configPath); err != nil {
		t.Fatalf("WriteConfig2File() error = %v", err)
	}

	t.Setenv("FEISHU_APP_ID", "")
	t.Setenv("FEISHU_APP_SECRET", "")
	t.Setenv("FEISHU_AUTH_TYPE", "")
	t.Setenv("FEISHU_OAUTH_REDIRECT_URI", "")
	t.Setenv("FEISHU_WEB_CONFIG_PATH", configPath)

	if err := initWebAuth(); err != nil {
		t.Fatalf("initWebAuth() error = %v", err)
	}

	if webFeishuConfig.AppId != "cli_xxx" || webFeishuConfig.AppSecret != "secret" {
		t.Fatalf("initWebAuth() did not load persisted credentials: %+v", webFeishuConfig)
	}
	if webFeishuConfig.AuthType != core.AuthTypeUser {
		t.Fatalf("initWebAuth() auth type = %q, want %q", webFeishuConfig.AuthType, core.AuthTypeUser)
	}
	if webFeishuConfig.OAuthRedirectURL != "https://example.com/auth/callback" {
		t.Fatalf("initWebAuth() redirect uri = %q", webFeishuConfig.OAuthRedirectURL)
	}
	if webFeishuConfig.HasUserSession() {
		t.Fatal("initWebAuth() should not restore persisted user session")
	}
}

func TestSessionStoreRemovesExpiredSessions(t *testing.T) {
	store := &sessionStore{
		sessions: map[string]*sessionData{
			"expired": {ExpiresAt: time.Now().Add(-time.Minute)},
			"active":  {ExpiresAt: time.Now().Add(time.Hour)},
		},
	}

	if _, ok := store.get("expired"); ok {
		t.Fatal("get() should reject expired session")
	}
	if _, ok := store.sessions["expired"]; ok {
		t.Fatal("expired session should be deleted from store")
	}

	store.lastCleanup = time.Time{}
	store.update("active", func(session *sessionData) {})
	if _, ok := store.sessions["active"]; !ok {
		t.Fatal("active session should remain after cleanup")
	}
}

func TestDownloadHandlerRejectsInvalidURL(t *testing.T) {
	restore := snapshotWebAuthState()
	defer restore()

	gin.SetMode(gin.TestMode)
	webFeishuConfig = core.FeishuConfig{
		AppId:     "cli_xxx",
		AppSecret: "secret",
		AuthType:  core.AuthTypeApp,
	}

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/download?url=not-a-feishu-url", nil)

	downloadHandler(context)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("downloadHandler() status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
}
