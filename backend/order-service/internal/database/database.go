package database

import (
	"fmt"

	"order-service/internal/config"
	"order-service/internal/model"

	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Database struct {
	DB *gorm.DB
}

// 初始化数据库连接
func NewDatabase(cfg *config.DatabaseConfig) (*Database, error) {
	var db *gorm.DB
	var err error

	// GORM 配置
	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	}

	switch cfg.Driver {
	case "postgres":
		dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s TimeZone=%s",
			cfg.Postgres.Host,
			cfg.Postgres.Port,
			cfg.Postgres.User,
			cfg.Postgres.Password,
			cfg.Postgres.DBName,
			cfg.Postgres.SSLMode,
			cfg.Postgres.TimeZone,
		)
		db, err = gorm.Open(postgres.Open(dsn), gormConfig)
	case "mysql":
		dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=%t&loc=%s",
			cfg.MySQL.User,
			cfg.MySQL.Password,
			cfg.MySQL.Host,
			cfg.MySQL.Port,
			cfg.MySQL.DBName,
			cfg.MySQL.Charset,
			cfg.MySQL.ParseTime,
			cfg.MySQL.Loc,
		)
		db, err = gorm.Open(mysql.Open(dsn), gormConfig)
	default:
		return nil, fmt.Errorf("unsupported database driver: %s", cfg.Driver)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// 配置连接池
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database instance: %w", err)
	}

	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	return &Database{DB: db}, nil
}

// 自动迁移数据库表
func (d *Database) AutoMigrate() error {
	err := d.DB.AutoMigrate(
		&model.Order{},
		&model.OrderItem{},
		&model.OrderFailure{},
		&model.OrderIdempotency{},
		&model.OrderStats{},
	)
	if err != nil {
		return fmt.Errorf("failed to auto migrate: %w", err)
	}

	// 创建唯一索引
	if err := model.CreateUniqueIndexes(d.DB); err != nil {
		return fmt.Errorf("failed to create unique indexes: %w", err)
	}

	return nil
}

// 健康检查
func (d *Database) HealthCheck() error {
	sqlDB, err := d.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Ping()
}

// 关闭数据库连接
func (d *Database) Close() error {
	sqlDB, err := d.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// 开始事务
func (d *Database) BeginTx() *gorm.DB {
	return d.DB.Begin()
}

// 获取数据库实例
func (d *Database) GetDB() *gorm.DB {
	return d.DB
}
