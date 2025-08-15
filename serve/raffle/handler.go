package raffle

import (
	"WudangMeta/cmn"
	"WudangMeta/serve/user"
	"encoding/json"
	"errors"
	"fmt"
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
	HandleUpdateConsumePoints(c *gin.Context)
	HandleQueryConsumePoints(c *gin.Context)
	HandleDeletePrizes(c *gin.Context)
	HandleCreateDesignatedUser(c *gin.Context)
	HandleDeleteDesignatedUsers(c *gin.Context)
	HandleQueryDesignatedUsers(c *gin.Context)
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
			"msg":    err.Error(),
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

	// 获取手机号筛选参数（可选）
	mobilePhone := c.Query("mobilePhone")

	// 计算偏移量
	offset := (page - 1) * size

	// 查询中奖用户信息
	var winners []cmn.VRaffleWinnerInfo
	var total int64

	// 构建查询条件
	query := cmn.GormDB.Model(&cmn.VRaffleWinnerInfo{})
	if mobilePhone != "" {
		query = query.Where("mobile_phone = ?", mobilePhone)
	}

	// 先查询总数
	if err = query.Count(&total).Error; err != nil {
		z.Error("failed to count raffle winners", zap.Error(err))
		c.JSON(http.StatusOK, gin.H{
			"status": -1,
			"msg":    "查询中奖用户总数失败",
		})
		return
	}

	// 分页查询数据
	if err = query.
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
	if size > 1000 {
		size = 1000
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
	if err = cmn.GormDB.Model(&cmn.TRaffleWinners{}).
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

// HandleUpdateConsumePoints 更新抽奖消耗积分配置
func (h *handler) HandleUpdateConsumePoints(c *gin.Context) {
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

	var updateData struct {
		ConsumePointsKey   string `json:"consumePointsKey"`
		ConsumePointsValue int64  `json:"consumePointsValue"`
	}
	if err := json.Unmarshal(req.Data, &updateData); err != nil {
		z.Error("failed to unmarshal request data", zap.Error(err))
		c.JSON(http.StatusOK, gin.H{
			"status": 1,
			"msg":    "请求体data字段格式错误",
		})
		return
	}

	// 验证参数
	if updateData.ConsumePointsValue < 0 {
		c.JSON(http.StatusOK, gin.H{
			"status": 1,
			"msg":    "消耗积分不能为负数",
		})
		return
	}

	// 更新数据库配置表中的消耗积分值
	consumePointsValueStr := strconv.FormatInt(updateData.ConsumePointsValue, 10)
	if err := cmn.GormDB.Model(&cmn.TCfgCommon{}).Where("key = ?", cfgKeyConsumePointsValue).Update("value", consumePointsValueStr).Error; err != nil {
		z.Error("failed to update consume points value in config table", zap.Error(err))
		c.JSON(http.StatusOK, gin.H{
			"status": -1,
			"msg":    "更新消耗积分值配置失败",
		})
		return
	}

	// 只在 ConsumePointsKey 不为空时更新配置表中的消耗积分键
	if updateData.ConsumePointsKey != "" {
		if err := cmn.GormDB.Model(&cmn.TCfgCommon{}).Where("key = ?", cfgKeyConsumePointsKey).Update("value", updateData.ConsumePointsKey).Error; err != nil {
			z.Error("failed to update consume points key in config table", zap.Error(err))
			c.JSON(http.StatusOK, gin.H{
				"status": -1,
				"msg":    "更新消耗积分键配置失败",
			})
			return
		}
	}

	// 调用machine的resetConsumePoints方法重置抽奖机消耗积分
	if err := machine.resetConsumePoints(updateData.ConsumePointsKey, updateData.ConsumePointsValue); err != nil {
		z.Error("failed to reset consume points", zap.Error(err))
		c.JSON(http.StatusOK, gin.H{
			"status": -1,
			"msg":    "重置抽奖机消耗积分失败",
		})
		return
	}

	c.JSON(http.StatusOK, cmn.ReplyProto{
		Status: 0,
		Msg:    "抽奖消耗积分配置更新成功",
	})
}

// HandleQueryConsumePoints 查询当前抽奖消耗积分配置
func (h *handler) HandleQueryConsumePoints(c *gin.Context) {
	// 查询消耗积分键配置
	var consumePointsKeyConfig cmn.TCfgCommon
	if err := cmn.GormDB.Where("key = ?", cfgKeyConsumePointsKey).First(&consumePointsKeyConfig).Error; err != nil {
		z.Error("failed to query consume points key config", zap.Error(err))
		c.JSON(http.StatusOK, gin.H{
			"status": -1,
			"msg":    "查询消耗积分键配置失败",
		})
		return
	}

	// 查询消耗积分值配置
	var consumePointsValueConfig cmn.TCfgCommon
	if err := cmn.GormDB.Where("key = ?", cfgKeyConsumePointsValue).First(&consumePointsValueConfig).Error; err != nil {
		z.Error("failed to query consume points value config", zap.Error(err))
		c.JSON(http.StatusOK, gin.H{
			"status": -1,
			"msg":    "查询消耗积分值配置失败",
		})
		return
	}

	// 转换积分值为整数
	consumePointsValue, err := strconv.ParseInt(consumePointsValueConfig.Value, 10, 64)
	if err != nil {
		z.Error("failed to parse consume points value", zap.Error(err))
		c.JSON(http.StatusOK, gin.H{
			"status": -1,
			"msg":    "消耗积分值格式错误",
		})
		return
	}

	// 构造响应数据
	responseData := map[string]interface{}{
		"consumePointsKey":   consumePointsKeyConfig.Value,
		"consumePointsValue": consumePointsValue,
	}

	// 序列化响应数据
	responseJson, err := json.Marshal(responseData)
	if err != nil {
		z.Error("failed to marshal response data", zap.Error(err))
		c.JSON(http.StatusOK, gin.H{
			"status": -1,
			"msg":    "响应数据序列化失败",
		})
		return
	}

	c.JSON(http.StatusOK, cmn.ReplyProto{
		Status: 0,
		Msg:    "查询消耗积分配置成功",
		Data:   responseJson,
	})
}

// HandleDeletePrizes 批量删除奖品
func (h *handler) HandleDeletePrizes(c *gin.Context) {
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

	var deleteData struct {
		PrizeIds []int64 `json:"prizeIds"`
	}
	if err := json.Unmarshal(req.Data, &deleteData); err != nil {
		z.Error("failed to unmarshal request data", zap.Error(err))
		c.JSON(http.StatusOK, gin.H{
			"status": 1,
			"msg":    "请求体data字段格式错误",
		})
		return
	}

	// 验证参数
	if len(deleteData.PrizeIds) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"status": 1,
			"msg":    "奖品ID列表不能为空",
		})
		return
	}

	// 限制批量删除数量
	if len(deleteData.PrizeIds) > 100 {
		c.JSON(http.StatusOK, gin.H{
			"status": 1,
			"msg":    "单次最多删除100个奖品",
		})
		return
	}

	// 检查奖品是否存在
	var existingPrizes []cmn.TRafflePrize
	if err := cmn.GormDB.Where("id IN ?", deleteData.PrizeIds).Find(&existingPrizes).Error; err != nil {
		z.Error("failed to query existing prizes", zap.Error(err))
		c.JSON(http.StatusOK, gin.H{
			"status": -1,
			"msg":    "查询奖品失败",
		})
		return
	}

	// 检查是否所有奖品都存在
	if len(existingPrizes) != len(deleteData.PrizeIds) {
		c.JSON(http.StatusOK, gin.H{
			"status": 1,
			"msg":    "部分奖品不存在",
		})
		return
	}

	// 开启事务进行删除和同步操作
	tx := cmn.GormDB.Begin()
	if tx.Error != nil {
		z.Error("failed to begin transaction", zap.Error(tx.Error))
		c.JSON(http.StatusOK, gin.H{
			"status": -1,
			"msg":    "开启事务失败",
		})
		return
	}

	// 批量删除奖品
	result := tx.Where("id IN ?", deleteData.PrizeIds).Delete(&cmn.TRafflePrize{})
	if result.Error != nil {
		z.Error("failed to delete prizes", zap.Error(result.Error))
		tx.Rollback()
		c.JSON(http.StatusOK, gin.H{
			"status": -1,
			"msg":    "删除奖品失败",
		})
		return
	}

	// 删除奖品后，需要重新同步到内存奖池
	if err := machine.syncPrizesFromDB(); err != nil {
		z.Error("failed to sync prizes to memory", zap.Error(err))
		tx.Rollback()
		c.JSON(http.StatusOK, gin.H{
			"status": -1,
			"msg":    "同步奖品到内存失败，已回滚删除操作",
		})
		return
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		z.Error("failed to commit transaction", zap.Error(err))
		c.JSON(http.StatusOK, gin.H{
			"status": -1,
			"msg":    "提交事务失败",
		})
		return
	}

	c.JSON(http.StatusOK, cmn.ReplyProto{
		Status: 0,
		Msg:    fmt.Sprintf("成功删除%d个奖品", result.RowsAffected),
	})
}

