package user_mgt

import (
	"WugongMeta/cmn"
	"go.uber.org/zap"
)

const (
	smsCodeLength = 6 // 短信验证码长度
)

var z *zap.Logger

func Init() {
	z = cmn.GetLogger()

	cmn.MiniLogger.Info("[ OK ] user_mgt module initialized")
}
