package raffle

import (
	"WudangMeta/cmn"
	"sync"

	"github.com/spf13/viper"
	"go.uber.org/zap"
)

const (
	noPrizeSign = "未中奖"
)

var (
	machine *Machine  // 抽奖机实例
	once    sync.Once // 确保只初始化一次
	z       *zap.Logger
)

func Init() {
	z = cmn.GetLogger()

	consumePoints := viper.GetInt("raffle.consumePoints")
	if consumePoints < 0 {
		z.Fatal("[ FAIL ] raffle consume points must be greater than or equal to 0")
	}
	consumePointsKey := viper.GetString("raffle.consumePointsKey")
	if consumePointsKey == "" {
		z.Warn("[ WARN ] raffle consume points key is empty, using default 'default_points'")
		consumePointsKey = "default_points"
	}

	once.Do(func() {
		var err error
		machine, err = NewMachine(consumePoints, consumePointsKey)
		if err != nil {
			z.Fatal("[ FAIL ] failed to create raffle machine", zap.Error(err))
		}
	})
}
