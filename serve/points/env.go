package points

import (
	"WugongMeta/cmn"
	"go.uber.org/zap"
)

var z *zap.Logger

func Init() {
	z = cmn.GetLogger()

	cmn.MiniLogger.Info("[ OK ] points module initialized")
}
