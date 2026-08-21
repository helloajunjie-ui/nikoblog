package handlers

import (
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"nikoblog/internal/config"
	"nikoblog/internal/models"
	"nikoblog/internal/utils"
)

// 密保答错锁定阈值与时长
const (
	securityMaxFail = 2              // 答错 2 次
	securityLockDur = 24 * time.Hour // 锁定 24 小时
)

// 登录/改密失败锁定阈值与时长（防暴力破解）
const (
	loginMaxFail = 5                // 连续失败 5 次
	loginLockDur = 15 * time.Minute // 锁定 15 分钟
)

// AuthHandler 认证相关处理器
type AuthHandler struct {
	DB     *gorm.DB
	Config *config.Config
}

// NewAuthHandler 创建认证处理器
func NewAuthHandler(db *gorm.DB, cfg *config.Config) *AuthHandler {
	return &AuthHandler{DB: db, Config: cfg}
}

// RegisterRequest 注册请求体
type RegisterRequest struct {
	Username          string              `json:"username" binding:"required,min=3,max=64"`
	Password          string              `json:"password" binding:"required,min=6,max=72"`
	Nickname          string              `json:"nickname" binding:"max=64"`
	Email             string              `json:"email" binding:"required,email,max=128"`
	SecurityQuestions []models.SecurityQA `json:"security_questions" binding:"required,min=3,max=3"`
}

// Register 用户注册
func (h *AuthHandler) Register(c *gin.Context) {
	// 注册 IP 限流（每 IP 每小时最多 5 次）
	if !checkRegisterRateLimit(c) {
		return
	}

	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}

	req.Username = strings.TrimSpace(req.Username)
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	if req.Username == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "用户名不能为空"})
		return
	}

	// 校验密保问答：必须恰好 3 个，问题与答案均非空
	if len(req.SecurityQuestions) != 3 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请设置 3 个密保问题"})
		return
	}
	qaList := make(models.SecurityQAList, 0, 3)
	for _, qa := range req.SecurityQuestions {
		question := strings.TrimSpace(qa.Question)
		answer := strings.TrimSpace(qa.AnswerHash) // 前端传明文答案，此处字段复用
		if question == "" || answer == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "密保问题与答案不能为空"})
			return
		}
		answerHash, err := bcrypt.GenerateFromPassword([]byte(strings.ToLower(answer)), bcrypt.DefaultCost)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "密保答案加密失败"})
			return
		}
		qaList = append(qaList, models.SecurityQA{
			Question:   question,
			AnswerHash: string(answerHash),
		})
	}

	// 检查用户名是否已存在
	var count int64
	if err := h.DB.Model(&models.User{}).Where("username = ?", req.Username).Count(&count).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "数据库错误"})
		return
	}
	if count > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "用户名已存在"})
		return
	}

	// 检查邮箱是否已存在
	if err := h.DB.Model(&models.User{}).Where("email = ?", req.Email).Count(&count).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "数据库错误"})
		return
	}
	if count > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "该邮箱已被注册"})
		return
	}

	// bcrypt 加密密码
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "密码加密失败"})
		return
	}

	nickname := req.Nickname
	if nickname == "" {
		nickname = req.Username
	}

	// 首个注册用户自动成为管理员（博主）
	var userCount int64
	if err := h.DB.Model(&models.User{}).Count(&userCount).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "数据库错误"})
		return
	}
	role := models.RoleUser
	if userCount == 0 {
		role = models.RoleAdmin
	}

	user := models.User{
		Username:          req.Username,
		PasswordHash:      string(hash),
		Nickname:          nickname,
		Email:             req.Email,
		SecurityQuestions: qaList,
		Role:              role,
	}
	if err := h.DB.Create(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建用户失败"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "注册成功",
		"user":    user,
	})
}

// LoginRequest 登录请求体
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// Login 用户登录，返回 JWT Token
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}

	var user models.User
	if err := h.DB.Where("username = ?", strings.TrimSpace(req.Username)).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "用户名或密码错误"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "数据库错误"})
		return
	}

	// 检查登录锁定
	if user.LoginLockUntil != nil && time.Now().Before(*user.LoginLockUntil) {
		remain := time.Until(*user.LoginLockUntil).Round(time.Minute)
		c.JSON(http.StatusForbidden, gin.H{"error": "登录失败次数过多，账号已锁定，请 " + remain.String() + " 后再试"})
		return
	}

	// 校验密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		// 失败计数 +1，达到阈值则锁定
		newCount := user.LoginFailCount + 1
		if newCount >= loginMaxFail {
			lockUntil := time.Now().Add(loginLockDur)
			h.DB.Model(&user).Updates(map[string]interface{}{
				"login_fail_count": newCount,
				"login_lock_until": &lockUntil,
			})
			c.JSON(http.StatusForbidden, gin.H{"error": "登录失败次数过多，账号已锁定 15 分钟"})
			return
		}
		h.DB.Model(&user).Update("login_fail_count", newCount)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户名或密码错误，还可尝试 " + strconv.Itoa(loginMaxFail-newCount) + " 次"})
		return
	}

	// 登录成功：清零失败计数与锁定
	if user.LoginFailCount > 0 || user.LoginLockUntil != nil {
		h.DB.Model(&user).Updates(map[string]interface{}{
			"login_fail_count": 0,
			"login_lock_until": nil,
		})
	}

	// 生成 JWT
	token, err := utils.GenerateToken(user.ID, user.Username, user.Role, h.Config.JWTSecret)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "生成 Token 失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token": token,
		"user":  user,
	})
}

