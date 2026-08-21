package handlers

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// 注册 IP 限流阈值：每 IP 每小时最多 5 次注册
const (
	registerLimitMax    = 5
	registerLimitWindow = time.Hour
)

// ipRateLimiter 基于内存的 IP 滑动窗口限流器。
// 技术债：进程内 map，重启清零；单实例有效，多实例部署需改用 Redis 等共享存储。
type ipRateLimiter struct {
	mu    sync.Mutex
	items map[string][]time.Time // key: IP，value: 该窗口内的注册时间戳
}

// newIPRateLimiter 创建限流器
func newIPRateLimiter() *ipRateLimiter {
	return &ipRateLimiter{
		items: make(map[string][]time.Time),
	}
}

// allow 判断给定 IP 是否允许注册。
// 清理窗口外的旧记录，若窗口内次数已达上限则拒绝。
func (r *ipRateLimiter) allow(ip string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-registerLimitWindow)

	// 清理过期记录
	records := r.items[ip]
	kept := records[:0]
	for _, t := range records {
		if t.After(cutoff) {
			kept = append(kept, t)
		}
	}
	r.items[ip] = kept

	if len(kept) >= registerLimitMax {
		return false
	}

	// 记录本次注册
	r.items[ip] = append(r.items[ip], now)
	return true
}

// registerLimiter 全局注册限流器
var registerLimiter = newIPRateLimiter()

// clientIP 获取客户端真实 IP（优先 X-Forwarded-For / X-Real-IP，回退 RemoteAddr）
func clientIP(c *gin.Context) string {
	if ip := c.GetHeader("X-Forwarded-For"); ip != "" {
		return ip
	}
	if ip := c.GetHeader("X-Real-IP"); ip != "" {
		return ip
	}
	return c.ClientIP()
}

// checkRegisterRateLimit 注册前调用，超限则写入 429 响应并返回 false。
func checkRegisterRateLimit(c *gin.Context) bool {
	ip := clientIP(c)
	if !registerLimiter.allow(ip) {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "注册过于频繁，请 1 小时后再试"})
		return false
	}
	return true
}
