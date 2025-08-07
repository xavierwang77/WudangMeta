package task

import (
	"WugongMeta/cmn"
	"WugongMeta/serve/user_mgt"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"net/http"
)

type Handler interface {
	HandleAnalyzeMyFortune(c *gin.Context)
}

type handler struct {
}

func NewHandler() Handler {
	return &handler{}
}

// HandleAnalyzeMyFortune 处理分析我的运势请求
func (h *handler) HandleAnalyzeMyFortune(c *gin.Context) {
	userId, ok := user_mgt.GetCurrentUserID(c)
	if !ok {
		z.Error("failed to get current userId from context")
		c.JSON(http.StatusOK, gin.H{
			"status": 1,
			"msg":    "未登录或登录已过期",
		})
		return
	}

	var req cmn.ReqProto
	err := c.ShouldBindJSON(&req)
	if err != nil {
		z.Error("failed to bind request JSON", zap.Error(err))
		c.JSON(http.StatusOK, gin.H{
			"status": 1,
			"msg":    "请求体结构错误",
		})
		return
	}

	type ReqData struct {
		Name   string `json:"name" form:"name"`
		Gender string `json:"gender" form:"gender"`
		Birth  string `json:"birth" form:"birth"`
	}

	var reqData ReqData
	err = json.Unmarshal(req.Data, &reqData)
	if err != nil {
		z.Error("failed to unmarshal request data", zap.Error(err))
		c.JSON(http.StatusOK, gin.H{
			"status": 1,
			"msg":    "请求体数据错误",
		})
		return
	}

	var fortune Fortune
	var points float64

	err = cmn.GormDB.Transaction(func(tx *gorm.DB) error {
		fortune, points, err = AnalyzeAndSaveFortune(c, tx, userId, reqData.Name, reqData.Gender, reqData.Birth)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		z.Error("failed to analyze fortune", zap.Error(err))
		c.JSON(http.StatusOK, gin.H{
			"status": -1,
			"msg":    "运势分析失败",
		})
		return
	}

	replyData := map[string]interface{}{
		"fortune": fortune,
		"points":  points,
	}

	replyDataJson, err := json.Marshal(replyData)
	if err != nil {
		z.Error("failed to marshal reply data", zap.Error(err))
		c.JSON(http.StatusOK, gin.H{
			"status": -1,
			"msg":    "响应数据序列化失败",
		})
		return
	}

	c.JSON(http.StatusOK, cmn.ReplyProto{
		Status: 0,
		Msg:    "success",
		Data:   replyDataJson,
	})
	return
}
