package points

import (
	"WudangMeta/cmn"
	"WudangMeta/serve/user"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type Handler interface {
	HandleQueryMyPoints(c *gin.Context)
}

type handler struct {
}

func NewHandler() Handler {
	return &handler{}
}

// HandleQueryMyPoints 处理获取当前用户积分
func (h *handler) HandleQueryMyPoints(c *gin.Context) {
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

	// 查询用户积分
	var userPoints cmn.TUserPoints
	err := cmn.GormDB.Where("user_id = ?", userId).First(&userPoints).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			z.Info("user points not found", zap.String("user_id", userId.String()))
			c.JSON(http.StatusOK, cmn.ReplyProto{
				Status: 1,
				Msg:    "未找到用户积分记录",
			})
			return
		}
		z.Error("failed to query user points", zap.Error(err), zap.String("user_id", userId.String()))
		c.JSON(http.StatusOK, cmn.ReplyProto{
			Status: -1,
			Msg:    "查询用户积分失败",
		})
		return
	}

	responseData := map[string]interface{}{
		"points": userPoints.DefaultPoints,
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
		Msg:    "success",
		Data:   responseJson,
	})
}
