package points_core

import (
	"WugongMeta/cmn"
	"context"
	"go.uber.org/zap"
	"sync"
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
