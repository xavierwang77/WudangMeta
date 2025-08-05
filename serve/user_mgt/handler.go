package user_mgt

import (
	"WugongMeta/cmn"
	"WugongMeta/cmn/sms"
	"WugongMeta/serve/points"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"net/http"
	"time"
)

type Handler interface {
	SendSMSCode(c *gin.Context)
	SMSLogin(c *gin.Context)
}

type handler struct {
	smsSrv sms.Service
}

func NewHandler() Handler {
	return &handler{
		smsSrv: sms.NewService(),
	}
}

// SendSMSCode 发送SMS验证码
func (h *handler) SendSMSCode(c *gin.Context) {
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

// SMSLogin 使用短信验证码登录
func (h *handler) SMSLogin(c *gin.Context) {
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
				userId := uuid.New()
				user = cmn.TUser{
					Id:          userId,
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

				// 初始化新用户积分
				err = points.InitializeUserPoints(c, tx, userId)
				if err != nil {
					e := fmt.Errorf("failed to initialize user points: %w, userId: %s", err, userId.String())
					z.Error(e.Error())
					msg = "初始化用户积分失败"
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
		z.Error("transaction failed", zap.Error(err))
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
		Data:   []byte(`{"userId":"` + user.Id.String() + `","mobilePhone":"` + user.MobilePhone + `","nickName":"` + user.NickName + `"}`),
	})
	return
}
