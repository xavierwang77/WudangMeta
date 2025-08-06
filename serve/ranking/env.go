package ranking

import (
	"WugongMeta/cmn"
	"go.uber.org/zap"
)

var z *zap.Logger

func Init() {
	z = cmn.GetLogger()

	cmn.MiniLogger.Info("[ OK ] ranking module initialized")
}
