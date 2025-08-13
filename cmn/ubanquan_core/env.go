package ubanquan_core

import (
	"WudangMeta/cmn"

	"go.uber.org/zap"
)

const (
	assetPlatform = "ubanquan" // 资产平台标识
)

var z *zap.Logger

func Init() {
	z = cmn.GetLogger()
}
