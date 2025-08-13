package user

import (
	"WudangMeta/cmn"
	"WudangMeta/cmn/points_core"
	"WudangMeta/cmn/sms"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type Handler interface {
	HandleSendSMSCode(c *gin.Context)
	HandleSMSLogin(c *gin.Context)
	HandleGetCurrentUserInfo(c *gin.Context)
	HandleGetUserInfoByPhone(c *gin.Context)
	HandleQueryUserInfoList(c *gin.Context)
}

type handler struct {
	smsSrv sms.Service
}

func NewHandler() Handler {
	return &handler{
		smsSrv: sms.NewService(),
	}
}

// HandleSendSMSCode 处理发送SMS验证码
func (h *handler) HandleSendSMSCode(c *gin.Context) {
	phone := c.Query("mobilePhone")
	if phone == "" {
		z.Error("phone number is empty")
		c.JSON(http.StatusOK, cmn.ReplyProto{
			Status: 1,
			Msg:    "手机号不能为空",
		})
		return
	}

	code := cmn.RandDigits(smsCodeLength)
	if code == "" {
		z.Error("failed to generate SMS code, code is empty")
		c.JSON(http.StatusOK, cmn.ReplyProto{
			Status: 1,
			Msg:    "生成短信验证码失败",
		})
		return
	}

	err := h.smsSrv.SendVerifyCode(phone, code)
	if err != nil {
		c.JSON(http.StatusOK, cmn.ReplyProto{
			Status: 1,
			Msg:    "发送短信验证码失败: " + err.Error(),
		})
		return
	}

	codeRow := cmn.TSmsCodes{
		MobilePhone: phone,
		Code:        code,
		ExpiresAt:   time.Now().UnixMilli() + 5*time.Minute.Milliseconds(), // 设置验证码有效期为5分钟
	}
	err = cmn.GormDB.Create(&codeRow).Error
	if err != nil {
		z.Error("failed to save SMS code", zap.Error(err), zap.String("phone", phone), zap.String("code", code))
		c.JSON(http.StatusOK, cmn.ReplyProto{
			Status: 1,
			Msg:    "保存短信验证码失败: " + err.Error(),
		})
		return
	}

	z.Info("sent sms code", zap.String("phone", phone), zap.String("code", code))

	c.JSON(http.StatusOK, cmn.ReplyProto{
		Status: 0,
		Msg:    "短信验证码已发送",
	})
	return
}

// HandleSMSLogin 处理使用短信验证码登录
func (h *handler) HandleSMSLogin(c *gin.Context) {
	var req cmn.ReqProto
	err := c.ShouldBindJSON(&req)
	if err != nil {
		z.Error("failed to bind request", zap.Error(err))
		c.JSON(http.StatusOK, cmn.ReplyProto{
			Status: 1,
			Msg:    "请求参数错误，请检查是否符合请求协议",
		})
		return
	}

	type data struct {
		MobilePhone string `json:"mobilePhone"`
		Code        string `json:"code"`
	}

	var d data
	err = json.Unmarshal(req.Data, &d)
	if err != nil {
		z.Error("failed to unmarshal request data", zap.Error(err))
		c.JSON(http.StatusOK, cmn.ReplyProto{
			Status: 1,
			Msg:    "请求数据格式错误",
		})
		return
	}

	if d.MobilePhone == "" {
		z.Error("phone number is empty")
		c.JSON(http.StatusOK, cmn.ReplyProto{
			Status: 1,
			Msg:    "手机号不能为空",
		})
		return
	}

	if d.Code == "" {
		z.Error("verification code is empty")
		c.JSON(http.StatusOK, cmn.ReplyProto{
			Status: 1,
			Msg:    "验证码不能为空",
		})
		return
	}

	var user cmn.TUser

	var (
		status int
		msg    string
	)

	err = cmn.GormDB.Transaction(func(tx *gorm.DB) error {
		// 验证短信验证码
		var smsCode cmn.TSmsCodes
		err = tx.Where("mobile_phone = ? AND code = ? AND expires_at > ?", d.MobilePhone, d.Code, time.Now().UnixMilli()).First(&smsCode).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				e := fmt.Errorf("verification code not found or expired, phone: %s, code: %s", d.MobilePhone, d.Code)
				z.Error(e.Error())
				status = 1
				msg = "验证码错误或已过期，请重新获取"
				return e
			}
			e := fmt.Errorf("failed to query verification code: %w, phone: %s, code: %s", err, d.MobilePhone, d.Code)
			z.Error(e.Error())
			status = -1
			msg = "验证码验证失败，请稍后再试"
			return e
		}

		// 验证成功后删除验证码
		err = tx.Delete(&smsCode).Error
		if err != nil {
			z.Error("failed to delete verification code", zap.Error(err))
		}

		// 查找或创建用户
		err = tx.Where("mobile_phone = ?", d.MobilePhone).First(&user).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				// 用户不存在，创建新用户
				user = cmn.TUser{
					Id:          uuid.New(),
					MobilePhone: d.MobilePhone,
					NickName:    d.MobilePhone, // 默认昵称为手机号
					Status:      "00",          // 启用状态
					LoginTime:   time.Now().UnixMilli(),
				}
				err = tx.Create(&user).Error
				if err != nil {
					e := fmt.Errorf("failed to create user: %w, phone: %s", err, d.MobilePhone)
					z.Error(e.Error())
					msg = "创建用户失败"
					status = -1
					return e
				}
				z.Info("new user registered", zap.String("phone", d.MobilePhone), zap.String("userId", user.Id.String()))
			} else {
				e := fmt.Errorf("failed to query user: %w, phone: %s", err, d.MobilePhone)
				z.Error(e.Error())
				msg = "查询用户失败"
				status = -1
				return e
			}
		} else {
			// 用户存在，更新登录时间
			err = tx.Model(&user).Updates(map[string]interface{}{
				"updated_at": time.Now().UnixMilli(),
				"login_time": time.Now().UnixMilli(),
			}).Error
			if err != nil {
				z.Error("failed to update user login time", zap.Error(err))
			}
		}

		// 初始化用户积分（对已存在积分记录的用户不会重复初始化）
		err = points_core.InitializeUserPoints(c, tx, user.Id)
		if err != nil {
			e := fmt.Errorf("failed to initialize user points: %w, userId: %s", err, user.Id.String())
			msg = "初始化用户积分失败"
			status = -1
			return e
		}

		// 检查用户状态
		if user.Status != "00" {
			e := fmt.Errorf("user is disabled, userId: %s, status: %s", user.Id.String(), user.Status)
			z.Error(e.Error())
			msg = "用户已被禁用"
			status = 1
			return e
		}

		// 创建session
		session, err := sessionStore.Get(c.Request, userSessionKey)
		if err != nil {
			e := fmt.Errorf("failed to get session: %w", err)
			z.Error(e.Error())
			msg = "创建session失败"
			status = -1
			return e
		}

		// 设置session值
		session.Values["user_id"] = user.Id.String()
		session.Values["mobile_phone"] = user.MobilePhone
		session.Values["login_time"] = time.Now().Unix()

		// 保存session
		err = session.Save(c.Request, c.Writer)
		if err != nil {
			e := fmt.Errorf("failed to save session: %w", err)
			z.Error(e.Error())
			msg = "保存session失败"
			status = -1
			return e
		}

		return nil
	})
	if err != nil {
		c.JSON(http.StatusOK, cmn.ReplyProto{
			Status: status,
			Msg:    fmt.Sprintf("登录失败: %s", msg),
		})
		return
	}

	z.Info("user login successful", zap.String("userId", user.Id.String()), zap.String("phone", d.MobilePhone))

	c.JSON(http.StatusOK, cmn.ReplyProto{
		Status: 0,
		Msg:    "登录成功",
	})
	return
}

