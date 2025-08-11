package raffle

import (
	"WudangMeta/cmn"
	"WudangMeta/serve/user"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type Handler interface {
	HandleDoRaffle(c *gin.Context)
	HandleQueryRaffleWinners(c *gin.Context)
	HandleQueryMyWinnings(c *gin.Context)
	HandleQueryPrizes(c *gin.Context)
	HandleUpdatePrize(c *gin.Context)
	HandleCreatePrize(c *gin.Context)
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

// HandleQueryRaffleWinners 处理分页查询所有中奖用户信息请求
func (h *handler) HandleQueryRaffleWinners(c *gin.Context) {
	// 获取分页参数
	pageStr := c.Query("page")
	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	sizeStr := c.Query("pageSize")
	size, err := strconv.Atoi(sizeStr)
	if err != nil || size < 1 {
		size = 10
	}

	// 限制每页最大数量
	if size > 100 {
		size = 100
	}

	// 计算偏移量
	offset := (page - 1) * size

	// 查询中奖用户信息
	var winners []cmn.VRaffleWinnerInfo
	var total int64

	// 先查询总数
	if err := cmn.GormDB.Model(&cmn.VRaffleWinnerInfo{}).Count(&total).Error; err != nil {
		z.Error("failed to count raffle winners", zap.Error(err))
		c.JSON(http.StatusOK, gin.H{
			"status": -1,
			"msg":    "查询中奖用户总数失败",
		})
		return
	}

	// 分页查询数据
	if err := cmn.GormDB.Model(&cmn.VRaffleWinnerInfo{}).
		Order("created_at DESC").
		Offset(offset).
		Limit(size).
		Find(&winners).Error; err != nil {
		z.Error("failed to query raffle winners", zap.Error(err))
		c.JSON(http.StatusOK, gin.H{
			"status": -1,
			"msg":    "查询中奖用户信息失败",
		})
		return
	}

	// 将响应数据转换为JSON
	winnersJSON, err := json.Marshal(winners)
	if err != nil {
		z.Error("failed to marshal response data", zap.Error(err))
		c.JSON(http.StatusOK, gin.H{
			"status": -1,
			"msg":    "数据序列化失败",
		})
		return
	}

	c.JSON(http.StatusOK, cmn.ReplyProto{
		Status:   0,
		Msg:      "success",
		Data:     winnersJSON,
		RowCount: total,
	})
}

// HandleUpdatePrize 处理修改奖品信息请求
func (h *handler) HandleUpdatePrize(c *gin.Context) {
	// 获取奖品ID
	prizeIdStr := c.Param("id")
	if prizeIdStr == "" {
		c.JSON(http.StatusOK, gin.H{
			"status": 1,
			"msg":    "缺少奖品ID参数",
		})
		return
	}

	prizeId, err := strconv.ParseInt(prizeIdStr, 10, 64)
	if err != nil {
		z.Error("invalid prizeId", zap.Error(err))
		c.JSON(http.StatusOK, gin.H{
			"status": 1,
			"msg":    "奖品ID格式无效",
		})
		return
	}

	// 解析请求体
	var req cmn.ReqProto
	if err := c.ShouldBind(&req); err != nil {
		z.Error("failed to bind request", zap.Error(err))
		c.JSON(http.StatusOK, gin.H{
			"status": 1,
			"msg":    "请求体格式错误",
		})
		return
	}

	var updateData cmn.TRafflePrize
	if err := json.Unmarshal(req.Data, &updateData); err != nil {
		z.Error("failed to bind request data", zap.Error(err))
		c.JSON(http.StatusOK, gin.H{
			"status": 1,
			"msg":    "请求体data字段格式错误",
		})
		return
	}

	// 验证必要字段
	if updateData.Name == "" {
		c.JSON(http.StatusOK, gin.H{
			"status": 1,
			"msg":    "奖品名称不能为空",
		})
		return
	}

	if updateData.Probability < 0 || updateData.Probability > 1 {
		c.JSON(http.StatusOK, gin.H{
			"status": 1,
			"msg":    "奖品概率必须在0到1之间",
		})
		return
	}

	if updateData.TotalCount < 0 {
		c.JSON(http.StatusOK, gin.H{
			"status": 1,
			"msg":    "奖品总数不能为负数",
		})
		return
	}

	if updateData.RemainCount < 0 {
		c.JSON(http.StatusOK, gin.H{
			"status": 1,
			"msg":    "剩余奖品数量不能为负数",
		})
		return
	}

	if updateData.RemainCount > updateData.TotalCount {
		c.JSON(http.StatusOK, gin.H{
			"status": 1,
			"msg":    "剩余奖品数量不能大于总数",
		})
		return
	}

	// 检查奖品是否存在
	var existingPrize cmn.TRafflePrize
	if err := cmn.GormDB.First(&existingPrize, prizeId).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusOK, gin.H{
				"status": 1,
				"msg":    "奖品不存在",
			})
		} else {
			z.Error("failed to query prize", zap.Error(err))
			c.JSON(http.StatusOK, gin.H{
				"status": -1,
				"msg":    "查询奖品失败",
			})
		}
		return
	}

	// 更新奖品信息
	updateData.Id = prizeId
	if err := cmn.GormDB.Model(&existingPrize).Updates(&updateData).Error; err != nil {
		z.Error("failed to update prize", zap.Error(err))
		c.JSON(http.StatusOK, gin.H{
			"status": -1,
			"msg":    "更新奖品信息失败",
		})
		return
	}

	// 如果奖品信息发生变化，需要重新同步到内存奖池
	if err := machine.syncPrizesFromDB(); err != nil {
		z.Error("failed to sync prizes to memory", zap.Error(err))
	}

	c.JSON(http.StatusOK, cmn.ReplyProto{
		Status: 0,
		Msg:    "奖品信息更新成功",
	})
}

