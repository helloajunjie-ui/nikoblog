package handlers

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"nikoblog/internal/models"
)

// CronReloader 自动任务热更新接口。
// 由 cronjob.Manager 实现，注入到 AdminHandler 中，避免 handlers→cronjob→handlers 循环依赖。
type CronReloader interface {
	Reload()
}

// AdminHandler 后台管理处理器（仅 admin 角色可访问）
type AdminHandler struct {
	DB      *gorm.DB
	CronJob CronReloader // 自动任务引擎（热更新用），可为 nil
}

// NewAdminHandler 创建后台管理处理器
func NewAdminHandler(db *gorm.DB, cronJob CronReloader) *AdminHandler {
	return &AdminHandler{DB: db, CronJob: cronJob}
}

// getOrCreateSetting 获取单行 Setting，若不存在则创建默认配置
func (h *AdminHandler) getOrCreateSetting() (*models.Setting, error) {
	var setting models.Setting
	err := h.DB.First(&setting).Error
	if err == nil {
		return &setting, nil
	}
	if err != gorm.ErrRecordNotFound {
		return nil, err
	}
	// 不存在则创建默认配置（单行，ID=1）
	setting = models.Setting{
		ID:            1,
		BlogName:      "nikoblog",
		BlogDesc:      "",
		AllowRegister: true,
		AllowComment:  true,
	}
	if err := h.DB.Create(&setting).Error; err != nil {
		return nil, err
	}
	return &setting, nil
}

// GetSettings 获取博客设置
func (h *AdminHandler) GetSettings(c *gin.Context) {
	setting, err := h.getOrCreateSetting()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "读取设置失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, setting)
}

// UpdateSettingsRequest 更新博客设置请求体
type UpdateSettingsRequest struct {
	BlogName          string `json:"blog_name"`
	BlogDesc          string `json:"blog_desc"`
	AllowRegister     *bool  `json:"allow_register"`
	AllowComment      *bool  `json:"allow_comment"`
	AllowGuestComment *bool  `json:"allow_guest_comment"`
	AiApiUrl          string `json:"ai_api_url"`
	AiApiKey          string `json:"ai_api_key"`
	AiModel           string `json:"ai_model"`
	DealSourceUrl     string `json:"deal_source_url"`
	DealCronExpr      string `json:"deal_cron_expr"`
	AiSystemPrompt    string `json:"ai_system_prompt"`
}

// UpdateSettings 更新博客设置
func (h *AdminHandler) UpdateSettings(c *gin.Context) {
	var req UpdateSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}

	setting, err := h.getOrCreateSetting()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "读取设置失败: " + err.Error()})
		return
	}

	// 记录旧的执行计划，用于热更新判断
	oldExpr := strings.TrimSpace(setting.DealCronExpr)

	// 博客名不能为空
	if strings.TrimSpace(req.BlogName) != "" {
		setting.BlogName = strings.TrimSpace(req.BlogName)
	}
	setting.BlogDesc = req.BlogDesc
	if req.AllowRegister != nil {
		setting.AllowRegister = *req.AllowRegister
	}
	if req.AllowComment != nil {
		setting.AllowComment = *req.AllowComment
	}
	if req.AllowGuestComment != nil {
		setting.AllowGuestComment = *req.AllowGuestComment
	}
	setting.AiApiUrl = strings.TrimSpace(req.AiApiUrl)
	setting.AiApiKey = strings.TrimSpace(req.AiApiKey)
	setting.AiModel = strings.TrimSpace(req.AiModel)
	setting.DealSourceUrl = strings.TrimSpace(req.DealSourceUrl)
	setting.DealCronExpr = strings.TrimSpace(req.DealCronExpr)
	// AI 洗稿 System Prompt：保留用户输入的原始内容（含多行换行），不 Trim 内部空白
	setting.AiSystemPrompt = req.AiSystemPrompt

	if err := h.DB.Save(setting).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存设置失败: " + err.Error()})
		return
	}

	// 若执行计划发生改变，热更新自动任务（无需重启）
	if h.CronJob != nil && setting.DealCronExpr != oldExpr {
		h.CronJob.Reload()
	}

	c.JSON(http.StatusOK, setting)
}

// ListUsers 用户列表（分页）
func (h *AdminHandler) ListUsers(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	var total int64
	if err := h.DB.Model(&models.User{}).Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "统计用户失败: " + err.Error()})
		return
	}

	var users []models.User
	if err := h.DB.Order("id ASC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&users).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询用户失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"total":     total,
		"page":      page,
		"page_size": pageSize,
		"items":     users,
	})
}

// UpdateUserRoleRequest 修改用户角色请求体
type UpdateUserRoleRequest struct {
	Role string `json:"role" binding:"required"`
}

