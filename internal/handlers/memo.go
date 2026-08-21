package handlers

import (
	"bytes"
	"html"
	"net/http"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"nikoblog/internal/models"
	"nikoblog/internal/utils"
)

// MemoHandler 博文相关处理器
type MemoHandler struct {
	DB *gorm.DB
}

// NewMemoHandler 创建博文处理器
func NewMemoHandler(db *gorm.DB) *MemoHandler {
	return &MemoHandler{DB: db}
}

// CreateMemoRequest 发布博文请求体
type CreateMemoRequest struct {
	Content    string   `json:"content" binding:"required"`
	Images     []string `json:"images"`
	Visibility string   `json:"visibility"`
}

// Create 发布博文
func (h *MemoHandler) Create(c *gin.Context) {
	// 仅博主（admin）可发布博文
	if c.GetString("role") != models.RoleAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "仅博主可发布博文"})
		return
	}
	userID := c.GetUint("user_id")

	var req CreateMemoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}

	req.Content = strings.TrimSpace(req.Content)
	if req.Content == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "内容不能为空"})
		return
	}

	visibility := req.Visibility
	if visibility == "" {
		visibility = models.VisibilityPublic
	}
	if visibility != models.VisibilityPublic && visibility != models.VisibilityPrivate {
		c.JSON(http.StatusBadRequest, gin.H{"error": "可见性只能是 public 或 private"})
		return
	}

	memo, err := h.CreateMemoAsUser(userID, req.Content, visibility, req.Images)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "发布失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusCreated, memo)
}

// CreateMemoAsUser 以指定用户身份创建博文并维护标签（事务）。
// 供 HTTP handler 与后台自动任务引擎复用，保证发布逻辑单一来源。
func (h *MemoHandler) CreateMemoAsUser(userID uint, content, visibility string, images []string) (*models.Memo, error) {
	memo := models.Memo{
		UserID:     userID,
		Content:    content,
		Images:     models.StringList(images),
		Visibility: visibility,
	}

	// 事务：创建博文 + 维护标签
	err := h.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&memo).Error; err != nil {
			return err
		}
		return h.syncTags(tx, &memo)
	})
	if err != nil {
		return nil, err
	}

	// 重新加载关联标签
	h.DB.Preload("User").Preload("Tags").First(&memo, memo.ID)
	return &memo, nil
}

// List 博文列表（带可见性拦截）
// 未登录：只查 PUBLIC；已登录：PUBLIC + 自己的全部
// 过滤在 GORM 查询层完成，绝不在内存中过滤
func (h *MemoHandler) List(c *gin.Context) {
	// 从 Context 读取当前用户（未登录时为 0）
	uid := c.GetUint("user_id")

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	var memos []models.Memo
	query := h.DB.Model(&models.Memo{})

	// 可见性拦截：在 SQL/GORM 层过滤
	if uid == 0 {
		// 未登录：仅公开
		query = query.Where("visibility = ?", models.VisibilityPublic)
	} else {
		// 已登录：公开 或 自己的全部
		query = query.Where("visibility = ? OR user_id = ?", models.VisibilityPublic, uid)
	}

	var total int64
	query.Count(&total)

	err := query.Preload("User").Preload("Tags").
		Order("CASE WHEN pinned_at IS NOT NULL AND (pin_expire_at IS NULL OR pin_expire_at > CURRENT_TIMESTAMP) THEN 0 ELSE 1 END ASC").
		Order("id DESC").
		Offset((page - 1) * pageSize).Limit(pageSize).
		Find(&memos).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"total":     total,
		"page":      page,
		"page_size": pageSize,
		"items":     memos,
	})
}

// Get 获取单条博文
func (h *MemoHandler) Get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 ID"})
		return
	}

	uid := c.GetUint("user_id")

	var memo models.Memo
	query := h.DB.Preload("User").Preload("Tags").Where("id = ?", id)

	// 可见性拦截：未登录只能看公开
	if uid == 0 {
		query = query.Where("visibility = ?", models.VisibilityPublic)
	} else {
		query = query.Where("visibility = ? OR user_id = ?", models.VisibilityPublic, uid)
	}

	if err := query.First(&memo).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "博文不存在或无权访问"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}

	c.JSON(http.StatusOK, memo)
}

// UpdateRequest 更新博文请求体
type UpdateRequest struct {
	Content    *string  `json:"content"`
	Images     []string `json:"images"`
	Visibility *string  `json:"visibility"`
}

