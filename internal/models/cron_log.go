package models

import "time"

// CronLog 自动任务处理日志（用于持久化去重）。
// SourceURL 建立唯一索引：同一数据源条目只处理一次。
type CronLog struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	SourceURL string    `gorm:"size:512;uniqueIndex;not null" json:"source_url"` // 已处理条目的唯一标识（GUID/Link/Title）
	CreatedAt time.Time `json:"created_at"`
}
