package ranking

import (
	"WugongMeta/cmn"
	"context"
	"fmt"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// QueryAssetRankingList 查询资产排行榜列表
// 根据用户资产总值进行排名，支持分页和资产类型过滤
func QueryAssetRankingList(ctx context.Context, page, pageSize int64, filterMetaAsset []int64) ([]AssetRankingList, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}

	// 构建基础子查询，统计用户资产
	subQuery := cmn.GormDB.Model(&cmn.VUserAssetMeta{}).
		Select("user_id, mobile_phone, COUNT(*) as asset_count, SUM(meta_asset_value) as asset_value").
		Group("user_id, mobile_phone")

	// 如果有资产类型过滤条件，则添加过滤
	if len(filterMetaAsset) > 0 {
		subQuery = subQuery.Where("meta_asset_id IN ?", filterMetaAsset)
	}

	// 先查询总记录数
	var totalCount int64
	err := cmn.GormDB.Table("(?) as sub", subQuery).Count(&totalCount).Error
	if err != nil {
		z.Error("failed to query total count", zap.Error(err))
		return nil, 0, fmt.Errorf("failed to query total count: %w", err)
	}

	// 如果没有数据，返回空切片
	if totalCount == 0 {
		return []AssetRankingList{}, 0, nil
	}

	// 构建带排名的查询，使用窗口函数在数据库层面计算排名
	rankedQuery := cmn.GormDB.Table("(?) as ranked_data", subQuery).
		Select("user_id, mobile_phone, asset_count, asset_value, ROW_NUMBER() OVER (ORDER BY asset_value DESC) as ranking").
		Limit(int(pageSize)).
		Offset(int((page - 1) * pageSize))

	// 查询分页排行榜数据
	var rankingResults []struct {
		UserId      uuid.UUID `gorm:"column:user_id"`
		MobilePhone string    `gorm:"column:mobile_phone"`
		AssetCount  int64     `gorm:"column:asset_count"`
		AssetValue  float64   `gorm:"column:asset_value"`
		Ranking     int64     `gorm:"column:ranking"`
	}

	err = rankedQuery.Scan(&rankingResults).Error
	if err != nil {
		z.Error("failed to query ranking data", zap.Error(err))
		return nil, 0, fmt.Errorf("failed to query ranking data: %w", err)
	}

	// 构建返回结果
	rankingList := make([]AssetRankingList, len(rankingResults))
	for i, result := range rankingResults {
		// 计算实际排名（考虑分页偏移）
		actualRanking := (page-1)*pageSize + int64(i) + 1
		rankingList[i] = AssetRankingList{
			UserId:      result.UserId,
			MobilePhone: result.MobilePhone,
			AssetCount:  result.AssetCount,
			AssetValue:  result.AssetValue,
			Ranking:     actualRanking,
		}
	}

	return rankingList, totalCount, nil
}
