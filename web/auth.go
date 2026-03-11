package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/Wsine/feishu2md/core"
	"github.com/gin-gonic/gin"
)

const (
	defaultWebRedirectURL = "http://127.0.0.1:8080/auth/callback"
	sessionCookieName     = "feishu2md_session"
)

type sessionData struct {
	ExpectedState string
	NextURL       string
	UserAuth      core.UserAuthState
}

type sessionStore struct {
	mu       sync.RWMutex
	sessions map[string]*sessionData
}

var (
	webFeishuConfig core.FeishuConfig
	webOAuth        *core.OAuthService
	webSessions     = &sessionStore{sessions: map[string]*sessionData{}}
	webConfigPath   string
	webConfigMu     sync.RWMutex
)

func initWebAuth() error {
	webConfigPath = readWebConfigPath()
	webFeishuConfig = core.FeishuConfig{
		AppId:            os.Getenv("FEISHU_APP_ID"),
		AppSecret:        os.Getenv("FEISHU_APP_SECRET"),
		AuthType:         readWebAuthType(),
		OAuthRedirectURL: readWebRedirectURL(),
	}
	if err := loadWebConfigFromFile(); err != nil {
		return err
	}
	if !webHasCredentials() {
		return nil
	}
	if err := webFeishuConfig.Validate(); err != nil {
		return err
	}
	webOAuth = core.NewOAuthService(webFeishuConfig)
	return nil
}

func readWebConfigPath() string {
	if value := os.Getenv("FEISHU_WEB_CONFIG_PATH"); value != "" {
		return value
	}
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "web_config.json"
	}
	return filepath.Join(configDir, "feishu2md", "web_config.json")
}

func loadWebConfigFromFile() error {
	raw, err := os.ReadFile(webConfigPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	var config core.Config
	if err := json.Unmarshal(raw, &config); err != nil {
		return err
	}
	if webFeishuConfig.AppId == "" {
		webFeishuConfig.AppId = config.Feishu.AppId
	}
	if webFeishuConfig.AppSecret == "" {
		webFeishuConfig.AppSecret = config.Feishu.AppSecret
	}
	if webFeishuConfig.AuthType == "" {
		webFeishuConfig.AuthType = config.Feishu.AuthType
	}
	if webFeishuConfig.OAuthRedirectURL == "" {
		webFeishuConfig.OAuthRedirectURL = config.Feishu.OAuthRedirectURL
	}
	return nil
}

func persistWebConfig() error {
	conf := core.NewConfig("", "")
	conf.Feishu = webPersistedConfig()
	return conf.WriteConfig2File(webConfigPath)
}

func webPersistedConfig() core.FeishuConfig {
	config := webFeishuConfig
	config.SetUserAuthState(core.UserAuthState{})
	return config
}

func webHasCredentials() bool {
	return webFeishuConfig.AppId != "" && webFeishuConfig.AppSecret != ""
}

func readWebAuthType() string {
	value := os.Getenv("FEISHU_AUTH_TYPE")
	if value == "" {
		return core.AuthTypeApp
	}
	return value
}

func readWebRedirectURL() string {
	value := os.Getenv("FEISHU_OAUTH_REDIRECT_URI")
	if value == "" {
		return defaultWebRedirectURL
	}
	return value
}

func webCallbackPath() string {
	redirectURL, err := url.Parse(webFeishuConfig.OAuthRedirectURL)
	if err != nil || redirectURL.Path == "" {
		return "/auth/callback"
	}
	return redirectURL.Path
}

func loginPageData(c *gin.Context) gin.H {
	data := gin.H{
		"AuthType":      webFeishuConfig.AuthType,
		"Authenticated": false,
		"UserName":      "",
		"NeedsConfig":   !webHasCredentials(),
	}
	if data["NeedsConfig"].(bool) {
		return data
	}
	sessionID, ok := readSessionID(c)
	if !ok {
		return data
	}
	session, ok := webSessions.get(sessionID)
	if !ok || session.UserAuth.AccessToken == "" {
		return data
	}
	data["Authenticated"] = true
	data["UserName"] = session.UserAuth.Name
	return data
}

func authLoginHandler(c *gin.Context) {
	if !webHasCredentials() {
		c.String(http.StatusBadRequest, "missing app_id/app_secret, please configure first")
		return
	}
	if webFeishuConfig.AuthType != core.AuthTypeUser {
		c.Redirect(http.StatusFound, "/")
		return
	}
	sessionID := ensureSessionID(c)
	state, err := core.GenerateOAuthState()
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}
	nextURL := c.DefaultQuery("next", "/")
	webSessions.update(sessionID, func(session *sessionData) {
		session.ExpectedState = state
		session.NextURL = nextURL
	})
	c.Redirect(http.StatusFound, webOAuth.AuthorizeURL(webFeishuConfig.OAuthRedirectURL, state))
}

