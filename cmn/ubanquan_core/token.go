package ubanquan_core

import (
	"context"
	"sync"
	"time"

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
