package router

import (
	"WugongMeta/serve/asset"
	"WugongMeta/serve/points"
	"WugongMeta/serve/ranking"
	"WugongMeta/serve/ubanquan"
	"WugongMeta/serve/user_mgt"

	"github.com/gin-gonic/gin"
)

// InitRoutes 初始化路由
func InitRoutes(r *gin.Engine) {

	userMgtHandler := user_mgt.NewHandler()
	ubanquanHandler := ubanquan.NewHandler()
	pointsHandler := points.NewHandler()
	assetHandler := asset.NewHandler()
	rankingHandler := ranking.NewHandler()

	// 路由组 /api
	api := r.Group("/api")
	{
		api.GET("/sms-code", userMgtHandler.HandleSendSMSCode)   // 发送短信验证码
		api.POST("/login/by-sms", userMgtHandler.HandleSMSLogin) // 短信验证码登录

		// 需要认证的路由组
		authApi := api.Group("/")
		authApi.Use(user_mgt.AuthMiddleware())
		{
			authApi.GET("/ubanquan/authentication", ubanquanHandler.HandleAuthentication) // 优版权用户授权
			authApi.PUT("/ubanquan/asset", ubanquanHandler.HandleUpdateMyAsset)           // 更新优版权用户资产
			authApi.GET("/points/me", pointsHandler.HandleQueryMyPoints)                  // 获取我的积分
			authApi.GET("/asset/me", assetHandler.HandleQueryMyAsset)                     // 查询我的资产
			authApi.POST("/ranking/list", rankingHandler.HandleQueryRankingList)          // 查询排行榜列表
		}
	}
}