// UpdateSecurityRequest 更新密保问答请求体
type UpdateSecurityRequest struct {
	SecurityQuestions []models.SecurityQA `json:"security_questions" binding:"required,min=3,max=3"`
}

// UpdateSecurity 登录后更新密保问答
func (h *AuthHandler) UpdateSecurity(c *gin.Context) {
	uid := c.GetUint("user_id")
	var req UpdateSecurityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}

	if len(req.SecurityQuestions) != 3 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请设置 3 个密保问题"})
		return
	}

	qaList := make(models.SecurityQAList, 0, 3)
	for _, qa := range req.SecurityQuestions {
		question := strings.TrimSpace(qa.Question)
		answer := strings.TrimSpace(qa.AnswerHash)
		if question == "" || answer == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "密保问题与答案不能为空"})
			return
		}
		answerHash, err := bcrypt.GenerateFromPassword([]byte(strings.ToLower(answer)), bcrypt.DefaultCost)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "密保答案加密失败"})
			return
		}
		qaList = append(qaList, models.SecurityQA{
			Question:   question,
			AnswerHash: string(answerHash),
		})
	}

	// 更新密保并清零失败计数与锁定
	if err := h.DB.Model(&models.User{}).Where("id = ?", uid).Updates(map[string]interface{}{
		"security_questions":  qaList,
		"security_fail_count": 0,
		"security_lock_until": nil,
	}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新密保失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "密保更新成功"})
}

// SecurityQuestionRequest 获取密保问题请求体
type SecurityQuestionRequest struct {
	// 二选一：通过邮箱（找回用户名/密码）或 用户名+邮箱（找回密码）
	Email    string `json:"email" binding:"required,email"`
	Username string `json:"username"`
}

// SecurityQuestionResponse 返回随机一个密保问题
type SecurityQuestionResponse struct {
	Question string `json:"question"`
}

// GetSecurityQuestion 根据邮箱（可选用户名）返回随机一个密保问题
func (h *AuthHandler) GetSecurityQuestion(c *gin.Context) {
	var req SecurityQuestionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}

	email := strings.ToLower(strings.TrimSpace(req.Email))
	username := strings.TrimSpace(req.Username)

	var user models.User
	query := h.DB.Where("email = ?", email)
	if username != "" {
		query = query.Where("username = ?", username)
	}
	if err := query.First(&user).Error; err != nil {
		// 统一返回，避免暴露账号是否存在
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到匹配的账号"})
		return
	}

	// 检查是否处于锁定状态
	if user.SecurityLockUntil != nil && time.Now().Before(*user.SecurityLockUntil) {
		remain := time.Until(*user.SecurityLockUntil).Round(time.Minute)
		c.JSON(http.StatusForbidden, gin.H{"error": "密保验证失败次数过多，账号已锁定，请 " + remain.String() + " 后再试"})
		return
	}

	if len(user.SecurityQuestions) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "该账号未设置密保问题"})
		return
	}

	// 随机选一个密保问题
	qa := user.SecurityQuestions[rand.Intn(len(user.SecurityQuestions))]
	c.JSON(http.StatusOK, SecurityQuestionResponse{Question: qa.Question})
}

// ForgotUsernameRequest 忘记用户名请求体
type ForgotUsernameRequest struct {
	Email          string `json:"email" binding:"required,email"`
	SecurityAnswer string `json:"security_answer" binding:"required"`
}

// ForgotUsername 通过邮箱 + 密保答案找回用户名
func (h *AuthHandler) ForgotUsername(c *gin.Context) {
	var req ForgotUsernameRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}

	email := strings.ToLower(strings.TrimSpace(req.Email))
	var user models.User
	if err := h.DB.Where("email = ?", email).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "邮箱或密保答案不正确"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "数据库错误"})
		return
	}

	// 校验密保答案，失败则累计并可能锁定
	if !h.verifySecurityAnswer(c, &user, req.SecurityAnswer) {
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"username": user.Username,
	})
}

// ForgotPasswordRequest 忘记密码请求体
type ForgotPasswordRequest struct {
	Username       string `json:"username" binding:"required"`
	Email          string `json:"email" binding:"required,email"`
	SecurityAnswer string `json:"security_answer" binding:"required"`
	NewPassword    string `json:"new_password" binding:"required,min=6,max=72"`
}

