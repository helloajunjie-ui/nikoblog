// Package cronjob 提供后台自动任务引擎。
// 定时抓取数据源（RSS）最新内容，交给 AI 洗稿后自动发布为公开博文。
package cronjob

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/robfig/cron/v3"
	"gorm.io/gorm"

	"nikoblog/internal/ai"
	"nikoblog/internal/handlers"
	"nikoblog/internal/models"
)

// 预设的导购洗稿 Prompt（毒舌羊毛导购专家）
const dealPrompt = `你是一个资深、毒舌且极其务实的"羊毛导购专家"。你的唯一目的是帮读者省钱并避坑。
以下是一篇来自网络的促销/广告原文。不管原文写得多煽情、多长，你都必须无情地将其压缩并重构。绝不能全文照抄！不要任何废话！

必须严格且仅输出以下四个 Markdown 模块（包含标题）：

🏷️ 羊毛摘要
（用极简的一句话，说明这是卖什么的、有什么优惠。如果是常见的废话软文，直接提炼核心产品是什么。）

💰 核心价格与亮点
（只保留价格数字和最核心的硬件/服务参数。原文没有就写"未标明"。绝不复述对方的自吹自擂。）

⚠️ 青羽毒舌避坑
（以极度挑剔的眼光审视原文。寻找有没有跑路风险？是不是虚假宣传？价格真的便宜吗？直接给出你刻薄但理性的质疑。比如：对于API中转站，直接点出"中转站水深，注意跑路风险"。）

💡 终极购买建议
（非黑即白的结论：如"史低价格，闭眼入"、"骗小白的，别买"、"有白嫖额度可以试试，别充大钱"。）

【强制纪律】

最终输出字数绝不允许超过 600 字。
原文中的推广链接请原样保留，但放在文章最末尾，格式为：👉 直达围观 [blocked]
不要自己加客套话，不要写"好的，我已经为你重新排版"。直接输出结果！`

// feedItem 统一的数据源条目（兼容 RSS 2.0 与 Atom）
type feedItem struct {
	Title       string
	Link        string
	GUID        string
	Description string
}

// rss 结构：同时解析 RSS 2.0（<rss>/<channel>/<item>）与 Atom（<feed>/<entry>）。
// 两种格式共用同一结构体，通过根元素区分。
type rss struct {
	// RSS 2.0
	Channel struct {
		Items []struct {
			Title       string `xml:"title"`
			Link        string `xml:"link"`
			GUID        string `xml:"guid"`
			Description string `xml:"description"`
		} `xml:"item"`
	} `xml:"channel"`
	// Atom
	Entries []struct {
		Title string `xml:"title"`
		Link  struct {
			Href string `xml:"href,attr"`
		} `xml:"link"`
		ID          string `xml:"id"`
		Summary     string `xml:"summary"`
		Content     string `xml:"content"`
		Description string `xml:"description"`
	} `xml:"entry"`
}

// Reloader 热更新接口：供 handlers 包调用，避免 handlers→cronjob→handlers 循环依赖。
type Reloader interface {
	Reload()
}

// Manager 自动任务引擎
type Manager struct {
	DB          *gorm.DB
	MemoHandler *handlers.MemoHandler
	cron        *cron.Cron
}

// NewManager 创建自动任务引擎
func NewManager(db *gorm.DB, memoHandler *handlers.MemoHandler) *Manager {
	return &Manager{
		DB:          db,
		MemoHandler: memoHandler,
		// 使用标准 5 段 cron（与 cron.ParseStandard 一致），如 "0 10,16 * * *"
		cron: cron.New(),
	}
}

// Start 启动定时任务。根据 Setting 中的 DealCronExpr 注册任务。
func (m *Manager) Start() {
	var setting models.Setting
	if err := m.DB.First(&setting).Error; err != nil {
		log.Printf("[cronjob] 读取设置失败，自动任务未启动: %v", err)
		return
	}
	expr := strings.TrimSpace(setting.DealCronExpr)
	if expr == "" {
		log.Printf("[cronjob] 未配置执行计划（DealCronExpr），自动任务未启动")
		return
	}
	// 校验 cron 表达式
	if _, err := cron.ParseStandard(expr); err != nil {
		log.Printf("[cronjob] cron 表达式无效 %q，自动任务未启动: %v", expr, err)
		return
	}
	_, err := m.cron.AddFunc(expr, m.run)
	if err != nil {
		log.Printf("[cronjob] 注册定时任务失败: %v", err)
		return
	}
	m.cron.Start()
	log.Printf("[cronjob] 自动任务已启动，执行计划: %s", expr)
}

// Stop 停止定时任务
func (m *Manager) Stop() {
	if m.cron != nil {
		m.cron.Stop()
	}
}

