package cmn

import (
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