// HandleGetCurrentUserInfo 处理获取当前用户信息请求
func (h *handler) HandleGetCurrentUserInfo(c *gin.Context) {
	// 获取当前用户ID
	userID, exists := GetCurrentUserID(c)
	if !exists {
		z.Error("failed to get current user ID from context")
		c.JSON(http.StatusOK, cmn.ReplyProto{
			Status: 401,
			Msg:    "用户未登录或登录已过期",
		})
		return
	}

	// 从 VUserInfo 视图查询用户完整信息
	var userInfo cmn.VUserInfo
	err := cmn.GormDB.Where("id = ?", userID).First(&userInfo).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			z.Error("user not found in VUserInfo", zap.String("userID", userID.String()))
			c.JSON(http.StatusOK, cmn.ReplyProto{
				Status: 1,
				Msg:    "用户信息不存在",
			})
			return
		}
		z.Error("failed to query user info from VUserInfo", zap.Error(err), zap.String("userID", userID.String()))
		c.JSON(http.StatusOK, cmn.ReplyProto{
			Status: -1,
			Msg:    "查询用户信息失败",
		})
		return
	}

	// 将用户信息序列化为JSON并返回
	userInfoJSON, err := json.Marshal(userInfo)
	if err != nil {
		z.Error("failed to marshal user info", zap.Error(err), zap.String("userID", userID.String()))
		c.JSON(http.StatusOK, cmn.ReplyProto{
			Status: -1,
			Msg:    "序列化用户信息失败",
		})
		return
	}

	c.JSON(http.StatusOK, cmn.ReplyProto{
		Status: 0,
		Msg:    "获取用户信息成功",
		Data:   userInfoJSON,
	})
	return
}

