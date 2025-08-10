package cmn

import (
	"github.com/google/uuid"
	"gorm.io/datatypes"
)

const (
	TUserName         = "t_user"          // 用户信息表
	TUserExternalName = "t_user_external" // 用户外部信息表
	TSmsCodesName     = "t_sms_code"      // 短信验证码表

	TUserPointsName = "t_user_points" // 用户积分表

	TRaffleWinnersName = "t_raffle_winner" // 抽奖获奖者表
	TRaffleLogName     = "t_raffle_log"    // 抽奖日志表

	TMetaAssetName = "t_meta_asset" // 元资产表
	TUserAssetName = "t_user_asset" // 用户资产表

	TRankingListConfigName = "t_cfg_ranking_list" // 客户端排行榜配置表
	TCommonConfigName      = "t_cfg_common"       // 通用配置表

	TUserFortuneName = "t_user_fortune"  // 用户运势表
	TUserCheckInName = "t_user_check_in" // 用户签到表

	VUserAssetMetaName = "v_user_asset_meta" // 用户资产视图
	VUserInfoName      = "v_user_info"       // 用户信息视图
)

// TUser 用户信息表
type TUser struct {
	Id           uuid.UUID `gorm:"column:id;type:uuid;primaryKey;not null;unique;index"` // 用户ID
	OfficialName string    `gorm:"column:official_name;type:varchar(50)"`                // 真实姓名
	NickName     string    `gorm:"column:nick_name;type:varchar(50)"`                    // 昵称
	Email        string    `gorm:"column:email;type:varchar(30)"`                        // 邮箱
	MobilePhone  string    `gorm:"column:mobile_phone;type:varchar(11);uniqueIndex"`     // 手机号
	LoginTime    int64     `gorm:"column:login_time;type:bigint"`                        // 最近登录时间
	CreatedAt    int64     `gorm:"column:created_at;type:bigint;autoCreateTime:milli"`   // 创建时间
	UpdatedAt    int64     `gorm:"column:updated_at;type:bigint;autoUpdateTime:milli"`   // 更新时间
	Status       string    `gorm:"column:status;type:varchar(2);default:'00';index"`     // 用户状态 00:启用 01:禁用
}

func (TUser) TableName() string {
	return TUserName
}

// TUserExternal 用户外部信息表
type TUserExternal struct {
	Id              int64     `gorm:"column:id;type:bigint;primaryKey;autoIncrement"`  // ID
	UserId          uuid.UUID `gorm:"column:user_id;type:uuid;not null;index"`         // 用户ID
	Platform        string    `gorm:"column:platform;type:varchar(30);not null;index"` // 第三方平台标识
	AccessToken     string    `gorm:"column:access_token;type:text"`                   // 第三方平台访问令牌
	RefreshToken    string    `gorm:"column:refresh_token;type:text"`                  // 第三方平台刷新令牌
	TokenExpireTime int64     `gorm:"column:token_expire_time;type:bigint"`            // 第三方平台令牌过期时间
	OpenId          string    `gorm:"column:open_id;type:text;index"`                  // 第三方平台用户ID
	NickName        string    `gorm:"column:nick_name;type:text"`                      // 第三方平台用户昵称
	Avatar          string    `gorm:"column:avatar;type:text"`                         // 第三方平台用户头像

	UserInfo TUser `gorm:"foreignKey:UserId;references:Id;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`
}

func (TUserExternal) TableName() string {
	return TUserExternalName
}

// TRaffleWinners 抽奖中奖用户表
type TRaffleWinners struct {
	Id        int64     `gorm:"column:id;type:bigint;primaryKey;autoIncrement"`     // ID
	UserId    uuid.UUID `gorm:"column:user_id;type:uuid;not null;index"`            // 用户ID
	PrizeName string    `gorm:"column:prize_name;type:varchar(100);not null;index"` // 奖品名称
	CreatedAt int64     `gorm:"column:created_at;type:bigint;autoCreateTime:milli"` // 创建时间
	UpdatedAt int64     `gorm:"column:updated_at;type:bigint;autoUpdateTime:milli"` // 更新时间

	UserInfo TUser `gorm:"foreignKey:UserId;references:Id;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`
}

func (TRaffleWinners) TableName() string {
	return TRaffleWinnersName
}

