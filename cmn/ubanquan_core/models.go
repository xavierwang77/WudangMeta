package ubanquan_core

import "github.com/google/uuid"

type Token struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	ExpiresTime  int64  `json:"expiresTime"`
}

// AssetUpdateResult 资产更新结果
type AssetUpdateResult struct {
	UserId       uuid.UUID `json:"userId"`       // 用户ID
	AddedCount   int       `json:"addedCount"`   // 新增资产数量
	SkippedCount int       `json:"skippedCount"` // 跳过资产数量
	TotalCount   int       `json:"totalCount"`   // 总资产数量
	Success      bool      `json:"success"`      // 是否成功
	ErrorMsg     string    `json:"errorMsg"`     // 错误信息
}

// NFRInfo 优版权NFR信息结构
type NFRInfo struct {
	LockTag   int    `json:"lockTag"`
	CD        int    `json:"cd"`
	ThemeName string `json:"themeName"`
	CoverImg  string `json:"coverImg"`
	Name      string `json:"name"`
	AuctionNo string `json:"auctionNo"`
	ProductNo string `json:"productNo"`
	ThemeKey  string `json:"themeKey"`
}

// AssetData 资产数据结构
type AssetData struct {
	NFRInfoList     []NFRInfo `json:"nfrInfoList"`
	MetaProductName string    `json:"metaProductName"`
	MetaProductNo   string    `json:"metaProductNo"`
}

// UbanquanCardResponse 优版权卡片API响应结构
type UbanquanCardResponse struct {
	Code    interface{} `json:"code"`
	Data    []AssetData `json:"data"`
	Message interface{} `json:"message"`
	Success bool        `json:"success"`
}
