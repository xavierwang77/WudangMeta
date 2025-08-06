package ranking

import "github.com/google/uuid"

// AssetRankingList 资产排行榜列表项
type AssetRankingList struct {
	UserId      uuid.UUID `json:"userId"`      // 用户ID
	MobilePhone string    `json:"mobilePhone"` // 手机号
	AssetCount  int64     `json:"assetCount"`  // 资产数量
	AssetValue  float64   `json:"assetValue"`  // 资产总值
	Ranking     int64     `json:"ranking"`     // 排名
}