// HandleCreateDesignatedUser 创建抽奖指定获奖用户
func (h *handler) HandleCreateDesignatedUser(c *gin.Context) {
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

	var requestData struct {
		MobilePhone string `json:"mobilePhone"`
		PrizeId     int64  `json:"prizeId"`
	}
	if err := json.Unmarshal(req.Data, &requestData); err != nil {
		z.Error("failed to unmarshal request data", zap.Error(err))
		c.JSON(http.StatusOK, gin.H{
			"status": 1,
			"msg":    "请求体data字段格式错误",
		})
		return
	}

	// 验证必要字段
	if requestData.MobilePhone == "" {
		c.JSON(http.StatusOK, gin.H{
			"status": 1,
			"msg":    "用户手机号不能为空",
		})
		return
	}

	if requestData.PrizeId <= 0 {
		c.JSON(http.StatusOK, gin.H{
			"status": 1,
			"msg":    "奖品ID不能为空",
		})
		return
	}

	// 根据手机号查询用户
	var u cmn.TUser
	if err := cmn.GormDB.First(&u, "mobile_phone = ?", requestData.MobilePhone).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusOK, gin.H{
				"status": 1,
				"msg":    "用户不存在",
			})
		} else {
			z.Error("failed to query user by mobile phone", zap.Error(err))
			c.JSON(http.StatusOK, gin.H{
				"status": -1,
				"msg":    "查询用户失败",
			})
		}
		return
	}

	// 检查奖品是否存在
	var prize cmn.TRafflePrize
	if err := cmn.GormDB.First(&prize, "id = ?", requestData.PrizeId).Error; err != nil {
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

	// 检查是否已存在相同的指定获奖用户记录
	var existingRecord cmn.TRaffleDesignatedUser
	if err := cmn.GormDB.Where("user_id = ? AND prize_id = ?", u.Id, requestData.PrizeId).First(&existingRecord).Error; err == nil {
		c.JSON(http.StatusOK, gin.H{
			"status": 1,
			"msg":    "该用户已被指定为此奖品的获奖者",
		})
		return
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		z.Error("failed to check existing designated user", zap.Error(err))
		c.JSON(http.StatusOK, gin.H{
			"status": -1,
			"msg":    "检查指定获奖用户失败",
		})
		return
	}

	// 创建指定获奖用户记录
	createData := cmn.TRaffleDesignatedUser{
		UserId:  u.Id,
		PrizeId: requestData.PrizeId,
	}
	if err := cmn.GormDB.Create(&createData).Error; err != nil {
		z.Error("failed to create designated user", zap.Error(err))
		c.JSON(http.StatusOK, gin.H{
			"status": -1,
			"msg":    "创建指定获奖用户失败",
		})
		return
	}

	// 将创建的数据转换为JSON
	createdDataJSON, err := json.Marshal(createData)
	if err != nil {
		z.Error("failed to marshal created data", zap.Error(err))
		c.JSON(http.StatusOK, gin.H{
			"status": -1,
			"msg":    "数据序列化失败",
		})
		return
	}

	c.JSON(http.StatusOK, cmn.ReplyProto{
		Status: 0,
		Msg:    "指定获奖用户创建成功",
		Data:   createdDataJSON,
	})
}

