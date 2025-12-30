package data

import (
	"fmt"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/ydcloud-dy/opshub/internal/conf"
)

// Data 数据层
type Data struct {
	db *gorm.DB
}

// NewData 创建数据层
func NewData(c *conf.Config) (*Data, error) {
	// 初始化 MySQL
	db, err := newMySQL(c.Database)
	if err != nil {
		return nil, fmt.Errorf("初始化MySQL失败: %w", err)
	}

	return &Data{
		db: db,
	}, nil
}

// newMySQL 创建 MySQL 连接
func newMySQL(cfg conf.DatabaseConfig) (*gorm.DB, error) {
	// GORM 配置 - 禁用外键约束
	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info), // 显示所有SQL
		DisableForeignKeyConstraintWhenMigrating: true, // 迁移时不创建外键约束
	}

	// 连接数据库
	db, err := gorm.Open(mysql.Open(cfg.GetDSN()), gormConfig)
	if err != nil {
		return nil, err
	}

	// 禁用外键检查（只在当前会话）
	db.Exec("SET FOREIGN_KEY_CHECKS = 0;")

	// 获取底层 sql.DB
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	// 设置连接池
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(time.Duration(cfg.ConnMaxLifetime) * time.Second)

	return db, nil
}

// DB 获取数据库连接
func (d *Data) DB() *gorm.DB {
	return d.db
}

// Close 关闭连接
func (d *Data) Close() error {
	sqlDB, err := d.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
