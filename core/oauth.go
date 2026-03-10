package core

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const (
	defaultOAuthBaseURL   = "https://open.feishu.cn"
	tokenRefreshBuffer    = 5 * time.Minute
	defaultAuthorizePath  = "/open-apis/authen/v1/index"
	defaultAccessPath     = "/open-apis/authen/v1/access_token"
	defaultRefreshPath    = "/open-apis/authen/v1/refresh_access_token"
	defaultUserInfoPath   = "/open-apis/authen/v1/user_info"
	defaultAppTokenPath   = "/open-apis/auth/v3/app_access_token/internal"
	defaultRequestTimeout = 15 * time.Second
)

var ErrUserLoginRequired = errors.New("user login required")

type UserAuthState struct {
	AccessToken        string
	RefreshToken       string
	AccessTokenExpiry  time.Time
	RefreshTokenExpiry time.Time
	UserID             string
	OpenID             string
	Name               string
}

type UserAccessTokenProvider interface {
	UserAccessToken(ctx context.Context) (string, error)
}

type OAuthService struct {
	appID      string
	appSecret  string
	baseURL    string
	httpClient *http.Client
}

type RefreshingUserTokenProvider struct {
	mu      sync.Mutex
	oauth   *OAuthService
	state   UserAuthState
	persist func(UserAuthState) error
}

type appAccessTokenResp struct {
	Code           int    `json:"code"`
	Msg            string `json:"msg"`
	AppAccessToken string `json:"app_access_token"`
}

type userTokenEnvelope struct {
	Code int              `json:"code"`
	Msg  string           `json:"msg"`
	Data userTokenPayload `json:"data"`
}

type userInfoEnvelope struct {
	Code int             `json:"code"`
	Msg  string          `json:"msg"`
	Data userInfoPayload `json:"data"`
}

type userTokenPayload struct {
	AccessToken      string `json:"access_token"`
	ExpiresIn        int    `json:"expires_in"`
	RefreshToken     string `json:"refresh_token"`
	RefreshExpiresIn int    `json:"refresh_expires_in"`
	UserID           string `json:"user_id"`
	OpenID           string `json:"open_id"`
	Name             string `json:"name"`
}

type userInfoPayload struct {
	UserID string `json:"user_id"`
	OpenID string `json:"open_id"`
	Name   string `json:"name"`
}

func NewOAuthService(config FeishuConfig) *OAuthService {
	return &OAuthService{
		appID:      config.AppId,
		appSecret:  config.AppSecret,
		baseURL:    defaultOAuthBaseURL,
		httpClient: &http.Client{Timeout: defaultRequestTimeout},
	}
}

func NewRefreshingUserTokenProvider(
	config FeishuConfig,
	persist func(UserAuthState) error,
) (*RefreshingUserTokenProvider, error) {
	state, err := config.UserAuthState()
	if err != nil {
		return nil, err
	}
	return &RefreshingUserTokenProvider{
		oauth:   NewOAuthService(config),
		state:   state,
		persist: persist,
	}, nil
}

func (s *OAuthService) AuthorizeURL(redirectURL, state string) string {
	values := url.Values{}
	values.Set("app_id", s.appID)
	values.Set("redirect_uri", redirectURL)
	values.Set("state", state)
	return s.baseURL + defaultAuthorizePath + "?" + values.Encode()
}

func (s *OAuthService) ExchangeCode(
	ctx context.Context,
	code string,
) (UserAuthState, error) {
	appToken, err := s.getAppAccessToken(ctx)
	if err != nil {
		return UserAuthState{}, err
	}
	body := map[string]string{
		"grant_type": "authorization_code",
		"code":       code,
	}
	var envelope userTokenEnvelope
	if err = s.postJSON(ctx, defaultAccessPath, body, appToken, &envelope); err != nil {
		return UserAuthState{}, err
	}
	state := envelope.Data.toState(time.Now())
	return s.fillUserInfo(ctx, appToken, state)
}

func (s *OAuthService) Refresh(
	ctx context.Context,
	refreshToken string,
) (UserAuthState, error) {
	appToken, err := s.getAppAccessToken(ctx)
	if err != nil {
		return UserAuthState{}, err
	}
	body := map[string]string{
		"grant_type":    "refresh_token",
		"refresh_token": refreshToken,
	}
	var envelope userTokenEnvelope
	if err = s.postJSON(ctx, defaultRefreshPath, body, appToken, &envelope); err != nil {
		return UserAuthState{}, err
	}
	state := envelope.Data.toState(time.Now())
	if state.Name != "" || state.UserID != "" || state.OpenID != "" {
		return state, nil
	}
	return s.fillUserInfo(ctx, appToken, state)
}

