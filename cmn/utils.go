package cmn

import (
	"fmt"
	"math/rand"
	"time"
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
