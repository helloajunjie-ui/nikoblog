package database

import (
	"fmt"
	"os"
	"path/filepath"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"nikoblog/internal/config"
	"nikoblog/internal/models"
)

// Init 初始化 SQLite 连接并自动迁移模型
// 开启外键约束，确保数据目录存在
func Init(cfg *config.Config) (*gorm.DB, error) {
	// 确保数据目录存在
	if err := os.MkdirAll(cfg.DataDir, 0o755); err != nil {
		return nil, fmt.Errorf("创建数据目录失败: %w", err)
	}
	if err := os.MkdirAll(cfg.UploadDir, 0o755); err != nil {
		return nil, fmt.Errorf("创建上传目录失败: %w", err)
	}

	// SQLite DSN：开启外键约束
	dsn := fmt.Sprintf("file:%s?_foreign_keys=on", filepath.ToSlash(cfg.DBPath))

	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return nil, fmt.Errorf("打开 SQLite 失败: %w", err)
	}

	// 自动迁移模型
	if err := db.AutoMigrate(&models.User{}, &models.Memo{}, &models.Tag{}, &models.MemoTag{}, &models.Setting{}, &models.Comment{}, &models.CronLog{}); err != nil {
		return nil, fmt.Errorf("自动迁移失败: %w", err)
	}

	return db, nil
}
