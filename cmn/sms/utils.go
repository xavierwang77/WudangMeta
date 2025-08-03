package sms

import (
	"io"
	"net/http"
	"net/url"
	"regexp"
)

// Post 方式发起网络请求 ,params 是url.Values类型
func Post(apiURL string, params url.Values) (rs []byte, err error) {
	resp, err := http.PostForm(apiURL, params)
	if err != nil {
		return nil, err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			z.Error("close response body failed: " + err.Error())
			return
		}
	}(resp.Body)
	return io.ReadAll(resp.Body)
}

// IsValidPhone 验证手机号是否合法
func IsValidPhone(phone string) bool {
	regex := regexp.MustCompile(`^1[3-9]\d{9}$`)
	return regex.MatchString(phone)
}
