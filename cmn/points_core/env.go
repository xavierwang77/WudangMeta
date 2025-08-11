package points_core

import (
	"WudangMeta/cmn"
	"context"
	"sync"

	"go.uber.org/zap"
)

var once sync.Once

var z *zap.Logger

func Init() {
	z = cmn.GetLogger()

	ctx := context.Background()

	once.Do(func() {
		go userAssetPointsMaintainer(ctx, cmn.GormDB)
	})

	cmn.MiniLogger.Info("[ OK ] points-core module initialized")
}