// ForgotPassword 通过用户名 + 邮箱 + 密保答案重置密码
func (h *AuthHandler) ForgotPassword(c *gin.Context) {
	var req ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}

	username := strings.TrimSpace(req.Username)
	email := strings.ToLower(strings.TrimSpace(req.Email))

	var user models.User
	if err := h.DB.Where("username = ?", username).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "用户名、邮箱或密保答案不正确"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "数据库错误"})
		return
	}

	// 校验邮箱
	if !strings.EqualFold(user.Email, email) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户名、邮箱或密保答案不正确"})
		return
	}

	// 校验密保答案，失败则累计并可能锁定
	if !h.verifySecurityAnswer(c, &user, req.SecurityAnswer) {
		return
	}

	// 重置密码
	newHash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "密码加密失败"})
		return
	}
	if err := h.DB.Model(&user).Update("password_hash", string(newHash)).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "密码重置失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "密码重置成功",
	})
}

// ChangePasswordRequest 主动修改密码请求体
type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=6,max=72"`
}

// ChangePassword 已登录用户主动修改密码。
// 校验原密码，正确则直接更新为新密码；忘记密码请走 ForgotPassword 找回流程。
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}

	// 从 JWT 中间件写入的 context 获取当前用户 ID
	userID := c.GetUint("user_id")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return
	}

	var user models.User
	if err := h.DB.First(&user, userID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "用户不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "数据库错误"})
		return
	}

	// 检查改密锁定（与登录共用失败计数/锁定字段）
	if user.LoginLockUntil != nil && time.Now().Before(*user.LoginLockUntil) {
		remain := time.Until(*user.LoginLockUntil).Round(time.Minute)
		c.JSON(http.StatusForbidden, gin.H{"error": "操作失败次数过多，账号已锁定，请 " + remain.String() + " 后再试"})
		return
	}

	// 校验原密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.OldPassword)); err != nil {
		// 失败计数 +1，达到阈值则锁定
		newCount := user.LoginFailCount + 1
		if newCount >= loginMaxFail {
			lockUntil := time.Now().Add(loginLockDur)
			h.DB.Model(&user).Updates(map[string]interface{}{
				"login_fail_count": newCount,
				"login_lock_until": &lockUntil,
			})
			c.JSON(http.StatusForbidden, gin.H{"error": "操作失败次数过多，账号已锁定 15 分钟"})
			return
		}
		h.DB.Model(&user).Update("login_fail_count", newCount)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "原密码不正确，还可尝试 " + strconv.Itoa(loginMaxFail-newCount) + " 次"})
		return
	}

	// 生成新密码哈希并更新
	newHash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "密码加密失败"})
		return
	}
	if err := h.DB.Model(&user).Update("password_hash", string(newHash)).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "密码修改失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "密码修改成功",
	})
}

// verifySecurityAnswer 校验密保答案。
// 答对：清零失败计数并返回 true。
// 答错：失败计数 +1，达到阈值则锁定 24 小时，返回 false（已写入响应）。
func (h *AuthHandler) verifySecurityAnswer(c *gin.Context, user *models.User, answer string) bool {
	// 检查锁定
	if user.SecurityLockUntil != nil && time.Now().Before(*user.SecurityLockUntil) {
		remain := time.Until(*user.SecurityLockUntil).Round(time.Minute)
		c.JSON(http.StatusForbidden, gin.H{"error": "密保验证失败次数过多，账号已锁定，请 " + remain.String() + " 后再试"})
		return false
	}

	// 校验答案（任一密保问题匹配即通过）
	lowerAnswer := strings.ToLower(strings.TrimSpace(answer))
	matched := false
	for _, qa := range user.SecurityQuestions {
		if bcrypt.CompareHashAndPassword([]byte(qa.AnswerHash), []byte(lowerAnswer)) == nil {
			matched = true
			break
		}
	}

	if matched {
		// 答对：清零失败计数与锁定
		h.DB.Model(user).Updates(map[string]interface{}{
			"security_fail_count": 0,
			"security_lock_until": nil,
		})
		return true
	}

	// 答错：累计失败次数
	newCount := user.SecurityFailCount + 1
	if newCount >= securityMaxFail {
		lockUntil := time.Now().Add(securityLockDur)
		h.DB.Model(user).Updates(map[string]interface{}{
			"security_fail_count": newCount,
			"security_lock_until": &lockUntil,
		})
		c.JSON(http.StatusForbidden, gin.H{"error": "密保答案错误次数过多，账号已锁定 24 小时"})
		return false
	}

	h.DB.Model(user).Update("security_fail_count", newCount)
	c.JSON(http.StatusUnauthorized, gin.H{"error": "密保答案不正确，还可尝试 " + strconv.Itoa(securityMaxFail-newCount) + " 次"})
	return false
}
