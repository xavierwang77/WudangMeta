package user

import (
	"WudangMeta/cmn"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// AuthMiddleware 用户认证中间件
// 验证用户是否已登录，并将用户信息存储到上下文中
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取session
		session, err := sessionStore.Get(c.Request, userSessionKey)
		if err != nil {
			z.Error("failed to get session", zap.Error(err))
			c.JSON(http.StatusOK, cmn.ReplyProto{
				Status: 401,
				Msg:    "未登录或登录已过期",
			})
			c.Abort()
			return
		}

		// 检查session中是否有用户ID
		userIdStr, ok := session.Values["user_id"].(string)
		if !ok || userIdStr == "" {
			z.Error("user_id not found in session")
			c.JSON(http.StatusOK, cmn.ReplyProto{
				Status: 401,
				Msg:    "未登录或登录已过期",
			})
			c.Abort()
			return
		}

		// 解析用户ID
		userId, err := uuid.Parse(userIdStr)
		if err != nil {
			z.Error("invalid user_id in session", zap.Error(err), zap.String("user_id", userIdStr))
			c.JSON(http.StatusOK, cmn.ReplyProto{
				Status: 401,
				Msg:    "用户信息无效",
			})
			c.Abort()
			return
		}

		// 从数据库查询用户信息
		var user cmn.TUser
		err = cmn.GormDB.Where("id = ?", userId).First(&user).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				z.Error("user not found", zap.String("user_id", userIdStr))
				c.JSON(http.StatusOK, cmn.ReplyProto{
					Status: 401,
					Msg:    "用户不存在",
				})
				c.Abort()
				return
			}
			z.Error("failed to query user", zap.Error(err), zap.String("user_id", userIdStr))
			c.JSON(http.StatusOK, cmn.ReplyProto{
				Status: -1,
				Msg:    "查询用户信息失败",
			})
			c.Abort()
			return
		}

		// 检查用户状态
		if user.Status != "00" {
			z.Error("user is disabled", zap.String("user_id", userIdStr), zap.String("status", user.Status))
			c.JSON(http.StatusOK, cmn.ReplyProto{
				Status: 403,
				Msg:    "用户已被禁用",
			})
			c.Abort()
			return
		}

		// 查询用户外部信息（允许为空）
		var userExternal cmn.TUserExternal
		err = cmn.GormDB.Where("user_id = ?", userId).First(&userExternal).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			// 如果是其他错误（非记录不存在），记录日志但不中断请求
			z.Warn("failed to query user external info", zap.Error(err), zap.String("user_id", userIdStr))
		}

		// 将用户信息存储到上下文中，供后续处理器使用
		c.Set("current_user", user)
		c.Set("user_id", user.Id.String())
		c.Set("mobile_phone", user.MobilePhone)

		// 存储外部用户信息（可能为空）
		if err == nil {
			// 外部信息存在
			c.Set("current_user_external", userExternal)
			c.Set("external_open_id", userExternal.OpenId)
			c.Set("external_nick_name", userExternal.NickName)
			c.Set("external_avatar", userExternal.Avatar)
			z.Debug("user authenticated with external info",
				zap.String("user_id", user.Id.String()),
				zap.String("mobile_phone", user.MobilePhone),
				zap.String("external_open_id", userExternal.OpenId))
		} else {
			// 外部信息不存在，设置为nil
			c.Set("current_user_external", nil)
			c.Set("external_open_id", "")
			c.Set("external_nick_name", "")
			c.Set("external_avatar", "")
		}

		// 继续处理请求
		c.Next()
	}
}

// GetCurrentUser 从上下文中获取当前登录用户信息
// 该函数需要在AuthMiddleware之后使用
func GetCurrentUser(c *gin.Context) (*cmn.TUser, bool) {
	user, exists := c.Get("current_user")
	if !exists {
		return nil, false
	}

	currentUser, ok := user.(cmn.TUser)
	if !ok {
		return nil, false
	}

	return &currentUser, true
}

// GetCurrentUserIDStr 从上下文中获取当前登录用户ID字符串
// 该函数需要在AuthMiddleware之后使用
func GetCurrentUserIDStr(c *gin.Context) (string, bool) {
	userID, exists := c.Get("user_id")
	if !exists {
		return "", false
	}

	userIDStr, ok := userID.(string)
	if !ok {
		return "", false
	}

	return userIDStr, true
}

// GetCurrentUserID 从上下文中获取当前登录用户ID
func GetCurrentUserID(c *gin.Context) (uuid.UUID, bool) {
	userIDStr, exists := c.Get("user_id")
	if !exists {
		return uuid.Nil, false
	}

	userID, ok := userIDStr.(string)
	if !ok {
		return uuid.Nil, false
	}

	id, err := uuid.Parse(userID)
	if err != nil {
		return uuid.Nil, false
	}

	return id, true
}

// GetCurrentUserPhone 从上下文中获取当前登录用户手机号
// 该函数需要在AuthMiddleware之后使用
func GetCurrentUserPhone(c *gin.Context) (string, bool) {
	phone, exists := c.Get("mobile_phone")
	if !exists {
		return "", false
	}

	phoneStr, ok := phone.(string)
	if !ok {
		return "", false
	}

	return phoneStr, true
}

// GetCurrentUserExternal 从上下文中获取当前登录用户的外部信息
// 该函数需要在AuthMiddleware之后使用，返回的外部信息可能为nil
func GetCurrentUserExternal(c *gin.Context) (*cmn.TUserExternal, bool) {
	userExternal, exists := c.Get("current_user_external")
	if !exists {
		return nil, false
	}

	// 检查是否为nil
	if userExternal == nil {
		return nil, true // 存在但为nil，表示用户没有外部信息
	}

	currentUserExternal, ok := userExternal.(cmn.TUserExternal)
	if !ok {
		return nil, false
	}

	return &currentUserExternal, true
}

// GetCurrentUserExternalOpenId 从上下文中获取当前登录用户的外部OpenId
// 该函数需要在AuthMiddleware之后使用
func GetCurrentUserExternalOpenId(c *gin.Context) (string, bool) {
	openId, exists := c.Get("external_open_id")
	if !exists {
		return "", false
	}

	openIdStr, ok := openId.(string)
	if !ok {
		return "", false
	}

	return openIdStr, true
}

// GetCurrentUserExternalNickName 从上下文中获取当前登录用户的外部昵称
// 该函数需要在AuthMiddleware之后使用
func GetCurrentUserExternalNickName(c *gin.Context) (string, bool) {
	nickName, exists := c.Get("external_nick_name")
	if !exists {
		return "", false
	}

	nickNameStr, ok := nickName.(string)
	if !ok {
		return "", false
	}

	return nickNameStr, true
}

// GetCurrentUserExternalAvatar 从上下文中获取当前登录用户的外部头像
// 该函数需要在AuthMiddleware之后使用
func GetCurrentUserExternalAvatar(c *gin.Context) (string, bool) {
	avatar, exists := c.Get("external_avatar")
	if !exists {
		return "", false
	}

	avatarStr, ok := avatar.(string)
	if !ok {
		return "", false
	}

	return avatarStr, true
}

// HasExternalInfo 检查当前用户是否有外部信息
// 该函数需要在AuthMiddleware之后使用
func HasExternalInfo(c *gin.Context) bool {
	userExternal, exists := GetCurrentUserExternal(c)
	if !exists {
		return false
	}
	return userExternal != nil
}