// Update 更新博文（仅作者本人）
func (h *MemoHandler) Update(c *gin.Context) {
	// 仅博主（admin）可修改博文
	if c.GetString("role") != models.RoleAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "仅博主可修改博文"})
		return
	}
	userID := c.GetUint("user_id")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 ID"})
		return
	}

	var memo models.Memo
	if err := h.DB.First(&memo, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "博文不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}

	// 权限校验：仅作者本人可修改
	if memo.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权修改他人的博文"})
		return
	}

	var req UpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}

	// 事务：更新博文 + 重新同步标签
	err = h.DB.Transaction(func(tx *gorm.DB) error {
		updates := map[string]interface{}{}
		if req.Content != nil {
			content := strings.TrimSpace(*req.Content)
			if content == "" {
				return gorm.ErrInvalidData
			}
			updates["content"] = content
		}
		if req.Images != nil {
			updates["images"] = models.StringList(req.Images)
		}
		if req.Visibility != nil {
			v := *req.Visibility
			if v != models.VisibilityPublic && v != models.VisibilityPrivate {
				return gorm.ErrInvalidData
			}
			updates["visibility"] = v
		}
		if len(updates) > 0 {
			if err := tx.Model(&memo).Updates(updates).Error; err != nil {
				return err
			}
		}
		// 若内容变化，重新同步标签
		if req.Content != nil {
			if err := h.syncTags(tx, &memo); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		if err == gorm.ErrInvalidData {
			c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新失败: " + err.Error()})
		return
	}

	h.DB.Preload("User").Preload("Tags").First(&memo, memo.ID)
	c.JSON(http.StatusOK, memo)
}

// Delete 删除博文（仅作者本人）
func (h *MemoHandler) Delete(c *gin.Context) {
	// 仅博主（admin）可删除博文
	if c.GetString("role") != models.RoleAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "仅博主可删除博文"})
		return
	}
	userID := c.GetUint("user_id")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 ID"})
		return
	}

	var memo models.Memo
	if err := h.DB.First(&memo, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "博文不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}

	if memo.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权删除他人的博文"})
		return
	}

	// 删除博文（GORM 会自动清理 many2many 关联）
	if err := h.DB.Delete(&memo).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}

// Search 搜索博文：内容 LIKE + 标签关联
func (h *MemoHandler) Search(c *gin.Context) {
	uid := c.GetUint("user_id")

	keyword := strings.TrimSpace(c.Query("q"))
	tag := strings.TrimSpace(c.Query("tag"))

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	if keyword == "" && tag == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "必须提供 q 或 tag 参数"})
		return
	}

	query := h.DB.Model(&models.Memo{})

	// 可见性拦截（SQL 层）
	if uid == 0 {
		query = query.Where("visibility = ?", models.VisibilityPublic)
	} else {
		query = query.Where("visibility = ? OR user_id = ?", models.VisibilityPublic, uid)
	}

	// 内容 LIKE 搜索
	if keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where("content LIKE ?", like)
	}

	// 标签关联搜索：通过中间表关联
	if tag != "" {
		query = query.Where("id IN (SELECT memo_id FROM memo_tags WHERE tag_id IN (SELECT id FROM tags WHERE name = ?))", tag)
	}

	var total int64
	query.Count(&total)

	var memos []models.Memo
	err := query.Preload("User").Preload("Tags").
		Order("CASE WHEN pinned_at IS NOT NULL AND (pin_expire_at IS NULL OR pin_expire_at > CURRENT_TIMESTAMP) THEN 0 ELSE 1 END ASC").
		Order("id DESC").
		Offset((page - 1) * pageSize).Limit(pageSize).
		Find(&memos).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"total":     total,
		"page":      page,
		"page_size": pageSize,
		"items":     memos,
	})
}

// ListTags 标签列表（带博文计数）
func (h *MemoHandler) ListTags(c *gin.Context) {
	var tags []models.Tag
	if err := h.DB.Order("memo_count DESC").Find(&tags).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": tags})
}

// HotTag 热门标签结果项
type HotTag struct {
	TagName string `json:"tagName"`
	Count   int64  `json:"count"`
}

// GetHotTags 获取热门标签：JOIN memo_tags 统计每个标签被 PUBLIC 博文引用的次数，按次数倒序
func (h *MemoHandler) GetHotTags(c *gin.Context) {
	var hotTags []HotTag
	err := h.DB.Model(&models.Tag{}).
		Select("tags.name AS tag_name, COUNT(memo_tags.memo_id) AS count").
		Joins("JOIN memo_tags ON memo_tags.tag_id = tags.id").
		Joins("JOIN memos ON memos.id = memo_tags.memo_id").
		Where("memos.visibility = ?", models.VisibilityPublic).
		Group("tags.id").
		Order("count DESC").
		Scan(&hotTags).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": hotTags})
}

