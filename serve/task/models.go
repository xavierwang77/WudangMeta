package task

import "github.com/google/uuid"

type UserData struct {
	UserId uuid.UUID `json:"userId"`
	Gender string    `json:"name" mapstructure:"gender"`
	Name   string    `json:"gender" mapstructure:"name"`
	Birth  string    `json:"birth" mapstructure:"birth"`
}

type Fortune struct {
	NftActivityAdvice map[string]string `mapstructure:"nftActivityAdvice" json:"nftActivityAdvice"` // 数藏活动建议
	FortuneAnalysis   map[string]string `mapstructure:"fortuneAnalysis" json:"fortuneAnalysis"`     // 运势分析
	FortunePercent    map[string]string `mapstructure:"fortunePercent" json:"fortunePercent"`       // 运势百分比
}

type LlmPrompt struct {
	Prompt       string   `mapstructure:"prompt"`
	UserInfo     UserData `mapstructure:"userInfo"`
	OutputStruct Fortune  `mapstructure:"outputStruct"`
}
