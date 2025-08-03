package cmn

import (
	"github.com/google/uuid"
	"gorm.io/datatypes"
)

const (
	UserTableName     = "t_user"     // 用户信息表
	SmsCodesTableName = "t_sms_code" // 短信验证码表

	RaffleWinnersTableName = "t_raffle_winner" // 抽奖获奖者表
	RaffleLogTableName     = "t_raffle_log"    // 抽奖日志表

	AssetArchiveTableName = "t_asset_archive" // 资产档案表
	UserAssetTable        = "t_user_asset"    // 用户资产表

	RankingListConfigTableName = "t_cfg_ranking_list" // 客户端排行榜配置表
	CommonConfigTableName      = "t_cfg_common"       // 通用配置表
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
	return UserTableName
}

// TRaffleWinners 抽奖中奖用户表
type TRaffleWinners struct {
	Id        int       `gorm:"column:id;type:int;primaryKey;autoIncrement"`        // ID
	UserId    uuid.UUID `gorm:"column:user_id;type:uuid;not null;index"`            // 用户ID
	PrizeName string    `gorm:"column:prize_name;type:varchar(100);not null;index"` // 奖品名称
	CreatedAt int64     `gorm:"column:created_at;type:bigint;autoCreateTime:milli"` // 创建时间
	UpdatedAt int64     `gorm:"column:updated_at;type:bigint;autoUpdateTime:milli"` // 更新时间
}

func (TRaffleWinners) TableName() string {
	return RaffleWinnersTableName
}

// RaffleLogTable 用户抽奖日志表
type RaffleLogTable struct {
	Id        int            `gorm:"column:id;type:int;primaryKey;autoIncrement"`        // ID
	UserId    uuid.UUID      `gorm:"column:user_id;type:uuid;not null"`                  // 用户ID
	Count     int            `gorm:"column:count;type:int;default:0"`                    // 抽奖次数
	Prizes    datatypes.JSON `gorm:"column:prizes;type:jsonb"`                           // 获得奖品
	CreatedAt int64          `gorm:"column:created_at;type:bigint;autoCreateTime:milli"` // 创建时间
	UpdatedAt int64          `gorm:"column:updated_at;type:bigint;autoUpdateTime:milli"` // 更新时间
}

func (RaffleLogTable) TableName() string {
	return RaffleLogTableName
}

// TSmsCodes 短信验证码表
type TSmsCodes struct {
	Id          int    `gorm:"column:id;type:int;primaryKey;autoIncrement"`        // ID
	MobilePhone string `gorm:"column:mobile_phone;type:varchar(11);not null"`      // 手机号
	Code        string `gorm:"column:code;type:varchar(10);not null"`              // 验证码
	ExpiresAt   int64  `gorm:"column:expires_at;type:bigint;not null"`             // 验证码过期时间
	CreatedAt   int64  `gorm:"column:created_at;type:bigint;autoCreateTime:milli"` // 创建时间
	UpdatedAt   int64  `gorm:"column:updated_at;type:bigint;autoUpdateTime:milli"` // 更新时间
}

func (TSmsCodes) TableName() string {
	return SmsCodesTableName
}
