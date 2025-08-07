package ubanquan

import (
	"WugongMeta/cmn"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

const (
	assetPlatform = "ubanquan" // 资产平台标识
)

var (
	appId     string
	appSecret string
)

var z *zap.Logger

func Init() {
	z = cmn.GetLogger()

	appId = viper.GetString("ubanquan.appId")
	if appId == "" {
		z.Fatal("[ FAIL ] ubanquan.appId is empty")
	}
	appSecret = viper.GetString("ubanquan.appSecret")
	if appSecret == "" {
		z.Fatal("[ FAIL ] ubanquan.appSecret is empty")
	}

	cmn.MiniLogger.Info("[ OK ] ubanquan module initialized")
}
