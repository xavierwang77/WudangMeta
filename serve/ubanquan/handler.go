package ubanquan

import (
	"WugongMeta/cmn"
	"WugongMeta/serve/user_mgt"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/valyala/fasthttp"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type Handler interface {
	Authentication(c *gin.Context)
	UpdateMyAsset(c *gin.Context)
}

type handler struct {
}

func NewHandler() Handler {
	return &handler{}
}

// Authentication 优版权用户授权
func (h *handler) Authentication(c *gin.Context) {
	// 从 query 参数获取 code
	code := c.Query("code") // 如果参数不存在会返回空字符串
	if code == "" {
		z.Error("missing query param: code")
		c.JSON(http.StatusOK, cmn.ReplyProto{
			Status: 1,
			Msg:    "缺少必要的 query 参数 code",
		})
		return
	}

	// 向优版权API发送GET请求
	url := fmt.Sprintf("https://test-apimall.ubanquan.cn/dapp/authentication?code=%s", code)
	fastReq := fasthttp.AcquireRequest()
	fastResp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(fastReq)
	defer fasthttp.ReleaseResponse(fastResp)

	fastReq.SetRequestURI(url)
	fastReq.Header.SetMethod("GET")

	// 发送请求
	client := &fasthttp.Client{
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	err := client.Do(fastReq, fastResp)
	if err != nil {
		z.Error("failed to send request to ubanquan API", zap.Error(err))
		c.JSON(http.StatusOK, cmn.ReplyProto{
			Status: -1,
			Msg:    "向优版权发送用户授权请求失败",
		})
		return
	}

	// 解析响应
	type UbanquanResponse struct {
		Code    string `json:"code"`
		Message string `json:"message"`
		Success bool   `json:"success"`
		Data    struct {
			OpenId   string `json:"openId"`
			NickName string `json:"nickName"`
			HeadImg  string `json:"headImg"`
		} `json:"data"`
	}

	var ubanquanResp UbanquanResponse
	err = json.Unmarshal(fastResp.Body(), &ubanquanResp)
	if err != nil {
		z.Error("failed to unmarshal ubanquan response", zap.Error(err))
		c.JSON(http.StatusOK, cmn.ReplyProto{
			Status: 1,
			Msg:    "第三方API响应解析失败",
		})
		return
	}

	// 检查优版权API响应状态
	if !ubanquanResp.Success {
		z.Error("ubanquan API returned error", zap.String("code", ubanquanResp.Code), zap.String("message", ubanquanResp.Message))
		c.JSON(http.StatusOK, cmn.ReplyProto{
			Status: -1,
			Msg:    fmt.Sprintf("优版权认证失败: %s", ubanquanResp.Message),
		})
		return
	}

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

	// 查找或创建用户外部信息记录
	var userExternal cmn.TUserExternal
	err = cmn.GormDB.Where("user_id = ?", userId).First(&userExternal).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// 创建新记录
			userExternal = cmn.TUserExternal{
				UserId:   userId,
				OpenId:   ubanquanResp.Data.OpenId,
				NickName: ubanquanResp.Data.NickName,
				Avatar:   ubanquanResp.Data.HeadImg,
			}
			err = cmn.GormDB.Create(&userExternal).Error
			if err != nil {
				z.Error("failed to create user external record", zap.Error(err))
				c.JSON(http.StatusOK, cmn.ReplyProto{
					Status: -1,
					Msg:    "创建用户外部信息失败",
				})
				return
			}
		} else {
			z.Error("failed to query user external record", zap.Error(err))
			c.JSON(http.StatusOK, cmn.ReplyProto{
				Status: -1,
				Msg:    "查询用户外部信息失败",
			})
			return
		}
	} else {
		// 更新现有记录
		userExternal.OpenId = ubanquanResp.Data.OpenId
		userExternal.NickName = ubanquanResp.Data.NickName
		userExternal.Avatar = ubanquanResp.Data.HeadImg
		err = cmn.GormDB.Save(&userExternal).Error
		if err != nil {
			z.Error("failed to update user external record", zap.Error(err))
			c.JSON(http.StatusOK, cmn.ReplyProto{
				Status: -1,
				Msg:    "更新用户外部信息失败",
			})
			return
		}
	}

	c.JSON(http.StatusOK, cmn.ReplyProto{
		Status: 0,
		Msg:    "success",
	})
}

