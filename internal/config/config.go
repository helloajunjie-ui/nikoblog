package config

import (
	"os"
)

// Config 应用配置
type Config struct {
	// Port HTTP 服务监听端口
	Port string
	// DataDir 数据目录（SQLite 文件与上传文件所在目录）
	DataDir string
	// DBPath SQLite 数据库文件路径
	DBPath string
	// JWTSecret JWT 签名密钥
	JWTSecret string
	// UploadDir 上传文件目录
	UploadDir string
	// MaxUploadSize 单文件上传大小上限（字节），默认 5MB
	MaxUploadSize int64
}

// Load 从环境变量加载配置，未设置时使用默认值
func Load() *Config {
	dataDir := getEnv("NIKOBLOG_DATA_DIR", "./data")
	return &Config{
		Port:          getEnv("NIKOBLOG_PORT", "8080"),
		DataDir:       dataDir,
		DBPath:        dataDir + "/nikoblog.db",
		JWTSecret:     getEnv("NIKOBLOG_JWT_SECRET", "nikoblog-dev-secret-change-me"),
		UploadDir:     dataDir + "/uploads",
		MaxUploadSize: 5 * 1024 * 1024, // 5MB
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
