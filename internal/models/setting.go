package models

import "time"

// Setting 博客设置（单行配置，存数据库表，可运行时修改）
type Setting struct {
	ID                uint      `gorm:"primaryKey" json:"id"`
	BlogName          string    `gorm:"size:128;not null;default:'nikoblog'" json:"blog_name"`
	BlogDesc          string    `gorm:"size:255" json:"blog_desc"`
	AllowRegister     bool      `gorm:"default:true;not null" json:"allow_register"`       // 是否开放注册
	AllowComment      bool      `gorm:"default:true;not null" json:"allow_comment"`        // 是否开放评论
	AllowGuestComment bool      `gorm:"default:false;not null" json:"allow_guest_comment"` // 是否允许游客（免注册）评论
	AiApiUrl          string    `gorm:"size:255" json:"ai_api_url"`                        // AI 服务 API 地址（OpenAI 兼容）
	AiApiKey          string    `gorm:"size:255" json:"ai_api_key"`                        // AI 服务 API Key
	AiModel           string    `gorm:"size:128" json:"ai_model"`                          // AI 模型名称
	DealSourceUrl     string    `gorm:"size:512" json:"deal_source_url"`                   // 自动任务数据源（RSS 链接）
	DealCronExpr      string    `gorm:"size:64" json:"deal_cron_expr"`                     // 自动任务 cron 表达式（如 0 10,16 * * *）
	AiSystemPrompt    string    `gorm:"type:text" json:"ai_system_prompt"`                 // AI 洗稿 System Prompt（后台可调教，为空时用代码内兜底 Prompt）
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}
