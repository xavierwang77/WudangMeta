package task

import (
	"WudangMeta/cmn"
	"WudangMeta/cmn/points_core"
	"WudangMeta/serve/user"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type Handler interface {
	HandleAnalyzeMyFortune(c *gin.Context)
	HandleDailyCheckIn(c *gin.Context)
	HandleQueryMyFortune(c *gin.Context)
}

type handler struct {
}

func NewHandler() Handler {
	return &handler{}
}

// HandleAnalyzeMyFortune 处理分析我的运势请求
func (h *handler) HandleAnalyzeMyFortune(c *gin.Context) {
	userId, ok := user.GetCurrentUserID(c)
	if !ok {
		z.Error("failed to get current userId from context")
		c.JSON(http.StatusOK, gin.H{
			"status": 401,
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

// HandleDailyCheckIn 处理每日签到请求
func (h *handler) HandleDailyCheckIn(c *gin.Context) {
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

	// 获取当前日期（只保留年月日）
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	tomorrow := today.Add(24 * time.Hour)
	// 转换为unix时间戳（毫秒）
	todayTimestamp := today.UnixMilli()
	tomorrowTimestamp := tomorrow.UnixMilli()

	var checkInRecord cmn.TUserCheckIn
	var alreadyCheckedIn bool

	err := cmn.GormDB.Transaction(func(tx *gorm.DB) error {
		// 检查今天是否已经签到
		err := tx.Where("user_id = ? AND created_at >= ? AND created_at < ?", userId, todayTimestamp, tomorrowTimestamp).
			First(&checkInRecord).Error
		if err == nil {
			// 今天已经签到过了
			alreadyCheckedIn = true
			return nil
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			// 数据库查询错误
			z.Error("failed to query check in record", zap.Error(err), zap.String("user_id", userId.String()))
			return err
		}

		// 今天还没有签到，创建签到记录
		checkInRecord = cmn.TUserCheckIn{
			UserId:    userId,
			Points:    dailyCheckInPoints,
			CreatedAt: now.UnixMilli(),
		}

		err = tx.Create(&checkInRecord).Error
		if err != nil {
			z.Error("failed to create check in record", zap.Error(err), zap.String("user_id", userId.String()))
			return err
		}

		// 累加用户积分
		err = points_core.AddUserPoints(c, tx, userId, dailyCheckInPoints)
		if err != nil {
			z.Error("failed to add user points", zap.Error(err), zap.String("user_id", userId.String()))
			return err
		}

		alreadyCheckedIn = false
		return nil
	})

	if err != nil {
		z.Error("failed to process daily check in", zap.Error(err))
		c.JSON(http.StatusOK, cmn.ReplyProto{
			Status: -1,
			Msg:    "签到失败",
		})
		return
	}

	// 构建响应数据
	replyData := map[string]interface{}{
		"alreadyCheckedIn": alreadyCheckedIn,
		"points":           checkInRecord.Points,
		"checkInTime":      checkInRecord.CreatedAt,
	}

	replyDataJson, err := json.Marshal(replyData)
	if err != nil {
		z.Error("failed to marshal reply data", zap.Error(err))
		c.JSON(http.StatusOK, cmn.ReplyProto{
			Status: -1,
			Msg:    "响应数据序列化失败",
		})
		return
	}

	var msg string
	if alreadyCheckedIn {
		msg = "今日已签到"
	} else {
		msg = "签到成功"
	}

	c.JSON(http.StatusOK, cmn.ReplyProto{
		Status: 0,
		Msg:    msg,
		Data:   replyDataJson,
	})
}

// HandleQueryMyFortune 查询我的运势数据
func (h *handler) HandleQueryMyFortune(c *gin.Context) {
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

	// 查询用户运势数据
	var userFortune cmn.TUserFortune
	err := cmn.GormDB.Where("user_id = ?", userId).First(&userFortune).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusOK, cmn.ReplyProto{
				Status: 1,
				Msg:    "暂无运势数据，请先进行运势分析",
			})
		} else {
			z.Error("failed to query user fortune", zap.Error(err), zap.String("user_id", userId.String()))
			c.JSON(http.StatusOK, cmn.ReplyProto{
				Status: -1,
				Msg:    "查询运势数据失败",
			})
		}
		return
	}

	// 将运势数据转换为JSON
	userFortuneJSON, err := json.Marshal(userFortune)
	if err != nil {
		z.Error("failed to marshal user fortune data", zap.Error(err))
		c.JSON(http.StatusOK, cmn.ReplyProto{
			Status: -1,
			Msg:    "运势数据序列化失败",
		})
		return
	}

	c.JSON(http.StatusOK, cmn.ReplyProto{
		Status: 0,
		Msg:    "success",
		Data:   userFortuneJSON,
	})
}
