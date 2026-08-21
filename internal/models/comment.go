package models

import "time"

// Comment 评论模型
// 登录用户评论：UserID 指向用户，GuestName 为空
// 游客评论：UserID 为 0，GuestName 为游客填写的昵称
type Comment struct {
	ID        uint       `gorm:"primaryKey" json:"id"`
	MemoID    uint       `gorm:"index;not null" json:"memo_id"`
	UserID    *uint      `gorm:"index" json:"user_id"` // NULL 表示游客评论
	User      *User      `gorm:"foreignKey:UserID" json:"user"`
	GuestName string     `gorm:"size:64" json:"guest_name"` // 游客昵称
	Content   string     `gorm:"type:text;not null" json:"content"`
	Images    StringList `gorm:"type:text" json:"images"` // JSON 类型图片列表（评论附图，如错误截图）
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}
