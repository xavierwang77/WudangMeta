package ubanquan

import (
	"WudangMeta/cmn"
	"WudangMeta/cmn/points_core"
	"WudangMeta/cmn/ubanquan_core"
	"WudangMeta/serve/user"
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
	HandleAuthentication(c *gin.Context)
	HandleUpdateMyAsset(c *gin.Context)
}

type handler struct {
}

func NewHandler() Handler {
	return &handler{}
}

// HandleAuthentication 处理优版权用户授权
func (h *handler) HandleAuthentication(c *gin.Context) {
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

	// 获取全局token
	token := ubanquan_core.GetGlobalToken()
	if token == nil {
		z.Error("global token is not available")
		c.JSON(http.StatusOK, cmn.ReplyProto{
			Status: -1,
			Msg:    "优版权token未初始化",
		})
		return
	}

	// 向优版权API发送GET请求
	url := fmt.Sprintf("%s/dapp/authentication?code=%s", ubanquan_core.BaseApiUrl, code)
	fastReq := fasthttp.AcquireRequest()
	fastResp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(fastReq)
	defer fasthttp.ReleaseResponse(fastResp)

	fastReq.SetRequestURI(url)
	fastReq.Header.SetMethod("GET")
	fastReq.Header.Set("access-token", token.AccessToken)

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
			Msg:    "优版权API响应解析失败",
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
	userId, ok := user.GetCurrentUserID(c)
	if !ok {
		z.Error("failed to get current user ID")
		c.JSON(http.StatusOK, cmn.ReplyProto{
			Status: 401,
			Msg:    "未登录或登录已过期",
		})
		return
	}

	var (
		status int
		msg    string
	)

	err = cmn.GormDB.Transaction(func(tx *gorm.DB) error {
		// 查找或创建用户外部信息记录
		var userExternal cmn.TUserExternal
		err = tx.Where("user_id = ? AND platform = ?", userId, ubanquan_core.PlatformName).First(&userExternal).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				// 创建新记录
				userExternal = cmn.TUserExternal{
					Platform: ubanquan_core.PlatformName,
					UserId:   userId,
					OpenId:   ubanquanResp.Data.OpenId,
					NickName: ubanquanResp.Data.NickName,
					Avatar:   ubanquanResp.Data.HeadImg,
				}
				err = tx.Create(&userExternal).Error
				if err != nil {
					e := fmt.Errorf("failed to create user external record: %w", err)
					z.Error(e.Error())
					status = -1
					msg = "创建用户外部信息失败"
					return e
				}
			} else {
				e := fmt.Errorf("failed to query user external record: %w", err)
				z.Error(e.Error())
				status = -1
				msg = "查询用户外部信息失败"
				return e
			}
		} else {
			// 更新现有记录
			userExternal.OpenId = ubanquanResp.Data.OpenId
			userExternal.NickName = ubanquanResp.Data.NickName
			userExternal.Avatar = ubanquanResp.Data.HeadImg
			err = tx.Save(&userExternal).Error
			if err != nil {
				e := fmt.Errorf("failed to update user external record: %w", err)
				z.Error(e.Error())
				status = -1
				msg = "更新用户外部信息失败"
				return e
			}
		}
		return nil
	})
	if err != nil {
		c.JSON(http.StatusOK, cmn.ReplyProto{
			Status: status,
			Msg:    msg,
		})
		return
	}

	c.JSON(http.StatusOK, cmn.ReplyProto{
		Status: 0,
		Msg:    "success",
	})
	return
}

