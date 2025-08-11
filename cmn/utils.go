package cmn

import (
	"errors"
	"fmt"
	"math/rand"
	"os"
	"time"

	"gorm.io/gorm"
)

// RandDigits 生成指定位数的随机数字字符串
func RandDigits(length int) string {
	if length <= 0 {
		return ""
	}

	// 创建独立随机数源
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	digits := make([]byte, length)
	for i := 0; i < length; i++ {
		digits[i] = '0' + byte(r.Intn(10)) // 0-9
	}
	return string(digits)
}

// GetDurationUntilNextTargetTime 计算当前时间到下一个指定时间点的间隔
func GetDurationUntilNextTargetTime(hour, minute, second int, locationName string) (time.Duration, error) {
	loc, err := time.LoadLocation(locationName)
	if err != nil {
		return 0, fmt.Errorf("failed to load location %s: %w", locationName, err)
	}

	now := time.Now().In(loc)

	// 构造今天的目标时间
	targetTime := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, second, 0, loc)

	// 如果当前时间已经过了目标时间，则加一天
	if now.After(targetTime) {
		targetTime = targetTime.AddDate(0, 0, 1)
	}

	return targetTime.Sub(now), nil
}

// InitDir 初始化传入的目录路径（如不存在则创建）
// 参数 dir 为目录路径（可以是多层）
// 若成功返回 nil，否则返回错误
func InitDir(dir string) error {
	if dir == "" {
		return fmt.Errorf("target directory path cannot be empty")
	}

	info, err := os.Stat(dir)
	if os.IsNotExist(err) {
		// 不存在就创建
		if mkErr := os.MkdirAll(dir, os.ModePerm); mkErr != nil {
			return fmt.Errorf("failed to create direction: %w", mkErr)
		}
		return nil
	} else if err != nil {
		return fmt.Errorf("failed to check target direction exist: %w", err)
	}

	if !info.IsDir() {
		return fmt.Errorf("target %s exist but not a direction", dir)
	}

	return nil
}

// GetConfigFromDB 从配置表读取配置值，如果不存在则创建默认值
func GetConfigFromDB(key, defaultValue string) (string, error) {
	var config TCfgCommon
	err := GormDB.Where("key = ?", key).First(&config).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// 配置不存在，创建默认配置
			config = TCfgCommon{
				Key:   key,
				Value: defaultValue,
			}
			if err = GormDB.Create(&config).Error; err != nil {
				return "", err
			}
			return defaultValue, nil
		}
		return "", err
	}
	return config.Value, nil
}
