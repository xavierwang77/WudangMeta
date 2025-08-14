package ubanquan_core

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/valyala/fasthttp"
	"go.uber.org/zap"
)

var (
	// 全局token和锁
	globalToken *Token
	tokenMutex  sync.RWMutex
)

// GetGlobalToken 获取全局token（线程安全）
func GetGlobalToken() *Token {
	tokenMutex.RLock()
	defer tokenMutex.RUnlock()
	if globalToken == nil {
		return nil
	}
	// 返回token的副本，避免外部修改
	return &Token{
		AccessToken:  globalToken.AccessToken,
		RefreshToken: globalToken.RefreshToken,
		ExpiresTime:  globalToken.ExpiresTime,
	}
}

// setGlobalToken 设置全局token（线程安全）
func setGlobalToken(token *Token) {
	tokenMutex.Lock()
	defer tokenMutex.Unlock()
	globalToken = token
}

// InitializeToken 初始化token
// 使用appId和appSecret获取初始token
func InitializeToken(ctx context.Context, appId, appSecret string) error {
	// 获取初始token
	token, err := fetchAccessToken(ctx, appId, appSecret)
	if err != nil {
		z.Error("failed to initialize token", zap.Error(err))
		return err
	}

	// 设置全局token
	setGlobalToken(token)

	return nil
}

// StartTokenMaintainer 启动token维护协程
// 在token过期前自动刷新token
func StartTokenMaintainer(ctx context.Context) {
	go func() {
		z.Info("ubanquan token maintainer started")
		ticker := time.NewTicker(30 * time.Second) // 每30秒检查一次
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				z.Info("token maintainer stopped due to context cancellation")
				return
			case <-ticker.C:
				// 检查token是否需要刷新
				if shouldRefreshToken() {
					if err := refreshGlobalToken(ctx); err != nil {
						z.Error("failed to refresh token", zap.Error(err))
						// 如果刷新失败，尝试重新获取token
						if err := reinitializeToken(ctx); err != nil {
							z.Error("failed to reinitialize token", zap.Error(err))
						}
					}
				}
			}
		}
	}()
}

// shouldRefreshToken 检查是否需要刷新token
// 在token过期前5分钟开始刷新
func shouldRefreshToken() bool {
	tokenMutex.RLock()
	defer tokenMutex.RUnlock()

	if globalToken == nil {
		return true
	}

	// 计算token过期时间（毫秒转换为秒）
	expiresAt := time.Unix(globalToken.ExpiresTime/1000, 0)
	// 在过期前5分钟开始刷新
	refreshTime := expiresAt.Add(-5 * time.Minute)

	return time.Now().After(refreshTime)
}

// refreshGlobalToken 刷新全局token
func refreshGlobalToken(ctx context.Context) error {
	tokenMutex.RLock()
	refreshToken := ""
	if globalToken != nil {
		refreshToken = globalToken.RefreshToken
	}
	tokenMutex.RUnlock()

	if refreshToken == "" {
		z.Warn("no refresh token available, cannot refresh")
		return nil
	}

	// 调用刷新token API
	newToken, err := refreshAccessToken(ctx, refreshToken)
	if err != nil {
		return err
	}

	// 更新全局token
	setGlobalToken(newToken)

	return nil
}

// reinitializeToken 重新初始化token
// 当刷新失败时使用appId和appSecret重新获取token
func reinitializeToken(ctx context.Context) error {
	// 使用全局的AppId和AppSecret重新获取token
	token, err := fetchAccessToken(ctx, AppId, AppSecret)
	if err != nil {
		return err
	}

	// 设置全局token
	setGlobalToken(token)

	return nil
}

// fetchAccessToken 获取访问令牌
// 向优版权API发送POST请求获取accessToken
func fetchAccessToken(ctx context.Context, appId string, appSecret string) (*Token, error) {
	// 构建请求体
	reqData := map[string]string{
		"appId":     appId,
		"appSecret": appSecret,
	}

	reqBody, err := json.Marshal(reqData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request data: %w", err)
	}

	// 发送POST请求
	fastReq := fasthttp.AcquireRequest()
	fastResp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(fastReq)
	defer fasthttp.ReleaseResponse(fastResp)

	fastReq.SetRequestURI(fmt.Sprintf("%s/dapp/token", BaseApiUrl))
	fastReq.Header.SetMethod("POST")
	fastReq.Header.SetContentType("application/json")
	fastReq.SetBody(reqBody)

	client := &fasthttp.Client{
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	err = client.Do(fastReq, fastResp)
	if err != nil {
		return nil, fmt.Errorf("failed to send request to ubanquan token API: %w", err)
	}

	// 解析响应
	var tokenResp struct {
		Success bool        `json:"success"`
		Code    interface{} `json:"code"`
		Message interface{} `json:"message"`
		Data    *Token      `json:"data"`
	}

	err = json.Unmarshal(fastResp.Body(), &tokenResp)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal token response: %w", err)
	}

	// 检查API响应状态
	if !tokenResp.Success {
		return nil, fmt.Errorf("ubanquan token API returned error, code: %v, message: %v", tokenResp.Code, tokenResp.Message)
	}

	if tokenResp.Data == nil {
		return nil, fmt.Errorf("ubanquan token API returned empty data")
	}

	return tokenResp.Data, nil
}

// refreshAccessToken 刷新访问令牌
// 向优版权API发送GET请求刷新令牌
func refreshAccessToken(ctx context.Context, refreshToken string) (*Token, error) {
	// 构建请求URL
	url := fmt.Sprintf("%s/dapp/flush?refreshToken=%s", BaseApiUrl, refreshToken)

	// 发送GET请求
	fastReq := fasthttp.AcquireRequest()
	fastResp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(fastReq)
	defer fasthttp.ReleaseResponse(fastResp)

	fastReq.SetRequestURI(url)
	fastReq.Header.SetMethod("GET")

	client := &fasthttp.Client{
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	err := client.Do(fastReq, fastResp)
	if err != nil {
		return nil, fmt.Errorf("failed to send request to ubanquan flush API: %w", err)
	}

	// 解析响应
	var tokenResp struct {
		Success bool        `json:"success"`
		Code    interface{} `json:"code"`
		Message interface{} `json:"message"`
		Data    *Token      `json:"data"`
	}

	err = json.Unmarshal(fastResp.Body(), &tokenResp)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal refresh token response: %w", err)
	}

	// 检查API响应状态
	if !tokenResp.Success {
		return nil, fmt.Errorf("ubanquan flush API returned error, code: %v, message: %v", tokenResp.Code, tokenResp.Message)
	}

	if tokenResp.Data == nil {
		return nil, fmt.Errorf("ubanquan flush API returned empty data")
	}

	return tokenResp.Data, nil
}