// HandleCreatePrize 新增奖品
func (h *handler) HandleCreatePrize(c *gin.Context) {
	// 解析请求体
	var req cmn.ReqProto
	if err := c.ShouldBind(&req); err != nil {
		z.Error("failed to bind request", zap.Error(err))
		c.JSON(http.StatusOK, gin.H{
			"status": 1,
			"msg":    "请求体格式错误",
		})
		return
	}

	var newPrize cmn.TRafflePrize
	if err := json.Unmarshal(req.Data, &newPrize); err != nil {
		z.Error("failed to bind JSON", zap.Error(err))
		c.JSON(http.StatusOK, gin.H{
			"status": 1,
			"msg":    "请求体data字段格式错误",
		})
		return
	}

	// 验证必要字段
	if newPrize.Name == "" {
		c.JSON(http.StatusOK, gin.H{
			"status": 1,
			"msg":    "奖品名称不能为空",
		})
		return
	}

	if newPrize.Probability < 0 || newPrize.Probability > 1 {
		c.JSON(http.StatusOK, gin.H{
			"status": 1,
			"msg":    "奖品概率必须在0到1之间",
		})
		return
	}

	if newPrize.TotalCount < 0 {
		c.JSON(http.StatusOK, gin.H{
			"status": 1,
			"msg":    "奖品总数不能为负数",
		})
		return
	}

	if newPrize.RemainCount < 0 {
		c.JSON(http.StatusOK, gin.H{
			"status": 1,
			"msg":    "剩余奖品数量不能为负数",
		})
		return
	}

	if newPrize.RemainCount > newPrize.TotalCount {
		c.JSON(http.StatusOK, gin.H{
			"status": 1,
			"msg":    "剩余奖品数量不能大于总数",
		})
		return
	}

	if newPrize.Cost < 0 {
		c.JSON(http.StatusOK, gin.H{
			"status": 1,
			"msg":    "奖品成本不能为负数",
		})
		return
	}

	// 设置默认状态
	if newPrize.Status == "" {
		newPrize.Status = "00" // 默认启用
	}

	// 检查奖品名称是否已存在
	var existingPrize cmn.TRafflePrize
	if err := cmn.GormDB.Where("name = ?", newPrize.Name).First(&existingPrize).Error; err == nil {
		c.JSON(http.StatusOK, gin.H{
			"status": 1,
			"msg":    "奖品名称已存在",
		})
		return
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		// 如果不是记录不存在的错误，说明查询出现了其他问题
		z.Error("failed to check existing prize", zap.Error(err))
		c.JSON(http.StatusOK, gin.H{
			"status": -1,
			"msg":    "检查奖品名称失败",
		})
		return
	}

	// 创建新奖品
	if err := cmn.GormDB.Create(&newPrize).Error; err != nil {
		z.Error("failed to create prize", zap.Error(err))
		c.JSON(http.StatusOK, gin.H{
			"status": -1,
			"msg":    "创建奖品失败",
		})
		return
	}

	// 新增奖品后，需要重新同步到内存奖池
	if err := machine.syncPrizesFromDB(); err != nil {
		z.Error("failed to sync prizes to memory", zap.Error(err))
	}

	// 将新奖品数据转换为JSON
	newPrizeJSON, err := json.Marshal(newPrize)
	if err != nil {
		z.Error("failed to marshal new prize data", zap.Error(err))
		c.JSON(http.StatusOK, gin.H{
			"status": -1,
			"msg":    "奖品数据序列化失败",
		})
		return
	}

	c.JSON(http.StatusOK, cmn.ReplyProto{
		Status: 0,
		Msg:    "奖品创建成功",
		Data:   newPrizeJSON,
	})
}