// HandleDeleteDesignatedUsers 批量删除抽奖指定获奖用户
func (h *handler) HandleDeleteDesignatedUsers(c *gin.Context) {
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

	var deleteData struct {
		Ids []int64 `json:"ids"`
	}
	if err := json.Unmarshal(req.Data, &deleteData); err != nil {
		z.Error("failed to unmarshal request data", zap.Error(err))
		c.JSON(http.StatusOK, gin.H{
			"status": 1,
			"msg":    "请求体data字段格式错误",
		})
		return
	}

	// 验证参数
	if len(deleteData.Ids) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"status": 1,
			"msg":    "ID列表不能为空",
		})
		return
	}

	// 限制批量删除数量
	if len(deleteData.Ids) > 100 {
		c.JSON(http.StatusOK, gin.H{
			"status": 1,
			"msg":    "单次最多删除100条记录",
		})
		return
	}

	// 检查记录是否存在
	var existingRecords []cmn.TRaffleDesignatedUser
	if err := cmn.GormDB.Where("id IN ?", deleteData.Ids).Find(&existingRecords).Error; err != nil {
		z.Error("failed to query existing designated users", zap.Error(err))
		c.JSON(http.StatusOK, gin.H{
			"status": -1,
			"msg":    "查询指定获奖用户失败",
		})
		return
	}

	// 检查是否所有记录都存在
	if len(existingRecords) != len(deleteData.Ids) {
		c.JSON(http.StatusOK, gin.H{
			"status": 1,
			"msg":    "部分记录不存在",
		})
		return
	}

	// 批量删除记录
	result := cmn.GormDB.Where("id IN ?", deleteData.Ids).Delete(&cmn.TRaffleDesignatedUser{})
	if result.Error != nil {
		z.Error("failed to delete designated users", zap.Error(result.Error))
		c.JSON(http.StatusOK, gin.H{
			"status": -1,
			"msg":    "删除指定获奖用户失败",
		})
		return
	}

	c.JSON(http.StatusOK, cmn.ReplyProto{
		Status: 0,
		Msg:    fmt.Sprintf("成功删除%d条指定获奖用户记录", result.RowsAffected),
	})
}

