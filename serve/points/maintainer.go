package points

import (
	"WugongMeta/cmn"
	"context"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"time"
)

func userAssetPointsMaintainer(ctx context.Context, db *gorm.DB) {
	for {
		// 计算距离下一次 00:00 的时间
		duration, err := cmn.GetDurationUntilNextTargetTime(0, 0, 0, "Asia/Shanghai")
		if err != nil {
			z.Error("failed to get duration until next target time", zap.Error(err))
			return
		}
		z.Info("userAssetPointsMaintainer sleep until next target time", zap.Duration("duration", duration))

		timer := time.NewTimer(duration)

		select {
		case <-ctx.Done():
			z.Info("userAssetPointsMaintainer stopped")
			timer.Stop()
			return
		case <-timer.C:
			// 每天 00:00 更新一次用户积分
			go func() {
				errs := AddAllUserPointsFromAssets(ctx, db)
				if len(errs) > 0 {
					z.Error("failed to add all user points from assets", zap.Any("errors", errs))
				}
			}()
		}
	}
}