func authCallbackHandler(c *gin.Context) {
	sessionID, ok := readSessionID(c)
	if !ok {
		c.String(http.StatusBadRequest, "missing session")
		return
	}
	session, ok := webSessions.get(sessionID)
	if !ok || session.ExpectedState == "" {
		c.String(http.StatusBadRequest, "oauth session not found")
		return
	}
	if err := validateWebCallback(c, session.ExpectedState); err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return
	}
	state, err := webOAuth.ExchangeCode(c.Request.Context(), c.Query("code"))
	if err != nil {
		c.String(http.StatusBadGateway, err.Error())
		return
	}
	webSessions.update(sessionID, func(current *sessionData) {
		current.ExpectedState = ""
		current.UserAuth = state
	})
	c.Redirect(http.StatusFound, session.NextURL)
}

func authLogoutHandler(c *gin.Context) {
	sessionID, ok := readSessionID(c)
	if ok {
		webSessions.delete(sessionID)
	}
	c.SetCookie(sessionCookieName, "", -1, "/", "", false, true)
	c.Redirect(http.StatusFound, "/")
}

func newWebClient(c *gin.Context) (*core.Client, error) {
	if !webHasCredentials() {
		return nil, fmt.Errorf("missing app_id/app_secret, please configure first")
	}
	if webFeishuConfig.AuthType != core.AuthTypeUser {
		return core.NewClient(webFeishuConfig, nil), nil
	}
	sessionID, ok := readSessionID(c)
	if !ok {
		return nil, core.ErrUserLoginRequired
	}
	session, ok := webSessions.get(sessionID)
	if !ok || session.UserAuth.RefreshToken == "" {
		return nil, core.ErrUserLoginRequired
	}
	config := webFeishuConfig
	config.SetUserAuthState(session.UserAuth)
	provider, err := core.NewRefreshingUserTokenProvider(
		config,
		func(state core.UserAuthState) error {
			webConfigMu.Lock()
			defer webConfigMu.Unlock()
			webSessions.update(sessionID, func(current *sessionData) {
				current.UserAuth = state
			})
			return nil
		},
	)
	if err != nil {
		return nil, err
	}
	return core.NewClient(webFeishuConfig, provider), nil
}

func validateWebCallback(c *gin.Context, expectedState string) error {
	if c.Query("state") != expectedState {
		return errors.New("invalid oauth state")
	}
	if errorText := c.Query("error"); errorText != "" {
		return errors.New(errorText)
	}
	if c.Query("code") == "" {
		return errors.New("missing oauth code")
	}
	return nil
}

func requireWebLogin(c *gin.Context) bool {
	if !webHasCredentials() {
		c.String(http.StatusBadRequest, "missing app_id/app_secret, please configure first")
		return true
	}
	if webFeishuConfig.AuthType != core.AuthTypeUser {
		return false
	}
	_, err := newWebClient(c)
	if err == nil {
		return false
	}
	if !errors.Is(err, core.ErrUserLoginRequired) {
		c.String(http.StatusInternalServerError, err.Error())
		return true
	}
	next := url.QueryEscape(c.Request.URL.RequestURI())
	c.Redirect(http.StatusFound, "/auth/login?next="+next)
	return true
}

type setupRequest struct {
	AppID       string `json:"app_id"`
	AppSecret   string `json:"app_secret"`
	AuthType    string `json:"auth_type"`
	RedirectURI string `json:"redirect_uri"`
}

func authBootstrapHandler(c *gin.Context) {
	webConfigMu.Lock()
	defer webConfigMu.Unlock()
	if webHasCredentials() {
		c.String(http.StatusForbidden, "app credentials already configured")
		return
	}
	var req setupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return
	}
	config := webFeishuConfig
	config.AppId = req.AppID
	config.AppSecret = req.AppSecret
	if req.AuthType != "" {
		config.AuthType = req.AuthType
	}
	if req.RedirectURI != "" {
		config.OAuthRedirectURL = req.RedirectURI
	}
	if err := config.Validate(); err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return
	}
	webFeishuConfig = config
	webOAuth = core.NewOAuthService(webFeishuConfig)
	if err := persistWebConfig(); err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}
	c.Status(http.StatusNoContent)
}

func ensureSessionID(c *gin.Context) string {
	if value, ok := readSessionID(c); ok {
		return value
	}
	sessionID, _ := core.GenerateOAuthState()
	c.SetCookie(sessionCookieName, sessionID, int((24 * time.Hour).Seconds()), "/", "", false, true)
	webSessions.update(sessionID, func(session *sessionData) {})
	return sessionID
}

func readSessionID(c *gin.Context) (string, bool) {
	value, err := c.Cookie(sessionCookieName)
	if err != nil || value == "" {
		return "", false
	}
	return value, true
}

func (s *sessionStore) get(sessionID string) (*sessionData, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	session, ok := s.sessions[sessionID]
	if !ok {
		return nil, false
	}
	copy := *session
	return &copy, true
}

func (s *sessionStore) update(sessionID string, update func(*sessionData)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	session, ok := s.sessions[sessionID]
	if !ok {
		session = &sessionData{}
		s.sessions[sessionID] = session
	}
	update(session)
}

func (s *sessionStore) delete(sessionID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, sessionID)
}