func (p *RefreshingUserTokenProvider) UserAccessToken(
	ctx context.Context,
) (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if token := p.currentAccessToken(); token != "" {
		return token, nil
	}
	if p.state.RefreshToken == "" || p.state.RefreshTokenExpired(time.Now()) {
		return "", ErrUserLoginRequired
	}
	refreshed, err := p.oauth.Refresh(ctx, p.state.RefreshToken)
	if err != nil {
		return "", err
	}
	p.state = refreshed
	if p.persist != nil {
		if err = p.persist(refreshed); err != nil {
			return "", err
		}
	}
	return refreshed.AccessToken, nil
}

func (p *RefreshingUserTokenProvider) CurrentState() UserAuthState {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.state
}

func GenerateOAuthState() (string, error) {
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func (s *OAuthService) getAppAccessToken(ctx context.Context) (string, error) {
	body := map[string]string{
		"app_id":     s.appID,
		"app_secret": s.appSecret,
	}
	var resp appAccessTokenResp
	if err := s.postJSON(ctx, defaultAppTokenPath, body, "", &resp); err != nil {
		return "", err
	}
	if resp.AppAccessToken == "" {
		return "", errors.New("empty app_access_token")
	}
	return resp.AppAccessToken, nil
}

func (s *OAuthService) fillUserInfo(
	ctx context.Context,
	appToken string,
	state UserAuthState,
) (UserAuthState, error) {
	if state.UserID != "" && state.OpenID != "" && state.Name != "" {
		return state, nil
	}
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		s.baseURL+defaultUserInfoPath,
		nil,
	)
	if err != nil {
		return UserAuthState{}, err
	}
	req.Header.Set("Authorization", "Bearer "+appToken)
	req.Header.Set("User-Access-Token", state.AccessToken)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return UserAuthState{}, err
	}
	defer resp.Body.Close()

	var envelope userInfoEnvelope
	if err = decodeJSONResponse(resp, &envelope); err != nil {
		return UserAuthState{}, err
	}
	if state.Name == "" {
		state.Name = envelope.Data.Name
	}
	if state.UserID == "" {
		state.UserID = envelope.Data.UserID
	}
	if state.OpenID == "" {
		state.OpenID = envelope.Data.OpenID
	}
	return state, nil
}

func (s *OAuthService) postJSON(
	ctx context.Context,
	path string,
	body interface{},
	bearerToken string,
	out interface{},
) error {
	payload, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		s.baseURL+path,
		bytes.NewReader(payload),
	)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	if bearerToken != "" {
		req.Header.Set("Authorization", "Bearer "+bearerToken)
	}
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return decodeJSONResponse(resp, out)
}

func decodeJSONResponse(resp *http.Response, out interface{}) error {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode >= http.StatusBadRequest {
		return fmt.Errorf("feishu api status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	if err = json.Unmarshal(body, out); err != nil {
		return err
	}
	switch value := out.(type) {
	case *appAccessTokenResp:
		if value.Code != 0 {
			return fmt.Errorf("feishu api error %d: %s", value.Code, value.Msg)
		}
	case *userTokenEnvelope:
		if value.Code != 0 {
			return fmt.Errorf("feishu api error %d: %s", value.Code, value.Msg)
		}
	case *userInfoEnvelope:
		if value.Code != 0 {
			return fmt.Errorf("feishu api error %d: %s", value.Code, value.Msg)
		}
	}
	return nil
}

func (p userTokenPayload) toState(now time.Time) UserAuthState {
	state := UserAuthState{
		AccessToken:  p.AccessToken,
		RefreshToken: p.RefreshToken,
		UserID:       p.UserID,
		OpenID:       p.OpenID,
		Name:         p.Name,
	}
	if p.ExpiresIn > 0 {
		state.AccessTokenExpiry = now.Add(time.Duration(p.ExpiresIn) * time.Second)
	}
	if p.RefreshExpiresIn > 0 {
		state.RefreshTokenExpiry = now.Add(time.Duration(p.RefreshExpiresIn) * time.Second)
	}
	return state
}

func (s UserAuthState) AccessTokenExpired(now time.Time) bool {
	if s.AccessTokenExpiry.IsZero() {
		return s.AccessToken == ""
	}
	return !now.Before(s.AccessTokenExpiry)
}

func (s UserAuthState) RefreshTokenExpired(now time.Time) bool {
	if s.RefreshTokenExpiry.IsZero() {
		return s.RefreshToken == ""
	}
	return !now.Before(s.RefreshTokenExpiry)
}

func (p *RefreshingUserTokenProvider) currentAccessToken() string {
	now := time.Now()
	if p.state.AccessToken == "" {
		return ""
	}
	if p.state.AccessTokenExpiry.IsZero() {
		return p.state.AccessToken
	}
	if now.Add(tokenRefreshBuffer).Before(p.state.AccessTokenExpiry) {
		return p.state.AccessToken
	}
	return ""
}
