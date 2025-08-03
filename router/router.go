package router

import (
	"WugongMeta/serve/user_mgt"

	"github.com/gin-gonic/gin"
)

// InitRoutes 初始化路由
func InitRoutes(r *gin.Engine) {

	userMgtHandler := user_mgt.NewHandler()

	// 路由组 /api
	api := r.Group("/api")
	{
		api.GET("/sms-code", userMgtHandler.SendSMSCode)
		api.POST("/login/by-sms", userMgtHandler.SMSLogin)
	}
}