// TRaffleLog 用户抽奖日志表
type TRaffleLog struct {
	Id        int64          `gorm:"column:id;type:bigint;primaryKey;autoIncrement"`     // ID
	UserId    uuid.UUID      `gorm:"column:user_id;type:uuid;not null"`                  // 用户ID
	Count     int64          `gorm:"column:count;type:bigint;default:0"`                 // 抽奖次数
	Prizes    datatypes.JSON `gorm:"column:prizes;type:jsonb"`                           // 获得奖品
	CreatedAt int64          `gorm:"column:created_at;type:bigint;autoCreateTime:milli"` // 创建时间
	UpdatedAt int64          `gorm:"column:updated_at;type:bigint;autoUpdateTime:milli"` // 更新时间

	UserInfo TUser `gorm:"foreignKey:UserId;references:Id;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`
}

func (TRaffleLog) TableName() string {
	return TRaffleLogName
}

// TSmsCodes 短信验证码表
type TSmsCodes struct {
	Id          int64  `gorm:"column:id;type:bigint;primaryKey;autoIncrement"`     // ID
	MobilePhone string `gorm:"column:mobile_phone;type:varchar(11);not null"`      // 手机号
	Code        string `gorm:"column:code;type:varchar(10);not null"`              // 验证码
	ExpiresAt   int64  `gorm:"column:expires_at;type:bigint;not null"`             // 验证码过期时间
	CreatedAt   int64  `gorm:"column:created_at;type:bigint;autoCreateTime:milli"` // 创建时间
	UpdatedAt   int64  `gorm:"column:updated_at;type:bigint;autoUpdateTime:milli"` // 更新时间
}

func (TSmsCodes) TableName() string {
	return TSmsCodesName
}

// TMetaAsset 元资产表
type TMetaAsset struct {
	Id         int64   `gorm:"column:id;type:bigint;primaryKey;autoIncrement"`     // 元资产ID
	Name       string  `gorm:"column:name;type:text;not null;unique;index"`        // 元资产名称
	CoverImg   string  `gorm:"column:cover_img;type:text"`                         // 元资产图片
	ExternalNo string  `gorm:"column:external_id;type:text"`                       // 元资产外部编号
	Value      float64 `gorm:"column:value;type:float"`                            // 元资产价值
	Platform   string  `gorm:"column:platform;type:text"`                          // 元资产所属平台
	CreatedAt  int64   `gorm:"column:created_at;type:bigint;autoCreateTime:milli"` // 创建时间
	UpdatedAt  int64   `gorm:"column:updated_at;type:bigint;autoUpdateTime:milli"` // 更新时间
}

func (TMetaAsset) TableName() string {
	return TMetaAssetName
}

// TUserAsset 用户资产表
type TUserAsset struct {
	Id          int64     `gorm:"column:id;type:bigint;primaryKey;autoIncrement"`     // ID
	UserId      uuid.UUID `gorm:"column:user_id;type:uuid;not null;index"`            // 用户ID
	MetaAssetId int64     `gorm:"column:meta_asset_id;type:bigint;not null;index"`    // 元资产ID
	Name        string    `gorm:"column:name;type:text;not null;index"`               // 资产名称
	ThemeName   string    `gorm:"column:theme_name;type:text"`                        // 资产主题名称
	ExternalNo  string    `gorm:"column:external_id;type:text"`                       // 资产外部编号
	CoverImg    string    `gorm:"column:cover_img;type:text"`                         // 资产图片
	CreatedAt   int64     `gorm:"column:created_at;type:bigint;autoCreateTime:milli"` // 创建时间
	UpdatedAt   int64     `gorm:"column:updated_at;type:bigint;autoUpdateTime:milli"` // 更新时间

	UserInfo  TUser      `gorm:"foreignKey:UserId;references:Id;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`
	MetaAsset TMetaAsset `gorm:"foreignKey:MetaAssetId;references:Id;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`
}

func (TUserAsset) TableName() string {
	return TUserAssetName
}

// TUserPoints 用户积分表
type TUserPoints struct {
	Id            int64     `gorm:"column:id;type:bigint;primaryKey;autoIncrement"`     // ID
	UserId        uuid.UUID `gorm:"column:user_id;type:uuid;not null;unique;index"`     // 用户ID
	DefaultPoints float64   `gorm:"column:default_points;type:float"`                   // 默认积分
	CreatedAt     int64     `gorm:"column:created_at;type:bigint;autoCreateTime:milli"` // 创建时间
	UpdatedAt     int64     `gorm:"column:updated_at;type:bigint;autoUpdateTime:milli"` // 更新时间

	UserInfo TUser `gorm:"foreignKey:UserId;references:Id;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`
}

func (TUserPoints) TableName() string {
	return TUserPointsName
}

