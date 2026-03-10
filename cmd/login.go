package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/Wsine/feishu2md/core"
)

const defaultCLIRedirectURL = "http://127.0.0.1:38080/auth/callback"

type LoginOpts struct {
	timeout time.Duration
}

var loginOpts = LoginOpts{
	timeout: 5 * time.Minute,
}

func handleLoginCommand() error {
	configPath, err := core.GetConfigFilePath()
	if err != nil {
		return err
	}
	config, err := core.ReadConfigFromFile(configPath)
	if err != nil {
		return err
	}
	redirectURL := config.Feishu.RedirectURL(defaultCLIRedirectURL)
	serverURL, listenAddr, callbackPath, err := callbackServerConfig(redirectURL)
	if err != nil {
		return err
	}
	state, err := core.GenerateOAuthState()
	if err != nil {
		return err
	}
	oauth := core.NewOAuthService(config.Feishu)
	loginURL := oauth.AuthorizeURL(serverURL.String(), state)
	resultCh := make(chan core.UserAuthState, 1)
	errCh := make(chan error, 1)

	server := &http.Server{
		Addr:    listenAddr,
		Handler: loginHandler(callbackPath, state, oauth, resultCh, errCh),
	}
	go runLoginServer(server, errCh)
	defer shutdownServer(server)

	fmt.Println("Open this URL in your browser and complete Feishu login:")
	fmt.Println(loginURL)

	ctx, cancel := context.WithTimeout(context.Background(), loginOpts.timeout)
	defer cancel()
	select {
	case result := <-resultCh:
		config.Feishu.SetUserAuthState(result)
		config.Feishu.AuthType = core.AuthTypeUser
		if err = config.WriteConfig2File(configPath); err != nil {
			return err
		}
		fmt.Printf("Login succeeded for %s (%s)\n", result.Name, result.UserID)
		return nil
	case err = <-errCh:
		return err
	case <-ctx.Done():
		return fmt.Errorf("login timed out after %s", loginOpts.timeout)
	}
}

func loginHandler(
	callbackPath string,
	expectedState string,
	oauth *core.OAuthService,
	resultCh chan<- core.UserAuthState,
	errCh chan<- error,
) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc(callbackPath, func(w http.ResponseWriter, r *http.Request) {
		if err := handleLoginCallback(r, expectedState); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			selectSendError(errCh, err)
			return
		}
		state, err := oauth.ExchangeCode(r.Context(), r.URL.Query().Get("code"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			selectSendError(errCh, err)
			return
		}
		_, _ = w.Write([]byte(successHTML))
		selectSendResult(resultCh, state)
	})
	return mux
}

func handleLoginCallback(r *http.Request, expectedState string) error {
	query := r.URL.Query()
	if query.Get("state") != expectedState {
		return errors.New("invalid oauth state")
	}
	if errorText := query.Get("error"); errorText != "" {
		return fmt.Errorf("oauth denied: %s", errorText)
	}
	if query.Get("code") == "" {
		return errors.New("missing oauth code")
	}
	return nil
}

func callbackServerConfig(raw string) (*url.URL, string, string, error) {
	redirectURL, err := url.Parse(raw)
	if err != nil {
		return nil, "", "", err
	}
	if redirectURL.Scheme != "http" {
		return nil, "", "", errors.New("cli redirect_uri must use http")
	}
	host := redirectURL.Hostname()
	if host == "" {
		return nil, "", "", errors.New("redirect_uri host is required")
	}
	port := redirectURL.Port()
	if port == "" {
		port = "80"
	}
	if redirectURL.Path == "" {
		redirectURL.Path = "/auth/callback"
	}
	return redirectURL, net.JoinHostPort(host, port), redirectURL.Path, nil
}

func runLoginServer(server *http.Server, errCh chan<- error) {
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		selectSendError(errCh, err)
	}
}

func shutdownServer(server *http.Server) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_ = server.Shutdown(ctx)
}

func selectSendError(errCh chan<- error, err error) {
	select {
	case errCh <- err:
	default:
	}
}

func selectSendResult(resultCh chan<- core.UserAuthState, result core.UserAuthState) {
	select {
	case resultCh <- result:
	default:
	}
}

const successHTML = `
<!DOCTYPE html>
<html lang="zh-CN">
<head><meta charset="UTF-8"><title>feishu2md 登录完成</title></head>
<body>
<h2>登录成功</h2>
<p>你可以回到终端继续使用 feishu2md。</p>
</body>
</html>
`
