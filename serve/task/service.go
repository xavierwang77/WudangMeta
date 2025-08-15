package task

import (
	"WudangMeta/cmn"
	"WudangMeta/cmn/llm"
	"WudangMeta/cmn/points_core"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// QueryExistFortuneUserData 查询已存在的运势用户数据
func QueryExistFortuneUserData(ctx context.Context) ([]UserData, error) {
	var records []cmn.TUserFortune
	if err := cmn.GormDB.Select("user_id", "name", "gender", "birth").
		Find(&records).Error; err != nil {
		z.Error("failed to query luck tendency source data", zap.Error(err))
		return nil, err
	}

	// 构建返回值
	result := make([]UserData, 0, len(records))
	for _, rec := range records {
		data := UserData{
			UserId: rec.UserId,
			Name:   rec.Name,
			Gender: rec.Gender,
			Birth:  rec.Birth,
		}
		result = append(result, data)
	}

	return result, nil
}

// AnalyzeFortune 分析用户的今日运势
// 返回: 运势分析数据、错误
func AnalyzeFortune(ctx context.Context, name, gender, birth string) (Fortune, error) {
	if name == "" || gender == "" || birth == "" {
		z.Error("name or gender or birth is empty")
		return Fortune{}, fmt.Errorf("name or gender or birth is empty")
	}

	llmService := llm.NewService()

	prompt := llmPrompt
	prompt.UserInfo.Name = name
	prompt.UserInfo.Gender = gender
	prompt.UserInfo.Birth = birth

	// 生成提示此字符串
	promptStr, err := prompt.ToJSONString()
	if err != nil {
		z.Error("failed to convert prompt to JSON string", zap.Error(err))
		return Fortune{}, err
	}

	// llm对话
	output, err := llmService.Chat(promptStr)
	if err != nil {
		return Fortune{}, err
	}

	// 解析llm的输出
	outputFormatted, err := ParseLlmOutputFormatWithMarkdown(output)
	if err != nil {
		z.Error("failed to parse Llm output format", zap.Error(err))
		return Fortune{}, err
	}

	return *outputFormatted, nil
}

// AnalyzeAndSaveFortune 分析用户的今日运势并保存到数据库
// 返回: 运势分析数据、此次分析增加的积分、错误
func AnalyzeAndSaveFortune(ctx context.Context, db *gorm.DB, userId uuid.UUID, name, gender, birth string) (Fortune, float64, error) {
	if userId == uuid.Nil || name == "" || gender == "" || birth == "" {
		e := fmt.Errorf("invalid userId or name or gender or both: %s", userId.String())
		z.Error(e.Error())
		return Fortune{}, 0, e
	}
	if db == nil {
		db = cmn.GormDB
	}

	fortune, err := AnalyzeFortune(ctx, name, gender, birth)
	if err != nil {
		return Fortune{}, 0, err
	}

	// 检查用户今天是否已经分析过运势
	now := time.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	todayStartMilli := todayStart.UnixMilli()
	tomorrowStart := todayStart.AddDate(0, 0, 1)
	tomorrowStartMilli := tomorrowStart.UnixMilli()
	var todayRowCount int64
	err = cmn.GormDB.Model(&cmn.TUserFortune{}).
		Where("user_id = ? AND updated_at >= ? AND updated_at < ?", userId, todayStartMilli, tomorrowStartMilli).
		Count(&todayRowCount).Error
	if err != nil {
		e := fmt.Errorf("failed to check user today fortune: %w", err)
		z.Error(e.Error(), zap.String("userId", userId.String()))
		return Fortune{}, 0, err
	}

	// 检查用户是否存在运势记录（用于判断是插入还是更新）
	var anyTimeRowCount int64
	err = cmn.GormDB.Model(&cmn.TUserFortune{}).
		Where("user_id = ?", userId).
		Count(&anyTimeRowCount).Error
	if err != nil {
		e := fmt.Errorf("failed to check user luck tendency: %w", err)
		z.Error(e.Error(), zap.String("userId", userId.String()))
		return Fortune{}, 0, err
	}

	userData := UserData{
		UserId: userId,
		Name:   name,
		Gender: gender,
		Birth:  birth,
	}

	var addedPoints float64 = 0

	if anyTimeRowCount > 0 {
		// 更新用户的运势记录
		err = UpdateFortune(ctx, db, userData, fortune)
		if err != nil {
			return Fortune{}, 0, err
		}
	} else {
		// 增加用户的运势记录
		err = InsertFortune(ctx, db, userData, fortune)
		if err != nil {
			return Fortune{}, 0, err
		}
	}

	// 只有当天第一次分析运势才增加积分
	if todayRowCount == 0 {
		err = points_core.AddUserPoints(ctx, db, userId, luckTendencyScore)
		if err != nil {
			return Fortune{}, 0, err
		}
		addedPoints = luckTendencyScore
	}

	return fortune, addedPoints, nil
}

// InsertFortune 插入用户运势倾向记录
func InsertFortune(ctx context.Context, db *gorm.DB, userData UserData, tendency Fortune) error {
	if userData.UserId == uuid.Nil {
		e := fmt.Errorf("userId is empty")
		z.Error(e.Error())
		return e
	}
	if db == nil {
		db = cmn.GormDB
	}

	jsonData, err := json.Marshal(tendency)
	if err != nil {
		z.Error("failed to marshal luck tendency", zap.String("userId", userData.UserId.String()), zap.Error(err))
		return err
	}

	record := cmn.TUserFortune{
		UserId: userData.UserId,
		Name:   userData.Name,
		Gender: userData.Gender,
		Birth:  userData.Birth,
		Data:   jsonData,
	}

	err = db.Model(&cmn.TUserFortune{}).Create(&record).Error
	if err != nil {
		z.Error("failed to create luck tendency", zap.String("userId", userData.UserId.String()), zap.Error(err))
		return err
	}

	return nil
}

// UpdateFortune 更新用户运势倾向记录
func UpdateFortune(ctx context.Context, db *gorm.DB, userData UserData, tendency Fortune) error {
	if userData.UserId == uuid.Nil {
		e := fmt.Errorf("userId is empty")
		z.Error(e.Error())
		return e
	}
	if db == nil {
		db = cmn.GormDB
	}

	jsonData, err := json.Marshal(tendency)
	if err != nil {
		z.Error("failed to marshal luck tendency", zap.String("userId", userData.UserId.String()), zap.Error(err))
		return err
	}

	err = db.Model(&cmn.TUserFortune{}).
		Where("user_id = ?", userData.UserId).
		Updates(map[string]interface{}{
			"name":       userData.Name,
			"gender":     userData.Gender,
			"birth":      userData.Birth,
			"data":       jsonData,
			"updated_at": time.Now().UnixMilli(),
		}).Error
	if err != nil {
		z.Error("failed to update luck tendency", zap.String("userId", userData.UserId.String()), zap.Error(err))
		return err
	}

	return nil
}

// RefreshAllUsersFortune 刷新所有用户的运势倾向数据
func RefreshAllUsersFortune(ctx context.Context) error {
	// 获取所有已存在运势数据用户的刷新源数据
	userData, err := QueryExistFortuneUserData(ctx)
	if err != nil {
		return err
	}

	for _, data := range userData {
		// 获取当前用户的新运势数据
		luckTendency, err := AnalyzeFortune(ctx, data.Name, data.Gender, data.Birth)
		if err != nil {
			z.Error("failed to analyze luck tendency data", zap.Error(err))
			continue // 继续处理下一个用户
		}
		// 更新用户的运势记录
		err = UpdateFortune(ctx, nil, data, luckTendency)
		if err != nil {
			z.Error("failed to update luck tendency data", zap.String("userId", data.UserId.String()), zap.Error(err))
			continue // 继续处理下一个用户
		}
	}

	return nil
}
