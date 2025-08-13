package ubanquan_core

import (
	"WudangMeta/cmn"
	"context"

	"github.com/spf13/viper"
	"go.uber.org/zap"
)

const (
	AssetPlatform = "ubanquan" // 资产平台标识
)

var (
	AppId      string
	AppSecret  string
	BaseApiUrl string
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
	BaseApiUrl = viper.GetString("ubanquan.baseApiUrl")
	if BaseApiUrl == "" {
		z.Fatal("[ FAIL ] ubanquan.baseApiUrl is empty")
	}

	// 初始化token
	ctx := context.Background()
	err := InitializeToken(ctx, AppId, AppSecret)
	if err != nil {
		z.Fatal("[ FAIL ] failed to initialize ubanquan token", zap.Error(err))
	}

	// 启动token维护协程
	StartTokenMaintainer(ctx)

	cmn.MiniLogger.Info("[ OK ] ubanquan-core module initialized",
		zap.String("appId", AppId),
		zap.String("appSecret", AppSecret[:10]+"...)"),
		zap.String("accessToken", GetGlobalToken().AccessToken[:10]+"...)"),
		zap.Int64("expiresTime", GetGlobalToken().ExpiresTime))
}
