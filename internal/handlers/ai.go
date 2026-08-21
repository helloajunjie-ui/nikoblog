package handlers

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"nikoblog/internal/ai"
	"nikoblog/internal/models"
)

// AIHandler AI 能力处理器（仅 admin 角色可访问）
type AIHandler struct {
	DB *gorm.DB
}

// NewAIHandler 创建 AI 处理器
func NewAIHandler(db *gorm.DB) *AIHandler {
	return &AIHandler{DB: db}
}

// PolishRequest AI 润色请求体
type PolishRequest struct {
	Content string `json:"content"`
}

// polishPrompt 预设的润色 Prompt
const polishPrompt = "你是一个专业的博客编辑，请修正以下文本的错别字，优化排版，并使其更具阅读吸引力。直接输出 Markdown 结果。"

// Polish 调用配置好的 AI 模型对博文内容进行润色
func (h *AIHandler) Polish(c *gin.Context) {
	var req PolishRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}
	content := strings.TrimSpace(req.Content)
	if content == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "内容不能为空"})
		return
	}

	// 读取 AI 配置
	var setting models.Setting
	if err := h.DB.First(&setting).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "读取 AI 配置失败: " + err.Error()})
		return
	}
	if strings.TrimSpace(setting.AiModel) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "AI 服务未配置，请先在后台设置 AI 模型"})
		return
	}

	client := ai.NewClient(setting.AiApiUrl, setting.AiApiKey, setting.AiModel)
	ctx, cancel := context.WithTimeout(c.Request.Context(), 60*time.Second)
	defer cancel()

	result, err := client.ChatCompletion(ctx, []ai.ChatMessage{
		{Role: "system", Content: polishPrompt},
		{Role: "user", Content: content},
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "AI 润色失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"content": result})
}
