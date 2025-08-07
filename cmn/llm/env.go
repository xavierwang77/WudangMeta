package llm

import (
	"WugongMeta/cmn"
	"fmt"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

var (
	logger   *zap.Logger
	enable   bool
	platform string

	deepSeekConfig DeepSeekConfig
)

func Init() {
	logger = cmn.GetLogger()

	enable = viper.GetBool("llm.enable")
	if !enable {
		cmn.MiniLogger.Info("[ -- ] llm module disabled")
		return
	}

	platform = viper.GetString("llm.platform")
	if platform == "" {
		logger.Fatal("[ FAIL ] llm platform not set")
	}

	switch platform {
	case "deepseek":
		err := initDeepSeek()
		if err != nil {
			logger.Fatal("[ FAIL ] failed to init deepseek", zap.Error(err))
		}
	}

	cmn.MiniLogger.Info("[ OK ] llm module initialed", zap.String("platform", platform))
}

func initDeepSeek() error {
	deepSeekConfig.ApiKey = viper.GetString("llm.data.apiKey")
	if deepSeekConfig.ApiKey == "" {
		logger.Error("api key not set")
		return fmt.Errorf("llm module api key not set")
	}

	deepSeekConfig.Model = viper.GetString("llm.data.model")
	if deepSeekConfig.Model == "" {
		logger.Error("model not set")
		return fmt.Errorf("llm module model not set")
	}

	deepSeekConfig.BaseUrl = viper.GetString("llm.data.baseUrl")
	if deepSeekConfig.BaseUrl == "" {
		logger.Error("base url not set")
		return fmt.Errorf("llm module base url not set")
	}

	return nil
}
