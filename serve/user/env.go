package user

import (
	"WudangMeta/cmn"
	"fmt"
	"net/http"

	"github.com/gorilla/sessions"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

const (
	smsCodeLength  = 6              // 短信验证码长度
	userSessionKey = "user-session" // 用户session的cookie名称
)

var (
	sessionStore *sessions.CookieStore
)

var z *zap.Logger

func Init() {
	z = cmn.GetLogger()

	err := initSessionStore()
	if err != nil {
		z.Fatal("[ FAIL ] failed to initialize session store", zap.Error(err))
	}

	cmn.MiniLogger.Info("[ OK ] user_mgt module initialized")
}

func initSessionStore() error {
	authKeyStr := viper.GetString("session.authKey")
	if authKeyStr == "" {
		return fmt.Errorf("gorilla session store key is empty")
	}
	encryptionKeyStr := viper.GetString("session.encryptionKey")
	if encryptionKeyStr == "" {
		return fmt.Errorf("gorilla session store encryption key is empty")
	}

	authKey := []byte(authKeyStr)
	encryptionKey := []byte(encryptionKeyStr)

	// 创建session store，配置需要与handler中保持一致
	sessionStore = sessions.NewCookieStore(authKey, encryptionKey)
	sessionStore.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   86400 * 30, // 30天
		HttpOnly: true,
		Secure:   false, // 开发环境设为false，生产环境应设为true
		SameSite: http.SameSiteLaxMode,
	}

	return nil
}
