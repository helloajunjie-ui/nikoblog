package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"nikoblog/internal/models"
)

// CommentHandler 评论相关处理器
type CommentHandler struct {
	DB *gorm.DB
}

// NewCommentHandler 创建评论处理器
func NewCommentHandler(db *gorm.DB) *CommentHandler {
	return &CommentHandler{DB: db}
}

// getCommentSetting 读取评论相关设置（AllowComment / AllowGuestComment）
// 若 Setting 不存在则返回默认值（允许评论、不允许游客评论）
func (h *CommentHandler) getCommentSetting() (allowComment, allowGuestComment bool) {
	var setting models.Setting
	err := h.DB.First(&setting).Error
	if err != nil {
		// 无设置行时按默认值处理：允许评论、不允许游客评论
		return true, false
	}
	return setting.AllowComment, setting.AllowGuestComment
}

// GetCommentSettings 获取评论相关设置（公开，供前端判断游客是否可评论）
func (h *CommentHandler) GetCommentSettings(c *gin.Context) {
	allowComment, allowGuestComment := h.getCommentSetting()
	c.JSON(http.StatusOK, gin.H{
		"allow_comment":       allowComment,
		"allow_guest_comment": allowGuestComment,
	})
}

// ListComments 获取某篇博文的评论列表（按时间正序）
func (h *CommentHandler) ListComments(c *gin.Context) {
	memoID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的博文 ID"})
		return
	}

	// 校验博文存在
	var memo models.Memo
	if err := h.DB.First(&memo, memoID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "博文不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}

	var comments []models.Comment
	if err := h.DB.Where("memo_id = ?", memoID).
		Preload("User").
		Order("created_at ASC").
		Find(&comments).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询评论失败"})
		return
	}

	c.JSON(http.StatusOK, comments)
}

// CreateCommentRequest 发表评论请求体
type CreateCommentRequest struct {
	Content   string   `json:"content" binding:"required"`
	GuestName string   `json:"guest_name"` // 游客昵称（游客评论时必填）
	Images    []string `json:"images"`     // 评论附图 URL 列表（可选）
}

// CreateComment 发表评论
// 登录用户：使用 UserID 关联；游客：需 AllowGuestComment 开启且填写 GuestName
func (h *CommentHandler) CreateComment(c *gin.Context) {
	memoID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的博文 ID"})
		return
	}

	// 校验博文存在
	var memo models.Memo
	if err := h.DB.First(&memo, memoID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "博文不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}

	var req CreateCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}

	req.Content = strings.TrimSpace(req.Content)
	if req.Content == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "评论内容不能为空"})
		return
	}

	allowComment, allowGuestComment := h.getCommentSetting()
	if !allowComment {
		c.JSON(http.StatusForbidden, gin.H{"error": "博主已关闭评论功能"})
		return
	}

	// 判断是否登录用户
	userID := c.GetUint("user_id")
	comment := models.Comment{
		MemoID:  uint(memoID),
		Content: req.Content,
		Images:  models.StringList(req.Images),
	}

	if userID > 0 {
		// 登录用户评论
		comment.UserID = &userID
		// 登录用户可附图，但限制最多 5 张
		if len(comment.Images) > 5 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "评论图片最多 5 张"})
			return
		}
	} else {
		// 游客评论：需开启免注册评论且填写昵称
		if !allowGuestComment {
			c.JSON(http.StatusForbidden, gin.H{"error": "博主未开放游客评论，请先登录"})
			return
		}
		req.GuestName = strings.TrimSpace(req.GuestName)
		if req.GuestName == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "游客评论请填写昵称"})
			return
		}
		if len([]rune(req.GuestName)) > 64 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "昵称过长（最多 64 字符）"})
			return
		}
		comment.GuestName = req.GuestName
		// 游客不允许上传图片（防滥用），忽略传入的图片
		comment.Images = nil
	}

	if err := h.DB.Create(&comment).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "发表评论失败"})
		return
	}

	// 返回完整评论（含用户信息）
	h.DB.Preload("User").First(&comment, comment.ID)
	c.JSON(http.StatusOK, comment)
}

// ListMyCommentedMemos 获取当前登录用户评论过的所有博文（去重，按最近评论时间倒序）
// 用于用户中心"我回复过的主题"，方便用户快速切回评论过的主题
func (h *CommentHandler) ListMyCommentedMemos(c *gin.Context) {
	userID := c.GetUint("user_id")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "请先登录"})
		return
	}

	// 查询该用户评论过的去重 memo_id 列表
	var memoIDs []uint
	if err := h.DB.Model(&models.Comment{}).
		Where("user_id = ?", userID).
		Distinct("memo_id").
		Pluck("memo_id", &memoIDs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}

	if len(memoIDs) == 0 {
		c.JSON(http.StatusOK, gin.H{"items": []models.Memo{}})
		return
	}

	// 查询这些博文（含作者与标签），按博文创建时间倒序（最新在前）
	var memos []models.Memo
	if err := h.DB.
		Preload("User").
		Preload("Tags").
		Where("id IN ?", memoIDs).
		Order("created_at DESC").
		Find(&memos).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询博文失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"items": memos})
}

// DeleteComment 删除评论（评论作者本人或 admin 可删）
func (h *CommentHandler) DeleteComment(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的评论 ID"})
		return
	}

	var comment models.Comment
	if err := h.DB.First(&comment, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "评论不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}

	userID := c.GetUint("user_id")
	role := c.GetString("role")

	// 仅评论作者本人或 admin 可删除
	isAuthor := comment.UserID != nil && *comment.UserID == userID
	isAdmin := role == models.RoleAdmin
	if !isAuthor && !isAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权删除该评论"})
		return
	}

	if err := h.DB.Delete(&comment).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除评论失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}
