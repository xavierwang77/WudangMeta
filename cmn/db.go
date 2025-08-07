package cmn

import (
	"fmt"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormLogger "gorm.io/gorm/logger"
	"time"
)

var (
	GormDB *gorm.DB
)

func InitDB() {
	// 从配置文件中读取数据库连接配置
	host := viper.GetString("dbms.host")
	port := viper.GetString("dbms.port")
	user := viper.GetString("dbms.user")
	pwd := viper.GetString("dbms.pwd")
	dbname := viper.GetString("dbms.db")
	if host == "" || port == "" || user == "" || pwd == "" || dbname == "" {
		logger.Fatal("[ FAIL ] db config not found")
		return
	}

	// 构建连接字符串
	dsn := fmt.Sprintf("user=%v password=%v dbname=%v host=%v port=%v sslmode=disable TimeZone=Asia/Shanghai", user, pwd, dbname, host, port)

	// 初始化数据库连接池
	var err error
	GormDB, err = initDBPool(dsn)
	if err != nil {
		logger.Fatal("[ FAIL ] init db pool failed: " + err.Error())
		return
	}

	// 删除所有视图
	err = dropAllViews(GormDB)
	if err != nil {
		logger.Fatal("[ FAIL ] drop all views failed: " + err.Error())
	}

	// 初始化表
	err = initTable(GormDB)
	if err != nil {
		logger.Fatal("[ FAIL ] init table failed: " + err.Error())
	}

	// 初始化视图
	err = initView(GormDB)
	if err != nil {
		logger.Fatal("[ FAIL ] init view failed: " + err.Error())
	}

	MiniLogger.Info("[ OK ] db module initialed")

	return
}

// 初始化数据库连接池
func initDBPool(dsn string) (*gorm.DB, error) {
	// 连接 Gorm 数据库
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: gormLogger.Default.LogMode(gormLogger.Error),
	})
	if err != nil {
		logger.Error("connect to pg failed: " + err.Error())
		return nil, err
	}

	// 获取底层的 sql.DB
	sqlDB, err := db.DB()
	if err != nil {
		logger.Error("get sql.DB failed: " + err.Error())
		return nil, err
	}

	// 配置连接池
	sqlDB.SetMaxIdleConns(10)             // 最大空闲连接数
	sqlDB.SetMaxOpenConns(100)            // 最大打开连接数
	sqlDB.SetConnMaxLifetime(time.Hour)   // 连接的最大存活时间
	sqlDB.SetConnMaxIdleTime(time.Minute) // 空闲连接的最大存活时间

	// 测试连接
	if err := sqlDB.Ping(); err != nil {
		logger.Error("ping pg failed: " + err.Error())
		return nil, err
	}

	logger.Info("PG pool initialed")

	return db, nil
}

// 初始化表
func initTable(db *gorm.DB) error {
	// 自动迁移
	err := db.AutoMigrate(
		&TUser{},
		&TUserExternal{},
		&TUserPoints{},
		&TSmsCodes{},
		&TRaffleWinners{},
		&TRaffleLog{},
		&TMetaAsset{},
		&TUserAsset{},
		&TUserFortune{},
		&TUserCheckIn{})
	if err != nil {
		logger.Error("auto migrate failed: " + err.Error())
		return err
	}

	logger.Info("PG table initialed")
	return nil
}

// 初始化视图
func initView(db *gorm.DB) error {
	// 创建 v_user_asset_meta 视图
	// 构造一个子查询，把两张表连接并选出所有需要的列
	q := db.
		Table("t_user_asset AS ua").
		Select(`
        ua.id,
        ua.user_id,
		u.mobile_phone,
		u.email,
		u.nick_name,
        ua.meta_asset_id,
        ma.name   AS meta_asset_name,
		ma.value AS meta_asset_value,
        ma.cover_img AS meta_cover_img,
        ua.name,
        ua.theme_name,
        ua.external_id,
        ua.cover_img,
        ua.created_at,
        ua.updated_at
    `).
		Joins("LEFT JOIN t_meta_asset AS ma ON ua.meta_asset_id = ma.id").
		Joins("LEFT JOIN t_user AS u ON ua.user_id = u.id")

	// 创建视图
	err := db.Migrator().CreateView(
		VUserAssetMeta{}.TableName(),
		gorm.ViewOption{Query: q},
	)
	if err != nil {
		logger.Error("create v_user_asset_meta failed: " + err.Error())
		return err
	}

	logger.Info("PG view initialed")

	return nil
}

// 删除当前 schema 中的所有视图
func dropAllViews(db *gorm.DB) error {
	type ViewInfo struct {
		ViewName string
	}

	var views []ViewInfo
	// 查询当前 schema 下所有视图名称
	err := db.Raw(`
		SELECT table_name AS view_name
		FROM information_schema.views
		WHERE table_schema = current_schema()
	`).Scan(&views).Error

	if err != nil {
		logger.Error("failed to query views", zap.Error(err))
		return err
	}

	for _, view := range views {
		logger.Info("Dropping view", zap.String("view", view.ViewName))
		dropSQL := fmt.Sprintf(`DROP VIEW IF EXISTS "%s" CASCADE`, view.ViewName)
		if err := db.Exec(dropSQL).Error; err != nil {
			logger.Error("failed to drop view", zap.String("view", view.ViewName), zap.Error(err))
			return err
		}
	}

	logger.Info("All views dropped successfully", zap.Int("count", len(views)))
	return nil
}
