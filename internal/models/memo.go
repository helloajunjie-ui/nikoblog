package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

// 博文可见性
const (
	VisibilityPublic  = "public"  // 公开，所有人可见
	VisibilityPrivate = "private" // 私密，仅作者可见
)

// StringList 用于存储 JSON 类型的字符串数组（替代繁琐的关联表）
type StringList []string

// Value 实现 driver.Valuer，将 StringList 序列化为 JSON 存储
func (s StringList) Value() (driver.Value, error) {
	if s == nil {
		return "[]", nil
	}
	return json.Marshal(s)
}

// Scan 实现 sql.Scanner，从数据库读取 JSON 到 StringList
func (s *StringList) Scan(value interface{}) error {
	if value == nil {
		*s = StringList{}
		return nil
	}
	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return fmt.Errorf("无法解析 StringList: %T", value)
	}
	return json.Unmarshal(bytes, s)
}

// Memo 博文模型
type Memo struct {
	ID          uint       `gorm:"primaryKey" json:"id"`
	UserID      uint       `gorm:"index;not null" json:"user_id"`
	User        User       `gorm:"foreignKey:UserID" json:"user"`
	Content     string     `gorm:"type:text;not null" json:"content"`
	Images      StringList `gorm:"type:text" json:"images"` // JSON 类型图片列表
	Visibility  string     `gorm:"size:16;default:public;index;not null" json:"visibility"`
	Tags        []Tag      `gorm:"many2many:memo_tags;" json:"tags"`
	PinnedAt    *time.Time `json:"pinned_at"`     // 置顶时间，非空表示已置顶（用于排序）
	PinExpireAt *time.Time `json:"pin_expire_at"` // 置顶截止时间，到期自动取消置顶；NULL 表示永久置顶
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}