// UpdateMyAsset 更新我的优版权资产
// 从优版权API获取用户资产信息并同步到本地数据库
func (h *handler) UpdateMyAsset(c *gin.Context) {
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

	// 获取用户的外部openId
	openId, ok := user_mgt.GetCurrentUserExternalOpenId(c)
	if !ok || openId == "" {
		z.Error("failed to get user external openId", zap.String("user_id", userId.String()))
		c.JSON(http.StatusOK, cmn.ReplyProto{
			Status: 1,
			Msg:    "用户未绑定优版权账号",
		})
		return
	}

	// 向优版权API发送GET请求获取用户资产
	url := fmt.Sprintf("https://test-apimall.ubanquan.cn/dapp/card?openId=%s", openId)
	fastReq := fasthttp.AcquireRequest()
	fastResp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(fastReq)
	defer fasthttp.ReleaseResponse(fastResp)

	fastReq.SetRequestURI(url)
	fastReq.Header.SetMethod("GET")

	// 发送请求
	client := &fasthttp.Client{
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	err := client.Do(fastReq, fastResp)
	if err != nil {
		z.Error("failed to send request to ubanquan card API", zap.Error(err))
		c.JSON(http.StatusOK, cmn.ReplyProto{
			Status: -1,
			Msg:    "获取用户资产信息失败",
		})
		return
	}

	// 解析响应
	type NFRInfo struct {
		LockTag   int    `json:"lockTag"`
		CD        int    `json:"cd"`
		ThemeName string `json:"themeName"`
		CoverImg  string `json:"coverImg"`
		Name      string `json:"name"`
		AuctionNo string `json:"auctionNo"`
		ProductNo string `json:"productNo"`
		ThemeKey  string `json:"themeKey"`
	}

	type AssetData struct {
		NFRInfoList     []NFRInfo `json:"nfrInfoList"`
		MetaProductName string    `json:"metaProductName"`
		MetaProductNo   string    `json:"metaProductNo"`
	}

	type UbanquanCardResponse struct {
		Code    interface{} `json:"code"`
		Data    []AssetData `json:"data"`
		Message interface{} `json:"message"`
		Success bool        `json:"success"`
	}

	var cardResp UbanquanCardResponse
	err = json.Unmarshal(fastResp.Body(), &cardResp)
	if err != nil {
		z.Error("failed to unmarshal ubanquan card response", zap.Error(err))
		c.JSON(http.StatusOK, cmn.ReplyProto{
			Status: -1,
			Msg:    "解析用户资产信息失败",
		})
		return
	}

	// 检查API响应状态
	if !cardResp.Success {
		z.Error("ubanquan card API returned error", zap.Any("code", cardResp.Code), zap.Any("message", cardResp.Message))
		c.JSON(http.StatusOK, cmn.ReplyProto{
			Status: -1,
			Msg:    "获取用户资产信息失败",
		})
		return
	}

	// 统计处理结果
	var addedCount int
	var skippedCount int

	// 遍历资产数据
	for _, assetData := range cardResp.Data {
		for _, nfrInfo := range assetData.NFRInfoList {
			// 查找匹配的元资产
			var metaAsset cmn.TMetaAsset
			err = cmn.GormDB.Where("name = ? AND platform = ?", nfrInfo.ThemeName, assetPlatform).First(&metaAsset).Error
			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					// 元资产不存在，跳过
					skippedCount++
					continue
				}
				z.Error("failed to query meta asset", zap.Error(err), zap.String("theme_name", nfrInfo.ThemeName))
				continue
			}

			// 检查用户是否已拥有该资产
			var existingUserAsset cmn.TUserAsset
			err = cmn.GormDB.Where("user_id = ? AND meta_asset_id = ? AND external_no = ?",
				userId, metaAsset.Id, nfrInfo.ProductNo).First(&existingUserAsset).Error
			if err == nil {
				// 资产已存在，跳过
				skippedCount++
				continue
			} else if !errors.Is(err, gorm.ErrRecordNotFound) {
				z.Error("failed to query existing user asset", zap.Error(err))
				continue
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

			err = cmn.GormDB.Create(&userAsset).Error
			if err != nil {
				z.Error("failed to create user asset", zap.Error(err),
					zap.String("user_id", userId.String()),
					zap.String("asset_name", nfrInfo.Name))
				continue
			}

			z.Info("user asset created successfully",
				zap.String("user_id", userId.String()),
				zap.String("asset_name", nfrInfo.Name),
				zap.String("theme_name", nfrInfo.ThemeName))
			addedCount++
		}
	}

	// 返回处理结果
	responseData := map[string]interface{}{
		"added_count":   addedCount,
		"skipped_count": skippedCount,
		"total_count":   addedCount + skippedCount,
	}

	responseJson, err := json.Marshal(responseData)
	if err != nil {
		z.Error("failed to marshal response data", zap.Error(err))
		c.JSON(http.StatusOK, cmn.ReplyProto{
			Status: -1,
			Msg:    "响应数据序列化失败",
		})
		return
	}

	c.JSON(http.StatusOK, cmn.ReplyProto{
		Status: 0,
		Msg:    fmt.Sprintf("资产同步完成，新增%d个，跳过%d个", addedCount, skippedCount),
		Data:   responseJson,
	})
}
