package ubanquan

import (
	"WudangMeta/cmn"

	"go.uber.org/zap"
)

var z *zap.Logger

func Init() {
	z = cmn.GetLogger()

	cmn.MiniLogger.Info("[ OK ] ubanquan module initialized")
}
