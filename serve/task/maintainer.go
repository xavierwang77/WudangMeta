package task

import (
	"WudangMeta/cmn"
	"context"
	"time"

	"go.uber.org/zap"
)

// 用于每日零点刷新所有用户运势
func fortuneRefresher(ctx context.Context) {
	for {
		// 计算距离下一次 00:00 的时间
		duration, err := cmn.GetDurationUntilNextTargetTime(0, 0, 0, "Asia/Shanghai")
		if err != nil {
			z.Error("failed to get duration until next target time", zap.Error(err))
			return
		}
		z.Info("luck-tendency-refresher sleep until next target time", zap.Duration("duration", duration))

		timer := time.NewTimer(duration)

		select {
		case <-ctx.Done():
			z.Info("luck-tendency-refresher stopped")
			timer.Stop()
			return
		case <-timer.C:
			// 每天 00:00 刷新一次
			err = RefreshAllUsersFortune(ctx)
			if err != nil {
				z.Error("failed to refresh all users' luck tendency ", zap.Error(err))
				continue
			}
		}
	}
}