// HandleGetUserInfoByPhone 根据手机号查询用户信息
func (h *handler) HandleGetUserInfoByPhone(c *gin.Context) {
	// 从query参数获取手机号
	mobilePhone := c.Query("mobilePhone")
	if mobilePhone == "" {
		z.Error("mobile phone is empty")
		c.JSON(http.StatusOK, cmn.ReplyProto{
			Status: 1,
			Msg:    "手机号不能为空",
		})
		return
	}

	// 从 VUserInfo 视图查询用户信息
	var userInfo cmn.VUserInfo
	err := cmn.GormDB.Where("mobile_phone = ?", mobilePhone).First(&userInfo).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			z.Error("user not found by mobile phone", zap.String("mobilePhone", mobilePhone))
			c.JSON(http.StatusOK, cmn.ReplyProto{
				Status: 1,
				Msg:    "用户不存在",
			})
			return
		}
		z.Error("failed to query user info by mobile phone", zap.Error(err), zap.String("mobilePhone", mobilePhone))
		c.JSON(http.StatusOK, cmn.ReplyProto{
			Status: -1,
			Msg:    "查询用户信息失败",
		})
		return
	}

	// 将用户信息序列化为JSON并返回
	userInfoJSON, err := json.Marshal(userInfo)
	if err != nil {
		z.Error("failed to marshal user info", zap.Error(err), zap.String("mobilePhone", mobilePhone))
		c.JSON(http.StatusOK, cmn.ReplyProto{
			Status: -1,
			Msg:    "序列化用户信息失败",
		})
		return
	}

	c.JSON(http.StatusOK, cmn.ReplyProto{
		Status: 0,
		Msg:    "查询用户信息成功",
		Data:   userInfoJSON,
	})
	return
}

// HandleQueryUserInfoList 分页查询用户信息列表
func (h *handler) HandleQueryUserInfoList(c *gin.Context) {
	// 获取分页参数
	pageStr := c.Query("page")
	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	sizeStr := c.Query("pageSize")
	pageSize, err := strconv.Atoi(sizeStr)
	if err != nil || pageSize < 1 {
		pageSize = 10
	}

	// 限制每页最大数量
	if pageSize > 100 {
		pageSize = 100
	}

	// 获取查询关键词（可选）
	keyword := c.Query("keyword")

	// 构建查询条件
	query := cmn.GormDB.Model(&cmn.VUserInfo{})
	if keyword == "raffle-winner" {
		// 只查询奖品数量不为0的用户
		query = query.Where("raffle_prize_count > 0")
	}

	// 查询总记录数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		z.Error("failed to count user info", zap.Error(err))
		c.JSON(http.StatusOK, cmn.ReplyProto{
			Status: -1,
			Msg:    "查询用户总数失败",
		})
		return
	}

	// 分页查询用户信息列表
	var userInfoList []cmn.VUserInfo
	offset := (page - 1) * pageSize
	err = query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&userInfoList).Error
	if err != nil {
		z.Error("failed to query user info list", zap.Error(err))
		c.JSON(http.StatusOK, cmn.ReplyProto{
			Status: -1,
			Msg:    "查询用户信息列表失败",
		})
		return
	}

	// 将响应数据序列化为JSON
	userInfoListJSON, err := json.Marshal(userInfoList)
	if err != nil {
		z.Error("failed to marshal user info list", zap.Error(err))
		c.JSON(http.StatusOK, cmn.ReplyProto{
			Status: -1,
			Msg:    "序列化用户信息列表失败",
		})
		return
	}

	c.JSON(http.StatusOK, cmn.ReplyProto{
		Status:   0,
		Msg:      "查询用户信息列表成功",
		Data:     userInfoListJSON,
		RowCount: total,
	})
}
