package task

import (
	"WudangMeta/cmn"
	"fmt"

	"github.com/spf13/viper"
	"go.uber.org/zap"
)

var z *zap.Logger

var (
	dailyCheckInScore float64 // 每日签到积分
	luckTendencyScore float64 // 运势分析积分
	llmPrompt         LlmPrompt
)

func Init() {
	z = cmn.GetLogger()

	enable := viper.GetBool("task.enable")
	if !enable {
		cmn.MiniLogger.Info("[ -- ] task module disabled")
		return
	}

	// 初始化每日签到积分
	dailyCheckInScore = viper.GetFloat64("task.reward.dailyCheckInPoints")
	if dailyCheckInScore <= 0 {
		z.Fatal("[ FAIL ] daily check in points must be greater than 0")
	}
	// 初始化运势分析积分
	luckTendencyScore = viper.GetFloat64("task.reward.luckTendencyPoints")
	if luckTendencyScore <= 0 {
		z.Fatal("[ FAIL ] luck tendency points must be greater than 0")
	}

	// 初始化大模型提示词
	err := initLlmPrompt()
	if err != nil {
		z.Fatal("[ FAIL ] failed to init llmPrompt", zap.Error(err))
	}

	//ctx := context.Background()
	//go fortuneRefresher(ctx)

	cmn.MiniLogger.Info("[ OK ] task module initialed", zap.Float64("dailyCheckInScore", dailyCheckInScore))
}

func initLlmPrompt() error {
	subTree := viper.Sub("task.llmPrompt")
	if subTree == nil {
		z.Error("task.llmPrompt subTree is nil")
		return fmt.Errorf("task.llmPrompt subTree is nil")
	}

	if err := subTree.Unmarshal(&llmPrompt); err != nil {
		z.Error("failed to unmarshal llmPrompt config", zap.Error(err))
		return err
	}

	if llmPrompt.Prompt == "" {
		z.Error("llmPrompt.Prompt is empty")
		return fmt.Errorf("llmPrompt.Prompt is empty")
	}

	return nil
}
