package raffle

import (
	"WudangMeta/cmn"
	"WudangMeta/serve/user"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type Handler interface {
	HandleDoRaffle(c *gin.Context)
}

type handler struct {
}

func NewHandler() Handler {
	return &handler{}
}

// HandleDoRaffle 处理抽奖请求
func (h *handler) HandleDoRaffle(c *gin.Context) {
	userId, ok := user.GetCurrentUserID(c)
	if !ok {
		z.Error("failed to get current userId from context")
		c.JSON(http.StatusOK, gin.H{
			"status": 401,
			"msg":    "未登录或登录已过期",
		})
		return
	}

	raffleCountStr := c.Query("raffleCount")
	if raffleCountStr == "" {
		z.Error("raffleCount is required")
		c.JSON(http.StatusOK, gin.H{
			"status": 1,
			"msg":    "缺少 raffleCount 参数",
		})
		return
	}

	raffleCount, err := strconv.ParseInt(raffleCountStr, 10, 64)
	if err != nil {
		z.Error("invalid raffleCountStr", zap.Error(err))
		c.JSON(http.StatusOK, gin.H{
			"status": 1,
			"msg":    "raffleCountStr 参数无效，无法转换为整数",
		})
		return
	}

	if raffleCount <= 0 || raffleCount > 10 {
		z.Error("raffleCount must be between 1 and 10", zap.Int64("raffleCount", raffleCount))
		c.JSON(http.StatusOK, gin.H{
			"status": 1,
			"msg":    "抽奖次数必须在 1 到 10 之间",
		})
		return
	}

	prizes, err := machine.doRaffle(userId, raffleCount)
	if err != nil {
		z.Error("failed to perform raffle", zap.Error(err))
		c.JSON(http.StatusOK, gin.H{
			"status": -1,
			"msg":    "抽奖失败，请稍后再试",
		})
		return
	}

	prizesJson, err := json.Marshal(prizes)
	if err != nil {
		z.Error("failed to marshal prizes", zap.Error(err))
		c.JSON(http.StatusOK, gin.H{
			"status": -1,
			"msg":    "抽奖结果序列化失败",
		})
		return
	}

	c.JSON(http.StatusOK, cmn.ReplyProto{
		Status: 0,
		Msg:    "success",
		Data:   prizesJson,
	})
	return
}
