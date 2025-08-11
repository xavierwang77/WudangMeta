package raffle

import (
	"WudangMeta/cmn"
	"strconv"
	"sync"

	"go.uber.org/zap"
)

const (
	noPrizeSign = "未中奖"

	cfgKeyConsumePointsKey   = "raffle.consumePointsKey"   // 抽奖消耗积分键的配置键
	cfgKeyConsumePointsValue = "raffle.consumePointsValue" // 抽奖消耗积分值的配置键
)

var (
	machine *Machine  // 抽奖机实例
	once    sync.Once // 确保只初始化一次
	z       *zap.Logger
)

func Init() {
	z = cmn.GetLogger()

	// 从配置表读取抽奖消耗积分键
	pointsKey, err := cmn.GetConfigFromDB(cfgKeyConsumePointsKey, "default_points")
	if err != nil {
		z.Fatal("[ FAIL ] failed to get consume points key from config table", zap.Error(err))
	}
	if pointsKey == "" {
		z.Warn("[ WARN ] raffle consume points key is empty, using default 'default_points'")
		pointsKey = "default_points"
	}

	// 从配置表读取抽奖消耗积分值
	consumePointsValueStr, err := cmn.GetConfigFromDB(cfgKeyConsumePointsValue, "100")
	if err != nil {
		z.Fatal("[ FAIL ] failed to get consume points from config table", zap.Error(err))
	}
	pointsValue, err := strconv.ParseInt(consumePointsValueStr, 10, 64)
	if err != nil {
		z.Fatal("[ FAIL ] invalid consume points value in config table", zap.String("value", consumePointsValueStr), zap.Error(err))
	}
	if pointsValue < 0 {
		z.Fatal("[ FAIL ] raffle consume points must be greater than or equal to 0")
	}

	once.Do(func() {
		var err error
		machine, err = NewMachine(pointsKey, pointsValue)
		if err != nil {
			z.Fatal("[ FAIL ] failed to create raffle machine", zap.Error(err))
		}
	})

	cmn.MiniLogger.Info("[ OK ] raffle module initialized", zap.String("consumePointsKey", machine.consumePointsKey), zap.Int64("consumePointsValue", pointsValue))
}