// run 单次任务执行：抓取 → 洗稿 → 发布
func (m *Manager) run() {
	log.Printf("[cronjob] 开始执行自动任务")
	var setting models.Setting
	if err := m.DB.First(&setting).Error; err != nil {
		log.Printf("[cronjob] 读取设置失败: %v", err)
		return
	}
	sourceURL := strings.TrimSpace(setting.DealSourceUrl)
	if sourceURL == "" {
		log.Printf("[cronjob] 未配置数据源（DealSourceUrl），跳过")
		return
	}
	if strings.TrimSpace(setting.AiModel) == "" {
		log.Printf("[cronjob] 未配置 AI 模型，跳过")
		return
	}

	// 1. 抓取 RSS 最新一条
	item, err := m.fetchLatest(sourceURL)
	if err != nil {
		log.Printf("[cronjob] 抓取数据源失败: %v", err)
		return
	}
	key := item.GUID
	if key == "" {
		key = item.Link
	}
	if key == "" {
		key = item.Title
	}
	if key == "" {
		log.Printf("[cronjob] 数据源条目缺少唯一标识，跳过")
		return
	}
	// 2. SQLite 持久化去重：先向 CronLog 插入该条目标识。
	// 若唯一索引冲突（SourceURL 已存在），说明已处理过，直接跳过。
	if err := m.DB.Create(&models.CronLog{SourceURL: key}).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) || strings.Contains(err.Error(), "UNIQUE constraint failed") {
			log.Printf("[cronjob] 条目已处理过，跳过: %s", key)
			return
		}
		log.Printf("[cronjob] 写入去重日志失败: %v", err)
		return
	}
	log.Printf("[cronjob] 新条目，开始洗稿: %s", key)

	// 3. 组装洗稿输入
	raw := item.Title
	if item.Description != "" {
		raw = item.Title + "\n\n" + item.Description
	}

	// 3. AI 洗稿
	// System Prompt 优先取后台配置的 AiSystemPrompt；若为空则回退到代码内兜底 Prompt（dealPrompt），防止系统崩溃。
	systemPrompt := strings.TrimSpace(setting.AiSystemPrompt)
	if systemPrompt == "" {
		systemPrompt = dealPrompt
	}
	client := ai.NewClient(setting.AiApiUrl, setting.AiApiKey, setting.AiModel)
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	result, err := client.ChatCompletion(ctx, []ai.ChatMessage{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: raw},
	})
	if err != nil {
		log.Printf("[cronjob] AI 洗稿失败: %v", err)
		return
	}
	result = strings.TrimSpace(result)
	if result == "" {
		log.Printf("[cronjob] AI 洗稿结果为空，跳过")
		return
	}

	// 4. 在 Go 代码层面强行追加原文链接与标签，不依赖 AI 拼接。
	//    保证链接不被大模型吃掉/写成 [blocked]，并触发 CreateMemoAsUser 的 #标签 正则提取。
	if item.Link != "" {
		result += "\n\n[👉 直达围观原帖](" + item.Link + ")"
	}
	result += "\n#自动抓取 #羊毛情报"

	// 5. 获取 admin 用户 ID 并发布为公开博文
	adminID, err := m.getAdminID()
	if err != nil {
		log.Printf("[cronjob] 获取管理员失败: %v", err)
		return
	}
	if _, err := m.MemoHandler.CreateMemoAsUser(adminID, result, models.VisibilityPublic, nil); err != nil {
		log.Printf("[cronjob] 发布博文失败: %v", err)
		return
	}

	log.Printf("[cronjob] 已自动发布新博文: %s", item.Title)
}

// Reload 热更新定时任务：停止旧的 cron，读取最新表达式重新注册并启动。
// 供后台设置保存后调用，无需重启进程。
func (m *Manager) Reload() {
	m.Stop()

	m.cron = cron.New()

	var setting models.Setting
	if err := m.DB.First(&setting).Error; err != nil {
		log.Printf("[cronjob] Reload 读取设置失败: %v", err)
		return
	}
	expr := strings.TrimSpace(setting.DealCronExpr)
	if expr == "" {
		log.Printf("[cronjob] Reload 后未配置执行计划（DealCronExpr），自动任务已停止")
		return
	}
	if _, err := cron.ParseStandard(expr); err != nil {
		log.Printf("[cronjob] Reload 后 cron 表达式无效 %q，自动任务已停止: %v", expr, err)
		return
	}
	if _, err := m.cron.AddFunc(expr, m.run); err != nil {
		log.Printf("[cronjob] Reload 注册定时任务失败: %v", err)
		return
	}
	m.cron.Start()
	log.Printf("[cronjob] 自动任务已热更新，执行计划: %s", expr)
}

// fetchLatest 抓取数据源并返回最新一条条目（兼容 RSS 2.0 与 Atom）
func (m *Manager) fetchLatest(url string) (*feedItem, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("请求数据源失败: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("数据源返回状态码 %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 5*1024*1024))
	if err != nil {
		return nil, fmt.Errorf("读取数据源失败: %w", err)
	}
	var feed rss
	if err := xml.Unmarshal(body, &feed); err != nil {
		return nil, fmt.Errorf("解析数据源失败: %w", err)
	}

	// 优先 RSS 2.0（<channel><item>）
	if len(feed.Channel.Items) > 0 {
		it := feed.Channel.Items[0]
		return &feedItem{Title: it.Title, Link: it.Link, GUID: it.GUID, Description: it.Description}, nil
	}
	// 其次 Atom（<feed><entry>）
	if len(feed.Entries) > 0 {
		e := feed.Entries[0]
		desc := e.Summary
		if desc == "" {
			desc = e.Content
		}
		if desc == "" {
			desc = e.Description
		}
		return &feedItem{Title: e.Title, Link: e.Link.Href, GUID: e.ID, Description: desc}, nil
	}
	return nil, fmt.Errorf("数据源无条目")
}

// getAdminID 查询第一个 admin 用户 ID
func (m *Manager) getAdminID() (uint, error) {
	var user models.User
	if err := m.DB.Where("role = ?", models.RoleAdmin).Order("id ASC").First(&user).Error; err != nil {
		return 0, err
	}
	return user.ID, nil
}
