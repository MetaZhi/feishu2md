package core

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"time"
)

const (
	AuthTypeApp  = "app"
	AuthTypeUser = "user"
)

type Config struct {
	Feishu FeishuConfig `json:"feishu"`
	Output OutputConfig `json:"output"`
}

type FeishuConfig struct {
	AppId                  string `json:"app_id"`
	AppSecret              string `json:"app_secret"`
	AuthType               string `json:"auth_type,omitempty"`
	OAuthRedirectURL       string `json:"oauth_redirect_url,omitempty"`
	UserAccessToken        string `json:"user_access_token,omitempty"`
	UserRefreshToken       string `json:"user_refresh_token,omitempty"`
	UserAccessTokenExpiry  string `json:"user_access_token_expiry,omitempty"`
	UserRefreshTokenExpiry string `json:"user_refresh_token_expiry,omitempty"`
	UserID                 string `json:"user_id,omitempty"`
	UserOpenID             string `json:"user_open_id,omitempty"`
	UserName               string `json:"user_name,omitempty"`
}

type OutputConfig struct {
	ImageDir        string `json:"image_dir"`
	TitleAsFilename bool   `json:"title_as_filename"`
	UseHTMLTags     bool   `json:"use_html_tags"`
	SkipImgDownload bool   `json:"skip_img_download"`
}

func NewConfig(appId, appSecret string) *Config {
	return &Config{
		Feishu: FeishuConfig{
			AppId:     appId,
			AppSecret: appSecret,
			AuthType:  AuthTypeApp,
		},
		Output: OutputConfig{
			ImageDir:        "static",
			TitleAsFilename: false,
			UseHTMLTags:     false,
			SkipImgDownload: false,
		},
	}
}

func (fc *FeishuConfig) Validate() error {
	if fc.AuthType == "" {
		fc.AuthType = AuthTypeApp
	}
	if fc.AuthType != AuthTypeApp && fc.AuthType != AuthTypeUser {
		return fmt.Errorf("invalid auth_type %q", fc.AuthType)
	}
	if fc.AppId == "" || fc.AppSecret == "" {
		return errors.New("app_id and app_secret are required")
	}
	if _, err := fc.UserAuthState(); err != nil {
		return err
	}
	return nil
}

func (fc FeishuConfig) RedirectURL(fallback string) string {
	if fc.OAuthRedirectURL != "" {
		return fc.OAuthRedirectURL
	}
	return fallback
}

func (fc FeishuConfig) UserAuthState() (UserAuthState, error) {
	accessExpiry, err := parseOptionalTime(fc.UserAccessTokenExpiry)
	if err != nil {
		return UserAuthState{}, fmt.Errorf("invalid user_access_token_expiry: %w", err)
	}
	refreshExpiry, err := parseOptionalTime(fc.UserRefreshTokenExpiry)
	if err != nil {
		return UserAuthState{}, fmt.Errorf("invalid user_refresh_token_expiry: %w", err)
	}
	return UserAuthState{
		AccessToken:        fc.UserAccessToken,
		RefreshToken:       fc.UserRefreshToken,
		AccessTokenExpiry:  accessExpiry,
		RefreshTokenExpiry: refreshExpiry,
		UserID:             fc.UserID,
		OpenID:             fc.UserOpenID,
		Name:               fc.UserName,
	}, nil
}

func (fc *FeishuConfig) SetUserAuthState(state UserAuthState) {
	fc.UserAccessToken = state.AccessToken
	fc.UserRefreshToken = state.RefreshToken
	fc.UserAccessTokenExpiry = formatOptionalTime(state.AccessTokenExpiry)
	fc.UserRefreshTokenExpiry = formatOptionalTime(state.RefreshTokenExpiry)
	fc.UserID = state.UserID
	fc.UserOpenID = state.OpenID
	fc.UserName = state.Name
}

func (fc FeishuConfig) HasUserSession() bool {
	return fc.UserAccessToken != "" || fc.UserRefreshToken != ""
}

func GetConfigFilePath() (string, error) {
	configPath, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	configFilePath := path.Join(configPath, "feishu2md", "config.json")
	return configFilePath, nil
}

func ReadConfigFromFile(configPath string) (*Config, error) {
	file, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}
	config := NewConfig("", "")
	if err = json.Unmarshal(file, &config); err != nil {
		return nil, err
	}
	if err = config.Feishu.Validate(); err != nil {
		return nil, err
	}
	return config, nil
}

func (conf *Config) WriteConfig2File(configPath string) error {
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		return err
	}
	file, err := json.MarshalIndent(conf, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(configPath, file, 0o600)
}

func parseOptionalTime(raw string) (time.Time, error) {
	if raw == "" {
		return time.Time{}, nil
	}
	return time.Parse(time.RFC3339, raw)
}

func formatOptionalTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
}