// TUserFortune 用户运势表
type TUserFortune struct {
	UserId    uuid.UUID      `gorm:"column:user_id;type:uuid;primaryKey;not null;index"` // 用户ID
	Name      string         `gorm:"column:name;type:varchar(50)"`                       // 用户姓名
	Gender    string         `gorm:"column:gender;type:varchar(50)"`                     // 用户性别
	Birth     string         `gorm:"column:birth;type:varchar(50)"`                      // 用户生日
	Data      datatypes.JSON `gorm:"column:data;type:jsonb"`                             // 运势数据
	CreatedAt int64          `gorm:"column:created_at;type:bigint;autoCreateTime:milli"` // 创建时间
	UpdatedAt int64          `gorm:"column:updated_at;type:bigint;autoUpdateTime:milli"` // 更新时间

	UserInfo TUser `gorm:"foreignKey:UserId;references:Id;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`
}

func (TUserFortune) TableName() string {
	return TUserFortuneName
}

// TUserCheckIn 用户签到表
type TUserCheckIn struct {
	Id        int64     `gorm:"column:id;type:bigint;primaryKey;autoIncrement"`     // ID
	UserId    uuid.UUID `gorm:"column:user_id;type:uuid;not null;index"`            // 用户ID
	Points    float64   `gorm:"column:points;type:double precision;default:0"`      // 签到获得积分
	CreatedAt int64     `gorm:"column:created_at;type:bigint;autoCreateTime:milli"` // 创建时间

	UserInfo TUser `gorm:"foreignKey:UserId;references:Id;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`
}

func (TUserCheckIn) TableName() string {
	return TUserCheckInName
}

// VUserAssetMeta 用户资产视图
type VUserAssetMeta struct {
	Id             int64     `json:"id" gorm:"column:id"`
	UserId         uuid.UUID `json:"userId" gorm:"column:user_id"`
	MobilePhone    string    `json:"mobilePhone" gorm:"column:mobile_phone"`
	Email          string    `json:"email" gorm:"column:email"`
	NickName       string    `json:"nickName" gorm:"column:nick_name"`
	MetaAssetId    int64     `json:"metaAssetId" gorm:"column:meta_asset_id"`
	MetaAssetName  string    `json:"metaAssetName" gorm:"column:meta_asset_name"`
	MetaAssetValue float64   `json:"metaAssetValue" gorm:"column:meta_asset_value"`
	MetaCoverImg   string    `json:"metaCoverImg" gorm:"column:meta_cover_img"`
	Name           string    `json:"name" gorm:"column:name"`
	ThemeName      string    `json:"themeName" gorm:"column:theme_name"`
	ExternalNo     string    `json:"externalNo" gorm:"column:external_id"`
	CoverImg       string    `json:"coverImg" gorm:"column:cover_img"`
	CreatedAt      int64     `json:"createdAt" gorm:"column:created_at"`
	UpdatedAt      int64     `json:"updatedAt" gorm:"column:updated_at"`
}

func (VUserAssetMeta) TableName() string {
	return VUserAssetMetaName
}

// VUserInfo 用户信息视图
type VUserInfo struct {
	Id               uuid.UUID `json:"id" gorm:"column:id;type:uuid;primaryKey;not null;unique;index"`                   // 用户ID
	OfficialName     string    `json:"officialName" gorm:"column:official_name;type:varchar(50)"`                        // 真实姓名
	NickName         string    `json:"nickName" gorm:"column:nick_name;type:varchar(50)"`                                // 昵称
	Email            string    `json:"email" gorm:"column:email;type:varchar(30)"`                                       // 邮箱
	MobilePhone      string    `json:"mobilePhone" gorm:"column:mobile_phone;type:varchar(11);uniqueIndex"`              // 手机号
	LoginTime        int64     `json:"loginTime" gorm:"column:login_time;type:bigint"`                                   // 最近登录时间
	CreatedAt        int64     `json:"createdAt" gorm:"column:created_at;type:bigint;autoCreateTime:milli"`              // 创建时间
	UpdatedAt        int64     `json:"updatedAt" gorm:"column:updated_at;type:bigint;autoUpdateTime:milli"`              // 更新时间
	Status           string    `json:"status" gorm:"column:status;type:varchar(2);default:'00';index"`                   // 用户状态 00:启用 01:禁用
	ExternalPlatform string    `json:"externalPlatform" gorm:"column:external_platform;type:varchar(30);not null;index"` // 第三方平台标识
	ExternalNickName string    `json:"externalNickName" gorm:"column:external_nick_name;type:text"`                      // 第三方平台用户昵称
	ExternalAvatar   string    `json:"externalAvatar" gorm:"column:external_avatar;type:text"`                           // 第三方平台用户头像
	DefaultPoints    float64   `json:"defaultPoints" gorm:"column:default_points;type:float"`                            // 默认积分
}

func (VUserInfo) TableName() string {
	return VUserInfoName
}
