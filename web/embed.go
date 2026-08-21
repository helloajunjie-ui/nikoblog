package web

import (
	"embed"
	"io/fs"
	"net/http"
	"path"
	"strings"
)

//go:embed dist
var distFS embed.FS

// DistFS 暴露嵌入的前端静态资源文件系统
func DistFS() fs.FS {
	sub, err := fs.Sub(distFS, "dist")
	if err != nil {
		panic("前端 dist 目录缺失，请先执行 npm run build: " + err.Error())
	}
	return sub
}

// Handler 返回一个 SPA 静态文件处理器。
// 对于 /api 和 /uploads 前缀直接放行（由其他路由处理），
// 其余路径尝试返回静态文件，找不到则回退到 index.html（支持前端路由）。
func Handler() http.Handler {
	sub := DistFS()
	fileServer := http.FileServer(http.FS(sub))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// API 与上传路径不在此处理
		if strings.HasPrefix(r.URL.Path, "/api") || strings.HasPrefix(r.URL.Path, "/uploads") {
			http.NotFound(w, r)
			return
		}

		// 尝试直接提供静态文件
		p := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		if p == "" {
			p = "index.html"
		}
		if _, err := fs.Stat(sub, p); err == nil {
			fileServer.ServeHTTP(w, r)
			return
		}

		// SPA fallback：返回 index.html
		r2 := r.Clone(r.Context())
		r2.URL.Path = "/"
		fileServer.ServeHTTP(w, r2)
	})
}