// syncTags 解析博文内容中的 #标签，维护 Tag 表及中间表
func (h *MemoHandler) syncTags(tx *gorm.DB, memo *models.Memo) error {
	names := utils.ExtractTags(memo.Content)

	var tags []models.Tag
	for _, name := range names {
		var tag models.Tag
		err := tx.Where("name = ?", name).First(&tag).Error
		if err == gorm.ErrRecordNotFound {
			tag = models.Tag{Name: name}
			if err := tx.Create(&tag).Error; err != nil {
				return err
			}
		} else if err != nil {
			return err
		}
		tags = append(tags, tag)
	}

	// 替换关联（GORM 自动维护 memo_tags 中间表）
	if err := tx.Model(memo).Association("Tags").Replace(tags); err != nil {
		return err
	}

	// 重新统计每个标签的博文数量
	return h.recountTags(tx)
}

// recountTags 重新统计所有标签的博文数量
func (h *MemoHandler) recountTags(tx *gorm.DB) error {
	var tags []models.Tag
	if err := tx.Find(&tags).Error; err != nil {
		return err
	}
	for i := range tags {
		var count int64
		if err := tx.Model(&models.Memo{}).
			Joins("JOIN memo_tags ON memo_tags.memo_id = memos.id").
			Where("memo_tags.tag_id = ?", tags[i].ID).
			Count(&count).Error; err != nil {
			return err
		}
		if err := tx.Model(&tags[i]).Update("memo_count", count).Error; err != nil {
			return err
		}
	}
	return nil
}

// feedItem RSS 输出项
type feedItem struct {
	Title       string
	Link        string
	Description string
	PubDate     string
	GUID        string
}

// feedData RSS 频道数据
type feedData struct {
	Title       string
	Link        string
	Description string
	LastBuild   string
	Items       []feedItem
}

// rssTemplate 标准 RSS 2.0 XML 模板（text/template 渲染）
const rssTemplate = `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <title>{{.Title}}</title>
    <link>{{.Link}}</link>
    <description>{{.Description}}</description>
    <lastBuildDate>{{.LastBuild}}</lastBuildDate>
    {{range .Items}}
    <item>
      <title>{{.Title}}</title>
      <link>{{.Link}}</link>
      <guid>{{.GUID}}</guid>
      <pubDate>{{.PubDate}}</pubDate>
      <description>{{.Description}}</description>
    </item>
    {{end}}
  </channel>
</rss>
`

// stripMarkdown 简单去除常见 Markdown 标记，返回纯文本（用于 RSS 标题与描述）
func stripMarkdown(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	// 去掉行首的标题/列表/引用标记
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		trimmed = strings.TrimLeft(trimmed, "#>*-+` \t")
		lines[i] = trimmed
	}
	s = strings.Join(lines, " ")
	// 去掉行内 Markdown 标记
	s = strings.ReplaceAll(s, "**", "")
	s = strings.ReplaceAll(s, "__", "")
	s = strings.ReplaceAll(s, "`", "")
	s = strings.ReplaceAll(s, "~~", "")
	s = strings.ReplaceAll(s, "![", "")
	s = strings.ReplaceAll(s, "](", " ")
	s = strings.ReplaceAll(s, ")", "")
	s = strings.ReplaceAll(s, "[", "")
	s = strings.ReplaceAll(s, "]", "")
	s = strings.ReplaceAll(s, "#", "")
	// 压缩空白
	s = strings.Join(strings.Fields(s), " ")
	return strings.TrimSpace(s)
}

// Feed 输出 RSS 2.0：最新 20 条公开博文，link 指向 /m/:id
func (h *MemoHandler) Feed(c *gin.Context) {
	var memos []models.Memo
	if err := h.DB.
		Where("visibility = ?", models.VisibilityPublic).
		Order("id DESC").
		Limit(20).
		Find(&memos).Error; err != nil {
		c.String(http.StatusInternalServerError, "生成 RSS 失败")
		return
	}

	now := time.Now()
	items := make([]feedItem, 0, len(memos))
	for _, m := range memos {
		plain := stripMarkdown(m.Content)
		title := plain
		// 标题取正文前 20 字
		runes := []rune(title)
		if len(runes) > 20 {
			title = string(runes[:20]) + "…"
		}
		if title == "" {
			title = "无标题博文"
		}
		link := "/m/" + strconv.FormatUint(uint64(m.ID), 10)
		items = append(items, feedItem{
			Title:       html.EscapeString(title),
			Link:        link,
			GUID:        link,
			Description: html.EscapeString(plain),
			PubDate:     m.CreatedAt.Format(time.RFC1123Z),
		})
	}

	data := feedData{
		Title:       "nikoblog",
		Link:        "/",
		Description: "nikoblog 最新博文",
		LastBuild:   now.Format(time.RFC1123Z),
		Items:       items,
	}

	tmpl, err := template.New("rss").Parse(rssTemplate)
	if err != nil {
		c.String(http.StatusInternalServerError, "生成 RSS 失败")
		return
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		c.String(http.StatusInternalServerError, "生成 RSS 失败")
		return
	}

	c.Header("Content-Type", "application/xml; charset=utf-8")
	c.String(http.StatusOK, buf.String())
}
