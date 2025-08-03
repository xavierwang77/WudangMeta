package sms

import (
	"WugongMeta/cmn"
	"github.com/spf13/viper"
	v20210111 "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/sms/v20210111"
	"go.uber.org/zap"
)

var (
	z        *zap.Logger
	platform string

	juheConfig JuheConfig

	tecentConfig TecentConfig
	tecentClient *v20210111.Client

	shxConfig ShxTongConfig
)

func Init() {
	z = cmn.GetLogger()

	// 若果没有开启短信服务，则不进行初始化
	enable := viper.GetBool("sms.enable")
	if !enable {
		cmn.MiniLogger.Info("[ -- ] sms module is disabled")
		return
	}

	platform = viper.GetString("sms.platform")
	switch platform {
	case "juhe":
		err := initJuheConfig()
		if err != nil {
			z.Fatal("[ FAIL ] init juhe sms config", zap.Error(err))
		}
	case "tecent":
		err := initTecentConfig()
		if err != nil {
			z.Fatal("[ FAIL ] init tecent sms config", zap.Error(err))
		}
	case "shx":
		err := initShxTongConfig()
		if err != nil {
			z.Fatal("[ FAIL ] init shx sms config", zap.Error(err))
		}
	default:
		z.Fatal("[ FAIL ] sms platform is not supported", zap.String("platform", platform))
	}

	cmn.MiniLogger.Info("[ OK ] sms module initialed", zap.String("platform", platform))
}
