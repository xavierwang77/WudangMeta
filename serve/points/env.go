package points

import (
	"WudangMeta/cmn"
	"context"
	"sync"

	"go.uber.org/zap"
)

var z *zap.Logger

var once sync.Once

func Init() {
	z = cmn.GetLogger()

	ctx := context.Background()

	once.Do(func() {
		go userAssetPointsMaintainer(ctx, cmn.GormDB)
	})

	cmn.MiniLogger.Info("[ OK ] points module initialized")
}
