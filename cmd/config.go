package main

import (
	"fmt"
	"os"

	"github.com/Wsine/feishu2md/core"
	"github.com/Wsine/feishu2md/utils"
)

type ConfigOpts struct {
	appId       string
	appSecret   string
	authType    string
	redirectURI string
}

var configOpts = ConfigOpts{}

func handleConfigCommand() error {
	configPath, err := core.GetConfigFilePath()
	if err != nil {
		return err
	}

	fmt.Println("Configuration file on: " + configPath)
	config, err := loadOrCreateConfig(configPath)
	if err != nil {
		return err
	}
	updateConfig(config)
	if err = config.Feishu.Validate(); err != nil {
		return err
	}
	if err = config.WriteConfig2File(configPath); err != nil {
		return err
	}
	fmt.Println(utils.PrettyPrint(redactedConfig(config)))
	return nil
}

func loadOrCreateConfig(configPath string) (*core.Config, error) {
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return core.NewConfig(configOpts.appId, configOpts.appSecret), nil
	}
	return core.ReadConfigFromFile(configPath)
}

func updateConfig(config *core.Config) {
	if configOpts.appId != "" {
		config.Feishu.AppId = configOpts.appId
	}
	if configOpts.appSecret != "" {
		config.Feishu.AppSecret = configOpts.appSecret
	}
	if configOpts.authType != "" {
		config.Feishu.AuthType = configOpts.authType
	}
	if configOpts.redirectURI != "" {
		config.Feishu.OAuthRedirectURL = configOpts.redirectURI
	}
}

func redactedConfig(config *core.Config) *core.Config {
	copy := *config
	copy.Feishu.AppSecret = maskSecret(copy.Feishu.AppSecret)
	copy.Feishu.UserAccessToken = maskSecret(copy.Feishu.UserAccessToken)
	copy.Feishu.UserRefreshToken = maskSecret(copy.Feishu.UserRefreshToken)
	return &copy
}

func maskSecret(value string) string {
	if len(value) <= 8 {
		if value == "" {
			return ""
		}
		return "********"
	}
	return value[:4] + "..." + value[len(value)-4:]
}