// HandleUpdateMyAsset 处理更新我的优版权资产
// 从优版权API获取用户资产信息并同步到本地数据库
func (h *handler) HandleUpdateMyAsset(c *gin.Context) {
	// 获取当前用户ID
	userId, ok := user.GetCurrentUserID(c)
	if !ok {
		z.Error("failed to get current user ID")
		c.JSON(http.StatusOK, cmn.ReplyProto{
			Status: 401,
			Msg:    "未登录或登录已过期",
		})
		return
	}

	// 获取用户的外部openId
	openId, ok := user.GetCurrentUserExternalOpenId(c)
	if !ok || openId == "" {
		z.Error("failed to get user external openId", zap.String("user_id", userId.String()))
		c.JSON(http.StatusOK, cmn.ReplyProto{
			Status: 1,
			Msg:    "用户未绑定优版权账号",
		})
		return
	}

	// 获取全局token
	token := ubanquan_core.GetGlobalToken()
	if token == nil {
		z.Error("global token is not available")
		c.JSON(http.StatusOK, cmn.ReplyProto{
			Status: -1,
			Msg:    "优版权token未初始化",
		})
		return
	}

	// 向优版权API发送GET请求获取用户资产
	url := fmt.Sprintf("%s/dapp/card?openId=%s", ubanquan_core.BaseApiUrl, openId)
	fastReq := fasthttp.AcquireRequest()
	fastResp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(fastReq)
	defer fasthttp.ReleaseResponse(fastResp)

	fastReq.SetRequestURI(url)
	fastReq.Header.SetMethod("GET")
	fastReq.Header.Set("access-token", token.AccessToken)

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

	var cardResp ubanquan_core.UbanquanCardResponse
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

	var (
		status int
		msg    string
	)

	err = cmn.GormDB.Transaction(func(tx *gorm.DB) error {
		// 遍历资产数据
		for _, assetData := range cardResp.Data {
			for _, nfrInfo := range assetData.NFRInfoList {
				// 查找匹配的元资产
				var metaAsset cmn.TMetaAsset
				err = tx.Where("name = ? AND platform = ?", nfrInfo.ThemeName, ubanquan_core.PlatformName).First(&metaAsset).Error
				if err != nil {
					if errors.Is(err, gorm.ErrRecordNotFound) {
						// 元资产不存在，创建新的元资产
						metaAsset = cmn.TMetaAsset{
							Name:       assetData.MetaProductName,
							CoverImg:   assetData.MetaProductImg,
							ExternalNo: assetData.MetaProductNo,
							Value:      0,
							Platform:   ubanquan_core.PlatformName,
						}
						err = tx.Create(&metaAsset).Error
						if err != nil {
							e := fmt.Errorf("failed to create meta asset: %w", err)
							z.Error(e.Error())
							status = -1
							msg = "创建元资产失败"
							return e
						}
					} else {
						e := fmt.Errorf("failed to query meta asset: %w", err)
						z.Error(e.Error())
						status = -1
						msg = "查询元资产失败"
						return e
					}
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
					e := fmt.Errorf("failed to query existing user asset: %w", err)
					z.Error(e.Error())
					status = -1
					msg = "查询用户是否已拥有资产失败"
					return e
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
					e := fmt.Errorf("failed to create user asset: %w, user_id: %s", err, userId.String())
					z.Error(e.Error())
					status = -1
					msg = "创建用户资产失败"
					return e
				}

				// 给用户增加该资产积分
				err = points_core.AddUserPointsByAsset(c, tx, userId, metaAsset.Id, 1)
				if err != nil {
					e := fmt.Errorf("failed to add user points by asset: %w, user_id: %s, meta_asset_id: %d", err, userId.String(), metaAsset.Id)
					status = -1
					msg = "添加用户积分失败"
					return e
				}

				addedCount++
			}
		}
		return nil
	})
	if err != nil {
		c.JSON(http.StatusOK, cmn.ReplyProto{
			Status: status,
			Msg:    fmt.Sprintf("资产同步失败: %s", msg),
		})
		return
	}

	// 返回处理结果
	responseData := map[string]interface{}{
		"addedCount":   addedCount,
		"skippedCount": skippedCount,
		"totalCount":   addedCount + skippedCount,
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
