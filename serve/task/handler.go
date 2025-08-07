package task

import (
	"WugongMeta/cmn"
	"WugongMeta/cmn/points_core"
	"WugongMeta/serve/user_mgt"
	"encoding/json"
	"errors"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"net/http"
	"time"
)

type Handler interface {
	HandleAnalyzeMyFortune(c *gin.Context)
	HandleDailyCheckIn(c *gin.Context)
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

// HandleDailyCheckIn 处理每日签到请求
func (h *handler) HandleDailyCheckIn(c *gin.Context) {
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
			Points:    dailyCheckInScore,
			CreatedAt: now.UnixMilli(),
		}

		err = tx.Create(&checkInRecord).Error
		if err != nil {
			z.Error("failed to create check in record", zap.Error(err), zap.String("user_id", userId.String()))
			return err
		}

		// 查询用户积分记录
		var userPoints cmn.TUserPoints
		err = tx.Where("user_id = ?", userId).First(&userPoints).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				// 用户积分记录不存在，先初始化
				err = points_core.InitializeUserPoints(c, tx, userId)
				if err != nil {
					z.Error("failed to initialize user points", zap.Error(err), zap.String("user_id", userId.String()))
					return err
				}
				// 重新查询用户积分记录
				err = tx.Where("user_id = ?", userId).First(&userPoints).Error
				if err != nil {
					z.Error("failed to query user points after initialization", zap.Error(err), zap.String("user_id", userId.String()))
					return err
				}
			} else {
				z.Error("failed to query user points", zap.Error(err), zap.String("user_id", userId.String()))
				return err
			}
		}

		// 累加签到积分到用户总积分
		newTotalPoints := userPoints.DefaultPoints + dailyCheckInScore
		err = tx.Model(&userPoints).Update("default_points", newTotalPoints).Error
		if err != nil {
			z.Error("failed to update user points", zap.Error(err), zap.String("user_id", userId.String()))
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
