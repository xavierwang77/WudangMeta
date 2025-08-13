package ubanquan_core

import (
	"WudangMeta/cmn"

	"github.com/spf13/viper"
	"go.uber.org/zap"
)

const (
	AssetPlatform = "ubanquan" // 资产平台标识
)

var (
	AppId     string
	AppSecret string
)

var z *zap.Logger

func Init() {
	z = cmn.GetLogger()

	AppId = viper.GetString("ubanquan.appId")
	if AppId == "" {
		z.Fatal("[ FAIL ] ubanquan.appId is empty")
	}
	AppSecret = viper.GetString("ubanquan.appSecret")
	if AppSecret == "" {
		z.Fatal("[ FAIL ] ubanquan.appSecret is empty")
	}

	cmn.MiniLogger.Info("[ OK ] ubanquan-core module initialized")
}
