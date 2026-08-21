package models

import "time"

// Tag 标签模型
type Tag struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"size:64;uniqueIndex;not null" json:"name"`
	MemoCount int       `gorm:"default:0" json:"memo_count"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// MemoTag 博文与标签的中间表（GORM many2many 自动维护）
type MemoTag struct {
	MemoID uint `gorm:"primaryKey"`
	TagID  uint `gorm:"primaryKey"`
}
