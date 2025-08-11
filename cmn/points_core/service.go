package points_core

import (
	"WudangMeta/cmn"
	"context"
	"fmt"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// InitializeUserPoints 根据资产初始化用户积分
// 仅为不存在积分记录的用户创建初始积分，不会更新已存在的记录
func InitializeUserPoints(ctx context.Context, db *gorm.DB, userId uuid.UUID) error {
	if db == nil {
		db = cmn.GormDB
	}
	if userId == uuid.Nil {
		e := fmt.Errorf("userId is nil")
		z.Error(e.Error())
		return e
	}

	// 检查用户积分记录是否已存在
	var existingPoints cmn.TUserPoints
	err := cmn.GormDB.Where("user_id = ?", userId).First(&existingPoints).Error
	if err == nil {
		// 记录已存在，不进行初始化
		return nil
	}

	// 查询用户所有资产及其对应的元资产信息
	var userAssets []struct {
		MetaAssetId    int64   `gorm:"column:meta_asset_id"`
		MetaAssetValue float64 `gorm:"column:meta_asset_value"`
		Count          int64   `gorm:"column:asset_count"`
	}

	// 使用 VUserAssetMeta 视图查询，统计每个元资产的数量
	err = db.Model(&cmn.VUserAssetMeta{}).
		Select("meta_asset_id, meta_asset_value, COUNT(id) as asset_count").
		Where("user_id = ?", userId).
		Group("meta_asset_id, meta_asset_value").
		Scan(&userAssets).Error

	if err != nil {
		z.Error("failed to query user assets", zap.Error(err), zap.String("user_id", userId.String()))
		return err
	}

	// 计算总积分
	var totalPoints float64
	for _, asset := range userAssets {
		totalPoints += asset.MetaAssetValue * float64(asset.Count)
	}

	// 创建用户积分记录
	userPoints := cmn.TUserPoints{
		UserId:        userId,
		DefaultPoints: totalPoints,
	}

	// 创建新的积分记录
	err = db.Create(&userPoints).Error
	if err != nil {
		z.Error("failed to create user points", zap.Error(err), zap.String("user_id", userId.String()))
		return err
	}

	z.Info("user points initialized successfully",
		zap.String("user_id", userId.String()),
		zap.Float64("initial_points", totalPoints))

	return nil
}

// AddUserPointsByAsset 根据指定的元资产ID和数量累加用户积分
// 查询元资产价值，计算积分并累加到用户现有积分上
func AddUserPointsByAsset(ctx context.Context, db *gorm.DB, userId uuid.UUID, metaAssetId int64, assetCount int64) error {
	if db == nil {
		db = cmn.GormDB
	}
	if userId == uuid.Nil {
		e := fmt.Errorf("userId is nil")
		z.Error(e.Error())
		return e
	}
	if metaAssetId <= 0 {
		e := fmt.Errorf("metaAssetId must be positive")
		z.Error(e.Error())
		return e
	}
	if assetCount <= 0 {
		e := fmt.Errorf("assetCount must be positive")
		z.Error(e.Error())
		return e
	}

	// 查询元资产价值
	var metaAsset cmn.TMetaAsset
	err := db.Where("id = ?", metaAssetId).First(&metaAsset).Error
	if err != nil {
		e := fmt.Errorf("failed to query meta asset: %w, metaAssetId: %d", err, metaAssetId)
		z.Error(e.Error())
		return e
	}

	// 计算要累加的积分
	assetPoints := metaAsset.Value * float64(assetCount)

	// 查询用户现有积分记录
	var userPoints cmn.TUserPoints
	err = db.Where("user_id = ?", userId).First(&userPoints).Error
	if err != nil {
		e := fmt.Errorf("failed to query user points: %w, userId: %s", err, userId.String())
		z.Error(e.Error())
		return e
	}

	// 累加积分到原有积分上
	newTotalPoints := userPoints.DefaultPoints + assetPoints

	// 更新积分记录
	err = db.Model(&userPoints).Update("default_points", newTotalPoints).Error
	if err != nil {
		e := fmt.Errorf("failed to update user points: %w, userId: %s", err, userId.String())
		z.Error(e.Error())
		return e
	}

	return nil
}

// AddUserPoints 增加用户积分
func AddUserPoints(ctx context.Context, db *gorm.DB, userId uuid.UUID, points float64) error {
	if db == nil {
		db = cmn.GormDB
	}
	if userId == uuid.Nil {
		e := fmt.Errorf("userId is nil")
		z.Error(e.Error())
		return e
	}
	if points <= 0 {
		e := fmt.Errorf("points must be positive")
		z.Error(e.Error())
		return e
	}

	// 查询用户现有积分记录
	var userPoints cmn.TUserPoints
	err := db.Where("user_id = ?", userId).First(&userPoints).Error
	if err != nil {
		e := fmt.Errorf("failed to query user points: %w, userId: %s", err, userId.String())
		z.Error(e.Error())
		return e
	}

	// 累加积分到原有积分上
	newTotalPoints := userPoints.DefaultPoints + points

	// 更新积分记录
	err = db.Model(&userPoints).Update("default_points", newTotalPoints).Error
	if err != nil {
		e := fmt.Errorf("failed to update user points: %w, userId: %s", err, userId.String())
		z.Error(e.Error())
		return e
	}

	return nil
}

// AddAllUserPointsFromAssets 根据资产计算并累加到已存在的用户积分
// 自动遍历积分表中的所有用户，计算其资产总价值并累加到现有积分上
func AddAllUserPointsFromAssets(ctx context.Context, db *gorm.DB) []error {
	if db == nil {
		db = cmn.GormDB
	}

	// 查询所有已存在积分记录的用户
	var allUserPoints []cmn.TUserPoints
	err := cmn.GormDB.Find(&allUserPoints).Error
	if err != nil {
		z.Error("failed to query all user points", zap.Error(err))
		return []error{err}
	}

	var successCount, failCount int
	var errs []error

	// 遍历每个用户进行积分更新
	for _, userPoint := range allUserPoints {
		err = addSingleUserPointsFromAssets(ctx, db, userPoint.UserId)
		if err != nil {
			errs = append(errs, err)
			failCount++
			continue
		}
		successCount++
	}

	if failCount > 0 {
		return errs
	}

	return nil
}

// addSingleUserPointsFromAssets  根据资产计算并累加到单个用户积分
// 计算用户资产总价值并累加到现有积分上
func addSingleUserPointsFromAssets(ctx context.Context, db *gorm.DB, userId uuid.UUID) error {
	if db == nil {
		db = cmn.GormDB
	}
	if userId == uuid.Nil {
		e := fmt.Errorf("userId is nil")
		z.Error(e.Error())
		return e
	}

	// 查询用户所有资产及其对应的元资产信息
	var userAssets []struct {
		MetaAssetId    int64   `gorm:"column:meta_asset_id"`
		MetaAssetValue float64 `gorm:"column:meta_asset_value"`
		Count          int64   `gorm:"column:asset_count"`
	}

	// 使用 VUserAssetMeta 视图查询，统计每个元资产的数量
	err := db.Model(&cmn.VUserAssetMeta{}).
		Select("meta_asset_id, meta_asset_value, COUNT(id) as asset_count").
		Where("user_id = ?", userId).
		Group("meta_asset_id, meta_asset_value").
		Scan(&userAssets).Error

	if err != nil {
		e := fmt.Errorf("failed to query user assets: %w, userId: %s", err, userId.String())
		z.Error(e.Error())
		return e
	}

	// 计算资产总积分
	var assetPoints float64
	for _, asset := range userAssets {
		assetPoints += asset.MetaAssetValue * float64(asset.Count)
	}

	// 查询现有积分记录
	var userPoints cmn.TUserPoints
	err = db.Where("user_id = ?", userId).First(&userPoints).Error
	if err != nil {
		return err
	}

	// 累加积分到原有积分上
	newTotalPoints := userPoints.DefaultPoints + assetPoints

	// 更新积分记录
	err = db.Model(&userPoints).Update("default_points", newTotalPoints).Error
	if err != nil {
		e := fmt.Errorf("failed to update user points: %w, userId: %v", err, userId)
		z.Error(e.Error())
		return e
	}

	return nil
}
