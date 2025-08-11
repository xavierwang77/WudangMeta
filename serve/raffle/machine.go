package raffle

import (
	"WudangMeta/cmn"
	"encoding/json"
	"errors"
	"fmt"
	"sync/atomic"

	"github.com/google/uuid"
	"github.com/mroth/weightedrand/v2"
	"go.uber.org/zap"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// Machine 抽奖机
type Machine struct {
	atomicPrizes       atomic.Value // 内存奖池
	consumePointsValue int64        // 单次抽奖消耗积分
	consumePointsKey   string       // 消耗的积分类型
}

func NewMachine(pointsKey string, consumePoints int64) (*Machine, error) {
	if consumePoints < 0 {
		e := fmt.Errorf("consumePointsValue %d < 0", consumePoints)
		return nil, e
	}
	if pointsKey == "" {
		pointsKey = "default_points"
	}

	m := &Machine{
		consumePointsValue: consumePoints,
		consumePointsKey:   pointsKey,
	}

	var emptyPrizes []cmn.TRafflePrize
	m.atomicPrizes.Store(emptyPrizes)

	err := m.syncPrizesFromDB()
	if err != nil {
		z.Error("failed to sync prizes from db", zap.Error(err))
		return nil, err
	}

	return m, nil
}

// syncPrizesFromDB 从数据库同步奖品到内存奖池，只同步剩余数量大于0的奖品
func (m *Machine) syncPrizesFromDB() error {
	var prizes []cmn.TRafflePrize
	err := cmn.GormDB.Find(&prizes).Error
	if err != nil {
		z.Error("failed to query all prizes", zap.Error(err))
		return err
	}

	// 过滤出剩余数量大于0的奖品
	var availablePrizes []cmn.TRafflePrize
	for _, prize := range prizes {
		if prize.RemainCount > 0 {
			availablePrizes = append(availablePrizes, prize)
		}
	}

	if len(availablePrizes) == 0 {
		z.Warn("no available prizes found in the database")
		return nil
	}

	// 更新内存奖池
	m.atomicPrizes.Store(availablePrizes)

	z.Info("synced available prizes from db", zap.Int("total", len(prizes)), zap.Int("available", len(availablePrizes)))
	return nil
}

// 重置单词抽奖消耗积分
func (m *Machine) resetConsumePoints(pointsKey string, pointsValue int64) error {
	if pointsValue < 0 {
		e := fmt.Errorf("consumePointsValue %d < 0", pointsValue)
		return e
	}

	m.consumePointsValue = pointsValue

	if pointsKey != "" {
		m.consumePointsKey = pointsKey
	}

	return nil
}

// 构建奖池（基于概率）
// 所有奖品的概率之和必须<=1.0，否则会导致概率失真
func (m *Machine) buildRafflePoolByProbability() []weightedrand.Choice[string, uint] {
	// 从内存奖池获取奖品列表
	prizes, ok := m.atomicPrizes.Load().([]cmn.TRafflePrize)
	if !ok || len(prizes) == 0 {
		z.Error("build raffle pool error: prizes are empty or invalid")
		return nil
	}

	var choices []weightedrand.Choice[string, uint]
	var totalProbability float64

	for _, prize := range prizes {
		if prize.Probability < 0 || prize.Probability > 1 {
			z.Error("invalid prize probability", zap.Float64("prizeProbability", prize.Probability))
			continue
		}

		choices = append(choices, weightedrand.Choice[string, uint]{
			Item:   prize.Name,
			Weight: uint(prize.Probability * 100000), // 放大精度以支持浮点概率
		})
		totalProbability += prize.Probability
	}

	// 补充“未中奖”选项，确保总概率为1.0（即100%）
	if totalProbability < 1.0 {
		choices = append(choices, weightedrand.Choice[string, uint]{
			Item:   noPrizeSign,
			Weight: uint((1.0 - totalProbability) * 100000),
		})
	}

	return choices
}

// doRaffle 执行抽奖逻辑，支持多次抽奖
func (m *Machine) doRaffle(userId uuid.UUID, raffleCount int64) ([]string, error) {
	// 获取当前奖池
	prizes := m.atomicPrizes.Load().([]cmn.TRafflePrize)
	if len(prizes) == 0 {
		z.Warn("no prizes available for raffle")
		return []string{}, nil
	}

	var results []string

	err := cmn.GormDB.Transaction(func(tx *gorm.DB) error {
		// 查询用户积分是否足够抽奖
		var userPoints float64
		err := tx.Select(m.consumePointsKey).Where("user_id = ?", userId).Scan(&userPoints).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				e := fmt.Errorf("user points not found for user_id %s", userId.String())
				z.Error(e.Error())
				return e
			}
			z.Error("failed to query user points", zap.Error(err), zap.String("user_id", userId.String()))
			return err
		}

		// 检查积分是否足够
		if userPoints < float64(m.consumePointsValue)*float64(raffleCount) {
			e := fmt.Errorf("insufficient points for user_id %s, current points: %.2f, required: %d", userId.String(), userPoints, m.consumePointsValue*raffleCount)
			z.Error(e.Error())
			return e
		}
		remainPoints := userPoints - float64(m.consumePointsValue)*float64(raffleCount)

		// 记录每次抽奖的奖品名
		var prizesWon []string

		// 多次抽奖
		for i := int64(0); i < raffleCount; i++ {
			// 根据概率选择奖品
			choices := m.buildRafflePoolByProbability()
			chooser, err := weightedrand.NewChooser(choices...)
			if err != nil {
				e := fmt.Errorf("failed to create chooser: %w", err)
				z.Error(e.Error())
				return e
			}

			selectedPrizeName := chooser.Pick()
			if selectedPrizeName != noPrizeSign {
				prizesWon = append(prizesWon, selectedPrizeName)
			}

			// 扣除用户积分
			err = tx.Model(&cmn.TUserPoints{}).
				Where("user_id = ?", userId).
				Update(m.consumePointsKey, remainPoints).Error
			if err != nil {
				e := fmt.Errorf("failed to deduct user points: %w", err)
				z.Error(e.Error())
				return e
			}

			// 查找对应的奖品信息
			var selectedPrize *cmn.TRafflePrize
			if selectedPrizeName != noPrizeSign {
				for _, prize := range prizes {
					if prize.Name == selectedPrizeName {
						selectedPrize = &prize
						break
					}
				}
			}

			// 如果中奖（不是 noPrizeSign），则更新奖品剩余数量
			if selectedPrize != nil {
				// 更新奖品剩余数量
				err = tx.Model(&cmn.TRafflePrize{}).Where("id = ?", selectedPrize.Id).Update("remain_count", gorm.Expr("remain_count - 1")).Error
				if err != nil {
					z.Error("failed to update prize remain count", zap.Error(err), zap.Int64("prize_id", selectedPrize.Id))
					return err
				}

				// 添加中奖记录
				winner := cmn.TRaffleWinners{
					UserId:    userId,
					PrizeName: selectedPrize.Name,
				}
				err = tx.Create(&winner).Error
				if err != nil {
					z.Error("failed to create winner record", zap.Error(err), zap.String("user_id", userId.String()), zap.String("prize_name", selectedPrize.Name))
					return err
				}
			}
		}

		// 创建抽奖日志
		var prizeDataJson []byte
		if len(prizesWon) > 0 {
			// 只记录奖品名数组，如果没有中奖，则记录空数组
			prizeDataJson, err = json.Marshal(prizesWon)
		} else {
			prizeDataJson, err = json.Marshal([]string{})
		}

		if err != nil {
			z.Error("failed to marshal prize data", zap.Error(err))
			return err
		}

		raffleLog := cmn.TRaffleLog{
			UserId: userId,
			Count:  raffleCount,
			Prizes: datatypes.JSON(prizeDataJson),
		}
		err = tx.Create(&raffleLog).Error
		if err != nil {
			z.Error("failed to create raffle log", zap.Error(err), zap.String("user_id", userId.String()))
			return err
		}

		// 如果抽中了奖品，则重新同步奖品到内存奖池
		if len(prizesWon) > 0 {
			err = m.syncPrizesFromDB()
			if err != nil {
				z.Error("failed to sync prizes from db after raffle", zap.Error(err))
				return err
			}
		}

		return nil
	})

	if err != nil {
		z.Error("raffle transaction failed", zap.Error(err), zap.String("user_id", userId.String()))
		return []string{}, err
	}

	// 返回所有中奖的奖品名
	return results, nil
}