// HandleQueryPrizes 查询所有奖品信息
func (h *handler) HandleQueryPrizes(c *gin.Context) {
	// 获取分页参数
	pageStr := c.Query("page")
	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	sizeStr := c.Query("pageSize")
	size, err := strconv.Atoi(sizeStr)
	if err != nil || size < 1 {
		size = 10
	}

	// 限制每页最大数量
	if size > 100 {
		size = 100
	}

	// 计算偏移量
	offset := (page - 1) * size

	// 获取状态过滤参数（可选）
	status := c.Query("status")

	// 查询所有奖品信息
	var prizes []cmn.TRafflePrize
	var total int64

	// 构建查询条件
	query := cmn.GormDB.Model(&cmn.TRafflePrize{})
	if status != "" {
		query = query.Where("status = ?", status)
	}

	// 先查询总数
	if err := query.Count(&total).Error; err != nil {
		z.Error("failed to count prizes", zap.Error(err))
		c.JSON(http.StatusOK, gin.H{
			"status": -1,
			"msg":    "查询奖品总数失败",
		})
		return
	}

	// 分页查询数据
	if err := query.
		Order("created_at DESC").
		Offset(offset).
		Limit(size).
		Find(&prizes).Error; err != nil {
		z.Error("failed to query prizes", zap.Error(err))
		c.JSON(http.StatusOK, gin.H{
			"status": -1,
			"msg":    "查询奖品信息失败",
		})
		return
	}

	// 将响应数据转换为JSON
	prizesJSON, err := json.Marshal(prizes)
	if err != nil {
		z.Error("failed to marshal response data", zap.Error(err))
		c.JSON(http.StatusOK, gin.H{
			"status": -1,
			"msg":    "数据序列化失败",
		})
		return
	}

	c.JSON(http.StatusOK, cmn.ReplyProto{
		Status:   0,
		Msg:      "success",
		Data:     prizesJSON,
		RowCount: total,
	})
}

// HandleQueryMyWinnings 查询我的中奖信息
func (h *handler) HandleQueryMyWinnings(c *gin.Context) {
	// 获取当前用户ID
	userId, ok := user.GetCurrentUserID(c)
	if !ok {
		z.Error("failed to get current userId from context")
		c.JSON(http.StatusOK, gin.H{
			"status": 401,
			"msg":    "未登录或登录已过期",
		})
		return
	}

	// 获取分页参数
	pageStr := c.Query("page")
	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	sizeStr := c.Query("pageSize")
	size, err := strconv.Atoi(sizeStr)
	if err != nil || size < 1 {
		size = 10
	}

	// 限制每页最大数量
	if size > 100 {
		size = 100
	}

	// 计算偏移量
	offset := (page - 1) * size

	// 查询我的中奖信息
	var myWinnings []cmn.TRaffleWinners
	var total int64

	// 先查询总数
	if err := cmn.GormDB.Model(&cmn.TRaffleWinners{}).Where("user_id = ?", userId).Count(&total).Error; err != nil {
		z.Error("failed to count my winnings", zap.Error(err))
		c.JSON(http.StatusOK, gin.H{
			"status": -1,
			"msg":    "查询我的中奖总数失败",
		})
		return
	}

	// 分页查询数据
	if err := cmn.GormDB.Model(&cmn.TRaffleWinners{}).
		Where("user_id = ?", userId).
		Order("created_at DESC").
		Offset(offset).
		Limit(size).
		Find(&myWinnings).Error; err != nil {
		z.Error("failed to query my winnings", zap.Error(err))
		c.JSON(http.StatusOK, gin.H{
			"status": -1,
			"msg":    "查询我的中奖信息失败",
		})
		return
	}

	// 将响应数据转换为JSON
	winningsJSON, err := json.Marshal(myWinnings)
	if err != nil {
		z.Error("failed to marshal response data", zap.Error(err))
		c.JSON(http.StatusOK, gin.H{
			"status": -1,
			"msg":    "数据序列化失败",
		})
		return
	}

	c.JSON(http.StatusOK, cmn.ReplyProto{
		Status:   0,
		Msg:      "success",
		Data:     winningsJSON,
		RowCount: total,
	})
}
