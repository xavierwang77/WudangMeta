package router

import (
	"WudangMeta/serve/asset"
	"WudangMeta/serve/points"
	"WudangMeta/serve/raffle"
	"WudangMeta/serve/ranking"
	"WudangMeta/serve/task"
	"WudangMeta/serve/ubanquan"
	"WudangMeta/serve/user"

	"github.com/gin-gonic/gin"
)

// InitRoutes 初始化路由
func InitRoutes(r *gin.Engine) {

	userMgtHandler := user.NewHandler()
	ubanquanHandler := ubanquan.NewHandler()
	pointsHandler := points.NewHandler()
	assetHandler := asset.NewHandler()
	rankingHandler := ranking.NewHandler()
	taskHandler := task.NewHandler()
	raffleHandler := raffle.NewHandler()

	// 路由组 /api
	api := r.Group("/api")
	{
		api.GET("/sms-code", userMgtHandler.HandleSendSMSCode)   // 发送短信验证码
		api.POST("/login/by-sms", userMgtHandler.HandleSMSLogin) // 短信验证码登录

		api.GET("/raffle/winners", raffleHandler.HandleQueryRaffleWinners)                // 查询抽奖获奖者
		api.PUT("/raffle/prize/:id", raffleHandler.HandleUpdatePrize)                     // 更新奖品信息
		api.POST("/raffle/prize", raffleHandler.HandleCreatePrize)                        // 新增奖品
		api.GET("/raffle/prizes", raffleHandler.HandleQueryPrizes)                        // 查询所有奖品信息
		api.DELETE("/raffle/prizes", raffleHandler.HandleDeletePrizes)                    // 删除奖品
		api.PUT("/raffle/config/consume-points", raffleHandler.HandleUpdateConsumePoints) // 更新抽奖消耗积分配置
		api.GET("/raffle/config/consume-points", raffleHandler.HandleQueryConsumePoints)  // 获取抽奖消耗积分配置
		api.GET("/user/info/single", userMgtHandler.HandleGetUserInfoByPhone)             // 获取单个用户信息
		api.GET("/user/info", userMgtHandler.HandleQueryUserInfoList)                     // 获取用户信息列表
		api.GET("/asset/meta", assetHandler.HandleQueryMetaAssets)                        // 查询元数据资产
		api.POST("/ranking/list", rankingHandler.HandleQueryRankingList)                  // 查询排行榜列表
		api.GET("/asset", assetHandler.HandleQueryUserAssetsByPhone)                      // 根据手机号查询用户资产
		api.GET("/raffle/designated-user", raffleHandler.HandleQueryDesignatedUsers)      // 查询指定用户的抽奖信息
		api.POST("/raffle/designated-user", raffleHandler.HandleCreateDesignatedUser)     // 新增指定用户抽奖信息
		api.DELETE("/raffle/designated-user", raffleHandler.HandleDeleteDesignatedUsers)  // 删除指定用户抽奖信息

		// 需要认证的路由组
		authApi := api.Group("/")
		authApi.Use(user.AuthMiddleware())
		{
			authApi.GET("/login-status", userMgtHandler.HandleCheckLoginStatue)           // 检查用户登录状态
			authApi.GET("/ubanquan/authentication", ubanquanHandler.HandleAuthentication) // 优版权用户授权
			authApi.PUT("/ubanquan/asset", ubanquanHandler.HandleUpdateMyAsset)           // 更新优版权用户资产
			authApi.GET("/points/me", pointsHandler.HandleQueryMyPoints)                  // 获取我的积分
			authApi.GET("/asset/me", assetHandler.HandleQueryMyAsset)                     // 查询我的资产
			authApi.POST("/task/fortune", taskHandler.HandleAnalyzeMyFortune)             // 分析我的运势
			authApi.GET("/task/fortune/me", taskHandler.HandleQueryMyFortune)             // 查询我的运势数据
			authApi.PATCH("/task/check-in", taskHandler.HandleDailyCheckIn)               // 每日签到
			authApi.GET("/user/info/me", userMgtHandler.HandleGetCurrentUserInfo)         // 获取当前用户信息
			authApi.GET("/raffle/do", raffleHandler.HandleDoRaffle)                       // 抽奖
			authApi.GET("/raffle/winnings/me", raffleHandler.HandleQueryMyWinnings)       // 查询我的中奖信息
		}
	}
}
