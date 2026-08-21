package main

import (
	"log"

	"github.com/gin-gonic/gin"

	"nikoblog/internal/config"
	"nikoblog/internal/cronjob"
	"nikoblog/internal/database"
	"nikoblog/internal/handlers"
	"nikoblog/internal/middleware"
	"nikoblog/web"
)

func main() {
	cfg := config.Load()

	// 初始化数据库
	db, err := database.Init(cfg)
	if err != nil {
		log.Fatalf("数据库初始化失败: %v", err)
	}

	// 初始化 Gin
	r := gin.Default()

	// 健康检查
	r.GET("/api/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// 认证处理器
	authHandler := handlers.NewAuthHandler(db, cfg)
	// 博文处理器
	memoHandler := handlers.NewMemoHandler(db)
	// 评论处理器
	commentHandler := handlers.NewCommentHandler(db)
	// 上传处理器
	uploadHandler := handlers.NewUploadHandler(cfg, db)
	// AI 能力处理器
	aiHandler := handlers.NewAIHandler(db)

	// 自动任务引擎（定时抓取 RSS → AI 洗稿 → 自动发布）
	cronjobManager := cronjob.NewManager(db, memoHandler)
	cronjobManager.Start()

	// 后台管理处理器（注入自动任务引擎，用于设置变更后热更新）
	adminHandler := handlers.NewAdminHandler(db, cronjobManager)

	// 静态文件服务：上传的图片
	r.Static("/uploads", cfg.UploadDir)

	// 前端静态资源（Go embed 打包的 Vite dist，SPA fallback）
	r.NoRoute(gin.WrapH(web.Handler()))

	// 公开路由
	api := r.Group("/api")
	{
		api.POST("/auth/register", authHandler.Register)
		api.POST("/auth/login", authHandler.Login)

		// 密保找回（公开）
		api.POST("/auth/security/question", authHandler.GetSecurityQuestion)
		api.POST("/auth/forgot/username", authHandler.ForgotUsername)
		api.POST("/auth/forgot/password", authHandler.ForgotPassword)

		// 公开的博文查询（未登录只查 PUBLIC，登录可查全量）
		// 使用可选鉴权：带有效 Token 则识别用户，否则视为未登录
		optionalAuth := middleware.OptionalAuthMiddleware(cfg.JWTSecret)
		api.GET("/memos", optionalAuth, memoHandler.List)
		api.GET("/memos/:id", optionalAuth, memoHandler.Get)
		api.GET("/memos/search", optionalAuth, memoHandler.Search)
		api.GET("/tags", memoHandler.ListTags)
		api.GET("/tags/hot", memoHandler.GetHotTags)

		// RSS 输出：最新公开博文（application/xml）
		api.GET("/feed", memoHandler.Feed)

		// 评论设置（公开，供前端判断游客是否可评论）
		api.GET("/settings/comments", commentHandler.GetCommentSettings)

		// 评论（公开读取 + 发表；发表支持登录用户与游客）
		api.GET("/memos/:id/comments", optionalAuth, commentHandler.ListComments)
		api.POST("/memos/:id/comments", optionalAuth, commentHandler.CreateComment)
	}

	// 受保护路由（需要 JWT）
	protected := api.Group("")
	protected.Use(middleware.AuthMiddleware(cfg.JWTSecret))
	{
		// 博文 CRUD
		protected.POST("/memos", memoHandler.Create)
		protected.PUT("/memos/:id", memoHandler.Update)
		protected.DELETE("/memos/:id", memoHandler.Delete)

		// 我评论过的博文（用户中心"回复过的主题"）
		protected.GET("/memos/commented", commentHandler.ListMyCommentedMemos)

		// 图片上传（需登录）
		protected.POST("/upload", uploadHandler.Upload)
		// 头像上传（需登录，最大 2MB，上传后自动更新用户头像）
		protected.POST("/upload/avatar", uploadHandler.UploadAvatar)

		// 删除评论（需登录，作者本人或 admin）
		protected.DELETE("/comments/:id", commentHandler.DeleteComment)

		// 更新密保问答（需登录）
		protected.PUT("/auth/security", authHandler.UpdateSecurity)
	}

	// 后台管理路由（需要 JWT + admin 角色）
	admin := api.Group("/admin")
	admin.Use(middleware.AuthMiddleware(cfg.JWTSecret), middleware.AdminMiddleware())
	{
		// 博客设置
		admin.GET("/settings", adminHandler.GetSettings)
		admin.PUT("/settings", adminHandler.UpdateSettings)

		// 用户管理
		admin.GET("/users", adminHandler.ListUsers)
		admin.PUT("/users/:id/role", adminHandler.UpdateUserRole)
		admin.DELETE("/users/:id", adminHandler.DeleteUser)

		// 文章管理
		admin.GET("/memos", adminHandler.ListAllMemos)
		admin.DELETE("/memos/:id", adminHandler.DeleteMemo)
		admin.PUT("/memos/:id/pin", adminHandler.PinMemo)

		// TAG 管理
		admin.GET("/tags", adminHandler.ListTags)
		admin.DELETE("/tags/:id", adminHandler.DeleteTag)

		// AI 能力
		admin.POST("/ai/polish", aiHandler.Polish)
	}

	log.Printf("nikoblog 服务启动，监听端口 %s", cfg.Port)
	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatalf("服务启动失败: %v", err)
	}
}
