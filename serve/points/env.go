package points

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

	cmn.MiniLogger.Info("[ OK ] points module initialized")
}
