package main

import (
	"errors"
	"net/http"
	"net/url"
	"os"
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
)

func initWebAuth() error {
	webFeishuConfig = core.FeishuConfig{
		AppId:            os.Getenv("FEISHU_APP_ID"),
		AppSecret:        os.Getenv("FEISHU_APP_SECRET"),
		AuthType:         readWebAuthType(),
		OAuthRedirectURL: readWebRedirectURL(),
	}
	if err := webFeishuConfig.Validate(); err != nil {
		return err
	}
	webOAuth = core.NewOAuthService(webFeishuConfig)
	return nil
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
