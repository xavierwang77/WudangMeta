package ubanquan_core

import (
	"WudangMeta/cmn"
	"WudangMeta/cmn/points_core"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/valyala/fasthttp"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// UpdateAllUsersAssets 遍历所有已绑定openId的用户，批量更新他们的优版权资产
// 返回所有用户的更新结果列表
func UpdateAllUsersAssets(ctx context.Context) ([]*AssetUpdateResult, error) {
	// 查询所有已绑定优版权账号的用户
	var userExternals []cmn.TUserExternal
	err := cmn.GormDB.Where("platform = ? AND open_id != '' AND open_id IS NOT NULL", AssetPlatform).Find(&userExternals).Error
	if err != nil {
		z.Error("failed to query users with ubanquan openId", zap.Error(err))
		return nil, fmt.Errorf("failed to query users with ubanquan openId: %w", err)
	}

	if len(userExternals) == 0 {
		z.Info("no users with ubanquan openId found")
		return []*AssetUpdateResult{}, nil
	}

	z.Info("starting batch update for all users with ubanquan openId", zap.Int("user_count", len(userExternals)))

	results := make([]*AssetUpdateResult, 0, len(userExternals))
	successCount := 0
	failureCount := 0

	// 遍历每个用户进行资产更新
	for _, userExternal := range userExternals {
		result, err := UpdateUserAssetByUserId(ctx, userExternal.UserId)
		if err != nil {
			z.Error("failed to update user asset",
				zap.Error(err),
				zap.String("user_id", userExternal.UserId.String()),
				zap.String("open_id", userExternal.OpenId))
			failureCount++
			// 即使单个用户更新失败，也继续处理其他用户
			if result == nil {
				result = &AssetUpdateResult{
					UserId:   userExternal.UserId,
					Success:  false,
					ErrorMsg: fmt.Sprintf("update failed: %v", err),
				}
			}
		} else if result.Success {
			successCount++
		} else {
			failureCount++
		}

		results = append(results, result)

		// 添加短暂延迟，避免对优版权API造成过大压力
		time.Sleep(100 * time.Millisecond)
	}

	z.Info("batch update completed",
		zap.Int("total_users", len(userExternals)),
		zap.Int("success_count", successCount),
		zap.Int("failure_count", failureCount))

	return results, nil
}

// UpdateUserAssetByUserId 根据用户ID更新单个用户的优版权资产
// 从优版权API获取指定用户的资产信息并同步到本地数据库
func UpdateUserAssetByUserId(ctx context.Context, userId uuid.UUID) (*AssetUpdateResult, error) {
	result := &AssetUpdateResult{
		UserId:  userId,
		Success: false,
	}

	// 获取用户的外部openId
	var userExternal cmn.TUserExternal
	err := cmn.GormDB.Where("user_id = ? AND platform = ?", userId, AssetPlatform).First(&userExternal).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			z.Error("user external info not found", zap.String("user_id", userId.String()))
			result.ErrorMsg = "user has not bound ubanquan account"
			return result, err
		}
		z.Error("failed to get user external info", zap.Error(err), zap.String("user_id", userId.String()))
		result.ErrorMsg = "failed to get user external info"
		return result, err
	}

	if userExternal.OpenId == "" {
		e := fmt.Errorf("user external openId is empty for user_id: %s", userId.String())
		z.Error(e.Error())
		result.ErrorMsg = "user has not bound ubanquan account"
		return result, e
	}

	// 调用优版权API获取用户资产
	cardResp, err := fetchUserAssetsFromUbanquan(userExternal.OpenId)
	if err != nil {
		z.Error("failed to fetch user assets from ubanquan", zap.Error(err), zap.String("user_id", userId.String()))
		result.ErrorMsg = fmt.Sprintf("failed to fetch user assets: %v", err)
		return result, err
	}

	// 同步资产到本地数据库
	addedCount, skippedCount, err := syncUserAssetsToDatabase(ctx, userId, cardResp)
	if err != nil {
		z.Error("failed to sync user assets to database", zap.Error(err), zap.String("user_id", userId.String()))
		result.ErrorMsg = fmt.Sprintf("failed to sync user assets: %v", err)
		return result, err
	}

	result.AddedCount = addedCount
	result.SkippedCount = skippedCount
	result.TotalCount = addedCount + skippedCount
	result.Success = true

	return result, nil
}

// fetchUserAssetsFromUbanquan 从优版权API获取用户资产信息
func fetchUserAssetsFromUbanquan(openId string) (*UbanquanCardResponse, error) {
	// 获取全局token
	token := GetGlobalToken()
	if token == nil {
		e := fmt.Errorf("global token is not available")
		z.Error(e.Error())
		return nil, e
	}

	// 向优版权API发送GET请求获取用户资产
	url := fmt.Sprintf("%s/dapp/card?openId=%s", BaseApiUrl, openId)
	fastReq := fasthttp.AcquireRequest()
	fastResp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(fastReq)
	defer fasthttp.ReleaseResponse(fastResp)

	fastReq.SetRequestURI(url)
	fastReq.Header.SetMethod("GET")
	fastReq.Header.Set("Authorization", token.AccessToken)

	// 发送请求
	client := &fasthttp.Client{
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	err := client.Do(fastReq, fastResp)
	if err != nil {
		return nil, fmt.Errorf("failed to send request to ubanquan card API: %w", err)
	}

	// 解析响应
	var cardResp UbanquanCardResponse
	err = json.Unmarshal(fastResp.Body(), &cardResp)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal ubanquan card response: %w", err)
	}

	// 检查API响应状态
	if !cardResp.Success {
		return nil, fmt.Errorf("ubanquan card API returned error, code: %v, message: %v", cardResp.Code, cardResp.Message)
	}

	return &cardResp, nil
}