// HandleQueryDesignatedUsers 分页查询抽奖指定获奖用户
func (h *handler) HandleQueryDesignatedUsers(c *gin.Context) {
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

	// 查询指定获奖用户信息
	var designatedUsers []cmn.VRaffleDesignatedUserPrizeInfo
	var total int64

	// 先查询总数
	if err := cmn.GormDB.Model(&cmn.VRaffleDesignatedUserPrizeInfo{}).Count(&total).Error; err != nil {
		z.Error("failed to count designated users", zap.Error(err))
		c.JSON(http.StatusOK, gin.H{
			"status": -1,
			"msg":    "查询指定获奖用户总数失败",
		})
		return
	}

	// 分页查询数据，按创建时间倒序排列
	if err = cmn.GormDB.Model(&cmn.VRaffleDesignatedUserPrizeInfo{}).
		Order("created_at DESC").
		Offset(offset).
		Limit(size).
		Find(&designatedUsers).Error; err != nil {
		z.Error("failed to query designated users", zap.Error(err))
		c.JSON(http.StatusOK, gin.H{
			"status": -1,
			"msg":    "查询指定获奖用户信息失败",
		})
		return
	}

	// 将响应数据转换为JSON
	designatedUsersJSON, err := json.Marshal(designatedUsers)
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
		Data:     designatedUsersJSON,
		RowCount: total,
	})
}
