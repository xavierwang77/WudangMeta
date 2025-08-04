package router

import (
	"WugongMeta/serve/ubanquan"
	"WugongMeta/serve/user_mgt"

	"github.com/gin-gonic/gin"
)

// InitRoutes 初始化路由
func InitRoutes(r *gin.Engine) {

	userMgtHandler := user_mgt.NewHandler()
	ubanquanHandler := ubanquan.NewHandler()

	// 路由组 /api
	api := r.Group("/api")
	{
		api.GET("/sms-code", userMgtHandler.SendSMSCode)   // 发送短信验证码
		api.POST("/login/by-sms", userMgtHandler.SMSLogin) // 短信验证码登录

		// 需要认证的路由组
		authApi := api.Group("/")
		authApi.Use(user_mgt.AuthMiddleware())
		{
			api.GET("/ubanquan/authentication", ubanquanHandler.Authentication) // 优版权用户授权
			api.PUT("/ubanquan/asset", ubanquanHandler.UpdateMyAsset)           // 更新优版权用户资产
		}
	}
}
