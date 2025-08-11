/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"WudangMeta/cmn"
	"WudangMeta/cmn/llm"
	"WudangMeta/cmn/points_core"
	"WudangMeta/cmn/sms"
	"WudangMeta/router"
	"WudangMeta/serve/asset"
	"WudangMeta/serve/points"
	"WudangMeta/serve/raffle"
	"WudangMeta/serve/ranking"
	"WudangMeta/serve/task"
	"WudangMeta/serve/ubanquan"
	"WudangMeta/serve/user"
	"fmt"

	"github.com/spf13/viper"
	"go.uber.org/zap"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cobra"
)

// serveCmd represents the serve command
var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start all services",
	Long:  `The serve command starts all the services required for the application to run.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("serve called")

		switch debug {
		case true:
			// 设置 Gin 模式为 Debug
			gin.SetMode(gin.DebugMode)
		case false:
			// 设置 Gin 模式为 Release
			gin.SetMode(gin.ReleaseMode)
		default:
			// 设置 Gin 模式为 Release
			gin.SetMode(gin.ReleaseMode)
		}

		// 全局唯一的 Gin 实例
		r := gin.Default()

		r.Use(gin.Logger())
		r.Use(gin.Recovery())

		// 初始化地基模块（顺序不能改变）
		cmn.InitLogger(debug)
		cmn.InitConfig()
		cmn.InitDB()
		logger := cmn.GetLogger()

		// 初始化公共模块
		sms.Init()
		points_core.Init()
		llm.Init()

		// 初始化服务模块
		user.Init()
		asset.Init()
		ubanquan.Init()
		points.Init()
		ranking.Init()
		task.Init()
		raffle.Init()

		cmn.MiniLogger.Info("[ YES ] all modules initialed", zap.String("version", cmn.Version))

		// 引入模块化路由
		router.InitRoutes(r)

		// 读取运行配置
		host := viper.GetString("server.host")
		port := viper.GetString("server.port")

		// 启动服务
		err := r.Run(host + ":" + port)
		if err != nil {
			logger.Error("gin run failed", zap.Error(err))
			return
		}
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)
}
