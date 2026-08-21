package handlers

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"nikoblog/internal/config"
	"nikoblog/internal/models"
)

// 允许的图片 MIME 类型
var allowedImageTypes = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/gif":  ".gif",
	"image/webp": ".webp",
}

// UploadHandler 图片上传处理器
type UploadHandler struct {
	Config *config.Config
	DB     *gorm.DB
}

// NewUploadHandler 创建上传处理器
func NewUploadHandler(cfg *config.Config, db *gorm.DB) *UploadHandler {
	return &UploadHandler{Config: cfg, DB: db}
}

// Upload 上传图片
// 限制：最大 5MB，仅允许 jpeg/png/gif/webp，校验 MIME Type 防木马
func (h *UploadHandler) Upload(c *gin.Context) {
	// 限制请求体大小（5MB + 少量余量）
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, h.Config.MaxUploadSize+1024)

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		if strings.Contains(err.Error(), "request body too large") {
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "文件过大，最大允许 5MB"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少文件字段 file"})
		return
	}
	defer file.Close()

	// 检查文件大小
	if header.Size > h.Config.MaxUploadSize {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "文件过大，最大允许 5MB"})
		return
	}

	// 校验 MIME Type（基于文件内容嗅探，而非仅信任扩展名）
	buf := make([]byte, 512)
	n, err := file.Read(buf)
	if err != nil && err != io.EOF {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "读取文件失败"})
		return
	}
	mimeType := http.DetectContentType(buf[:n])
	ext, ok := allowedImageTypes[mimeType]
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "不支持的文件类型，仅允许 jpeg/png/gif/webp 图片"})
		return
	}

	// 生成唯一文件名（时间戳 + 随机后缀）
	filename := fmt.Sprintf("%d_%s%s", time.Now().UnixNano(), randomString(8), ext)
	destPath := filepath.Join(h.Config.UploadDir, filename)

	// 写入文件（从文件开头重新读取）
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "文件处理失败"})
		return
	}
	out, err := os.Create(destPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存文件失败"})
		return
	}
	defer out.Close()

	if _, err := io.Copy(out, file); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "写入文件失败"})
		return
	}

	// 返回相对 URL（前端通过 /uploads/xxx 访问）
	relativeURL := "/uploads/" + filename
	c.JSON(http.StatusOK, gin.H{
		"url":  relativeURL,
		"name": filename,
	})
}

// avatarMaxSize 头像上传大小上限（2MB）
const avatarMaxSize = 2 * 1024 * 1024

// UploadAvatar 上传用户头像（需登录）
// 限制：最大 2MB，仅允许 jpeg/png/gif/webp 图片；上传成功后直接更新当前用户的 avatar 字段
func (h *UploadHandler) UploadAvatar(c *gin.Context) {
	userID := c.GetUint("user_id")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "请先登录"})
		return
	}

	// 限制请求体大小（2MB + 少量余量）
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, avatarMaxSize+1024)

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		if strings.Contains(err.Error(), "request body too large") {
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "头像文件过大，最大允许 2MB"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少文件字段 file"})
		return
	}
	defer file.Close()

	// 检查文件大小
	if header.Size > avatarMaxSize {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "头像文件过大，最大允许 2MB"})
		return
	}

	// 校验 MIME Type（基于文件内容嗅探）
	buf := make([]byte, 512)
	n, err := file.Read(buf)
	if err != nil && err != io.EOF {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "读取文件失败"})
		return
	}
	mimeType := http.DetectContentType(buf[:n])
	ext, ok := allowedImageTypes[mimeType]
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "不支持的文件类型，仅允许 jpeg/png/gif/webp 图片"})
		return
	}

	// 生成唯一文件名
	filename := fmt.Sprintf("avatar_%d_%s%s", time.Now().UnixNano(), randomString(8), ext)
	destPath := filepath.Join(h.Config.UploadDir, filename)

	// 写入文件（从文件开头重新读取）
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "文件处理失败"})
		return
	}
	out, err := os.Create(destPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存文件失败"})
		return
	}
	defer out.Close()

	if _, err := io.Copy(out, file); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "写入文件失败"})
		return
	}

	// 更新当前用户的头像字段
	relativeURL := "/uploads/" + filename
	if err := h.DB.Model(&models.User{}).Where("id = ?", userID).Update("avatar", relativeURL).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新头像失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"url":  relativeURL,
		"name": filename,
	})
}

// randomString 生成指定长度的随机字母数字字符串
func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	seed := uint64(time.Now().UnixNano())
	for i := range b {
		seed = seed*6364136223846793005 + 1442695040888963407
		b[i] = letters[(seed>>33)%uint64(len(letters))]
	}
	return string(b)
}
