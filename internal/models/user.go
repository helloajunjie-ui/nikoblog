package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"
)

// 用户角色
const (
	RoleUser  = "user"  // 普通用户
	RoleAdmin = "admin" // 管理员
)

// SecurityQA 单个密保问答（答案以 bcrypt 哈希存储，不存明文）
type SecurityQA struct {
	Question   string `json:"question"`
	AnswerHash string `json:"answer_hash"`
}

// SecurityQAList 密保问答列表（JSON 存储，支持 1-3 个）
type SecurityQAList []SecurityQA

// Value 实现 driver.Valuer，将列表序列化为 JSON 存储
func (l SecurityQAList) Value() (driver.Value, error) {
	if l == nil {
		return "[]", nil
	}
	b, err := json.Marshal(l)
	return string(b), err
}

// Scan 实现 sql.Scanner，从 JSON 反序列化
func (l *SecurityQAList) Scan(value interface{}) error {
	if value == nil {
		*l = SecurityQAList{}
		return nil
	}
	var b []byte
	switch v := value.(type) {
	case []byte:
		b = v
	case string:
		b = []byte(v)
	default:
		return errors.New("无法解析密保问答类型")
	}
	return json.Unmarshal(b, l)
}

// User 用户模型
type User struct {
	ID                uint           `gorm:"primaryKey" json:"id"`
	Username          string         `gorm:"size:64;uniqueIndex;not null" json:"username"`
	PasswordHash      string         `gorm:"size:255;not null" json:"-"`
	Nickname          string         `gorm:"size:64" json:"nickname"`
	Avatar            string         `gorm:"size:255" json:"avatar"`
	Email             string         `gorm:"size:128;uniqueIndex" json:"email"`
	SecurityQuestions SecurityQAList `gorm:"type:text" json:"security_questions"`
	SecurityFailCount int            `gorm:"default:0" json:"security_fail_count"`
	SecurityLockUntil *time.Time     `json:"security_lock_until"`
	// 登录/改密失败锁定（防暴力破解）：失败计数与锁定截止时间
	LoginFailCount int        `gorm:"default:0" json:"login_fail_count"`
	LoginLockUntil *time.Time `json:"login_lock_until"`
	Role           string     `gorm:"size:16;default:user;not null" json:"role"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}
