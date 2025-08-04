package asset

import (
	"encoding/json"
	"net/http"
	"strconv"

	"WugongMeta/cmn"
	"WugongMeta/serve/user_mgt"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type Handler interface {
	QueryMyAssets(c *gin.Context)
}

type handler struct {
}

func NewHandler() Handler {
	return &handler{}
}

// QueryMyAssets 查询我的资产
func (h *handler) QueryMyAssets(c *gin.Context) {
	// 获取当前用户ID
	userId, ok := user_mgt.GetCurrentUserID(c)
	if !ok {
		z.Error("failed to get current user ID")
		c.JSON(http.StatusOK, cmn.ReplyProto{
			Status: 1,
			Msg:    "未登录或登录已过期",
		})
		return
	}

	// 获取分页参数
	page := c.Query("page")
	if page == "" {
		page = "1"
	}
	pageSize := c.Query("page_size")
	if pageSize == "" {
		pageSize = "10"
	}

	// 转换分页参数
	pageInt, err := strconv.Atoi(page)
	if err != nil || pageInt < 1 {
		pageInt = 1
	}
	pageSizeInt, err := strconv.Atoi(pageSize)
	if err != nil || pageSizeInt < 1 || pageSizeInt > 100 {
		pageSizeInt = 10
	}

	// 计算偏移量
	offset := (pageInt - 1) * pageSizeInt

	// 查询用户资产总数
	var totalCount int64
	err = cmn.GormDB.Model(&cmn.VUserAssetMeta{}).Where("user_id = ?", userId).Count(&totalCount).Error
	if err != nil {
		z.Error("failed to count user assets", zap.Error(err), zap.String("user_id", userId.String()))
		c.JSON(http.StatusOK, cmn.ReplyProto{
			Status: -1,
			Msg:    "查询资产总数失败",
		})
		return
	}

	// 查询用户资产列表
	var userAssets []cmn.VUserAssetMeta
	err = cmn.GormDB.Where("user_id = ?", userId).
		Order("created_at DESC").
		Limit(pageSizeInt).
		Offset(offset).
		Find(&userAssets).Error
	if err != nil {
		z.Error("failed to query user assets", zap.Error(err), zap.String("user_id", userId.String()))
		c.JSON(http.StatusOK, cmn.ReplyProto{
			Status: -1,
			Msg:    "查询用户资产失败",
		})
		return
	}

	// 构建响应数据
	type AssetResponse struct {
		MetaAssetId   int64  `json:"meta_asset_id"`
		MetaAssetName string `json:"meta_asset_name"`
		MetaCoverImg  string `json:"meta_cover_img"`
		Name          string `json:"name"`
		ThemeName     string `json:"theme_name"`
		ExternalId    string `json:"external_id"`
		CoverImg      string `json:"cover_img"`
		CreatedAt     int64  `json:"created_at"`
		UpdatedAt     int64  `json:"updated_at"`
	}

	var responseAssets []AssetResponse
	for _, asset := range userAssets {
		responseAssets = append(responseAssets, AssetResponse{
			MetaAssetId:   asset.MetaAssetId,
			MetaAssetName: asset.MetaAssetName,
			MetaCoverImg:  asset.MetaCoverImg,
			Name:          asset.Name,
			ThemeName:     asset.ThemeName,
			ExternalId:    asset.ExternalNo,
			CoverImg:      asset.CoverImg,
			CreatedAt:     asset.CreatedAt,
			UpdatedAt:     asset.UpdatedAt,
		})
	}

	// 序列化响应数据
	responseJson, err := json.Marshal(responseAssets)
	if err != nil {
		z.Error("failed to marshal response data", zap.Error(err))
		c.JSON(http.StatusOK, cmn.ReplyProto{
			Status: -1,
			Msg:    "响应数据序列化失败",
		})
		return
	}

	// 返回成功响应
	c.JSON(http.StatusOK, cmn.ReplyProto{
		Status:   0,
		Msg:      "success",
		Data:     responseJson,
		RowCount: totalCount,
	})
}