// syncUserAssetsToDatabase 将用户资产同步到本地数据库
func syncUserAssetsToDatabase(ctx context.Context, userId uuid.UUID, cardResp *UbanquanCardResponse) (addedCount, skippedCount int, err error) {
	err = cmn.GormDB.Transaction(func(tx *gorm.DB) error {
		// 遍历资产数据
		for _, assetData := range cardResp.Data {
			for _, nfrInfo := range assetData.NFRInfoList {
				// 查找匹配的元资产
				var metaAsset cmn.TMetaAsset
				err = tx.Where("name = ? AND platform = ?", nfrInfo.ThemeName, AssetPlatform).First(&metaAsset).Error
				if err != nil {
					if errors.Is(err, gorm.ErrRecordNotFound) {
						// 元资产不存在，跳过
						skippedCount++
						continue
					}
					return fmt.Errorf("failed to query meta asset: %w", err)
				}

				// 检查用户是否已拥有该资产
				var existingUserAsset cmn.TUserAsset
				err = tx.Where("user_id = ? AND meta_asset_id = ? AND external_no = ?",
					userId, metaAsset.Id, nfrInfo.ProductNo).First(&existingUserAsset).Error
				if err == nil {
					// 资产已存在，跳过
					skippedCount++
					continue
				} else if !errors.Is(err, gorm.ErrRecordNotFound) {
					return fmt.Errorf("failed to query existing user asset: %w", err)
				}

				// 创建新的用户资产记录
				userAsset := cmn.TUserAsset{
					UserId:      userId,
					MetaAssetId: metaAsset.Id,
					Name:        nfrInfo.Name,
					ThemeName:   nfrInfo.ThemeName,
					ExternalNo:  nfrInfo.ProductNo,
					CoverImg:    nfrInfo.CoverImg,
				}

				err = tx.Create(&userAsset).Error
				if err != nil {
					return fmt.Errorf("failed to create user asset: %w, user_id: %s", err, userId.String())
				}

				// 给用户增加该资产积分
				err = points_core.AddUserPointsByAsset(ctx, tx, userId, metaAsset.Id, 1)
				if err != nil {
					return fmt.Errorf("failed to add user points by asset: %w, user_id: %s, meta_asset_id: %d", err, userId.String(), metaAsset.Id)
				}

				addedCount++
			}
		}
		return nil
	})

	return addedCount, skippedCount, err
}

// fetchAccessToken 获取访问令牌
// 向优版权API发送POST请求获取accessToken
func fetchAccessToken(ctx context.Context, appId string, appSecret string) (*Token, error) {
	// 构建请求体
	reqData := map[string]string{
		"appId":     appId,
		"appSecret": appSecret,
	}

	reqBody, err := json.Marshal(reqData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request data: %w", err)
	}

	// 发送POST请求
	fastReq := fasthttp.AcquireRequest()
	fastResp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(fastReq)
	defer fasthttp.ReleaseResponse(fastResp)

	fastReq.SetRequestURI(fmt.Sprintf("%s/dapp/token", BaseApiUrl))
	fastReq.Header.SetMethod("POST")
	fastReq.Header.SetContentType("application/json")
	fastReq.SetBody(reqBody)

	client := &fasthttp.Client{
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	err = client.Do(fastReq, fastResp)
	if err != nil {
		return nil, fmt.Errorf("failed to send request to ubanquan token API: %w", err)
	}

	// 解析响应
	var tokenResp struct {
		Success bool        `json:"success"`
		Code    interface{} `json:"code"`
		Message interface{} `json:"message"`
		Data    *Token      `json:"data"`
	}

	err = json.Unmarshal(fastResp.Body(), &tokenResp)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal token response: %w", err)
	}

	// 检查API响应状态
	if !tokenResp.Success {
		return nil, fmt.Errorf("ubanquan token API returned error, code: %v, message: %v", tokenResp.Code, tokenResp.Message)
	}

	if tokenResp.Data == nil {
		return nil, fmt.Errorf("ubanquan token API returned empty data")
	}

	return tokenResp.Data, nil
}

// refreshAccessToken 刷新访问令牌
// 向优版权API发送GET请求刷新令牌
func refreshAccessToken(ctx context.Context, refreshToken string) (*Token, error) {
	// 构建请求URL
	url := fmt.Sprintf("%s/dapp/flush?refreshToken=%s", BaseApiUrl, refreshToken)

	// 发送GET请求
	fastReq := fasthttp.AcquireRequest()
	fastResp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(fastReq)
	defer fasthttp.ReleaseResponse(fastResp)

	fastReq.SetRequestURI(url)
	fastReq.Header.SetMethod("GET")

	client := &fasthttp.Client{
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	err := client.Do(fastReq, fastResp)
	if err != nil {
		return nil, fmt.Errorf("failed to send request to ubanquan flush API: %w", err)
	}

	// 解析响应
	var tokenResp struct {
		Success bool        `json:"success"`
		Code    interface{} `json:"code"`
		Message interface{} `json:"message"`
		Data    *Token      `json:"data"`
	}

	err = json.Unmarshal(fastResp.Body(), &tokenResp)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal refresh token response: %w", err)
	}

	// 检查API响应状态
	if !tokenResp.Success {
		return nil, fmt.Errorf("ubanquan flush API returned error, code: %v, message: %v", tokenResp.Code, tokenResp.Message)
	}

	if tokenResp.Data == nil {
		return nil, fmt.Errorf("ubanquan flush API returned empty data")
	}

	return tokenResp.Data, nil
}