// UpdateUserRole 修改用户角色（user / admin）
func (h *AdminHandler) UpdateUserRole(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的用户ID"})
		return
	}

	var req UpdateUserRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}
	if req.Role != models.RoleUser && req.Role != models.RoleAdmin {
		c.JSON(http.StatusBadRequest, gin.H{"error": "角色只能是 user 或 admin"})
		return
	}

	// 禁止修改自己的角色（防止唯一 admin 被降级）
	currentUserID := c.GetUint("user_id")
	if currentUserID == uint(id) {
		c.JSON(http.StatusForbidden, gin.H{"error": "不能修改自己的角色"})
		return
	}

	var user models.User
	if err := h.DB.First(&user, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}

	user.Role = req.Role
	if err := h.DB.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "修改角色失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, user)
}

// DeleteUser 删除用户（级联删除其博文）
func (h *AdminHandler) DeleteUser(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的用户ID"})
		return
	}

	currentUserID := c.GetUint("user_id")
	if currentUserID == uint(id) {
		c.JSON(http.StatusForbidden, gin.H{"error": "不能删除自己"})
		return
	}

	var user models.User
	if err := h.DB.First(&user, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}

	// 事务：删除用户及其博文
	err = h.DB.Transaction(func(tx *gorm.DB) error {
		// 删除该用户的博文（many2many 关联由 GORM 自动清理）
		if err := tx.Where("user_id = ?", id).Delete(&models.Memo{}).Error; err != nil {
			return err
		}
		return tx.Delete(&user).Error
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除用户失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}

// ListAllMemos 全量博文列表（分页，含作者信息）
func (h *AdminHandler) ListAllMemos(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	var total int64
	if err := h.DB.Model(&models.Memo{}).Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "统计博文失败: " + err.Error()})
		return
	}

	var memos []models.Memo
	if err := h.DB.Preload("User").Preload("Tags").
		Order("CASE WHEN pinned_at IS NOT NULL AND (pin_expire_at IS NULL OR pin_expire_at > CURRENT_TIMESTAMP) THEN 0 ELSE 1 END ASC").
		Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&memos).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询博文失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"total":     total,
		"page":      page,
		"page_size": pageSize,
		"items":     memos,
	})
}

// DeleteMemo 删除任意博文（管理员）
func (h *AdminHandler) DeleteMemo(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的博文ID"})
		return
	}

	var memo models.Memo
	if err := h.DB.First(&memo, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "博文不存在"})
		return
	}

	if err := h.DB.Delete(&memo).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除博文失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}

// PinMemoRequest 置顶博文请求体
type PinMemoRequest struct {
	Pinned   bool   `json:"pinned"`    // true=置顶，false=取消置顶
	ExpireAt string `json:"expire_at"` // 置顶截止时间（RFC3339），留空表示永久置顶
}

// PinMemo 置顶/取消置顶博文（管理员）。
// 支持设置置顶截止时间，到期后自动取消置顶（排序时动态判断）。
func (h *AdminHandler) PinMemo(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的博文ID"})
		return
	}

	var req PinMemoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}

	var memo models.Memo
	if err := h.DB.First(&memo, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "博文不存在"})
		return
	}

	now := time.Now()
	if !req.Pinned {
		// 取消置顶
		if err := h.DB.Model(&memo).Updates(map[string]interface{}{
			"pinned_at":     nil,
			"pin_expire_at": nil,
		}).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "取消置顶失败: " + err.Error()})
			return
		}
	} else {
		// 置顶：解析截止时间（可选）
		var expireAt *time.Time
		if strings.TrimSpace(req.ExpireAt) != "" {
			t, err := time.Parse(time.RFC3339, strings.TrimSpace(req.ExpireAt))
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "截止时间格式错误，需为 RFC3339（如 2026-12-31T23:59:59+08:00）"})
				return
			}
			expireAt = &t
		}
		if err := h.DB.Model(&memo).Updates(map[string]interface{}{
			"pinned_at":     now,
			"pin_expire_at": expireAt,
		}).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "置顶失败: " + err.Error()})
			return
		}
	}

	h.DB.Preload("User").Preload("Tags").First(&memo, memo.ID)
	c.JSON(http.StatusOK, memo)
}

// ListTags 标签列表（全量）
func (h *AdminHandler) ListTags(c *gin.Context) {
	var tags []models.Tag
	if err := h.DB.Order("memo_count DESC").Find(&tags).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询标签失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": tags})
}

// DeleteTag 删除标签（同时清理博文关联）
func (h *AdminHandler) DeleteTag(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的标签ID"})
		return
	}

	var tag models.Tag
	if err := h.DB.First(&tag, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "标签不存在"})
		return
	}

	if err := h.DB.Delete(&tag).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除标签失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}
