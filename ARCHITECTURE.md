# nikoblog 架构文档

> 面向技术维护者。本文档描述当前磁盘上代码的真实物理状态，随代码变更同步更新。

## 1. 项目概览

nikoblog 是一个极简微博客系统（作者 niko，Q群 430291666），采用 **Go 后端 + Vue3 前端** 的单体架构。前端构建产物通过 Go `embed` 打包进单个二进制，最终以**单端口**对外提供服务（前端静态资源与 REST API 均由同一进程承载）。

- 后端：Go 1.26 + Gin v1.12 + GORM v1.31 + SQLite
- 前端：Vite v6 + Vue3 + Vue Router + Tailwind CSS v3.4
- 认证：JWT（HS256，有效期 24 小时）
- 数据库：SQLite（单文件，开启外键约束）

## 2. 目录结构

```
.
├── cmd/server/main.go        # 程序入口：路由注册、中间件装配、服务启动
├── internal/
│   ├── config/config.go      # 配置加载（环境变量 + 默认值）
│   ├── database/database.go  # SQLite 连接初始化 + GORM 自动迁移
│   ├── ai/client.go          # OpenAI 兼容 API 客户端（对话补全）
│   ├── cronjob/cronjob.go    # 自动任务引擎（RSS 抓取 → AI 洗稿 → 自动发布）
│   ├── handlers/             # HTTP 处理器（业务逻辑）
│   │   ├── auth.go           # 注册/登录/密保找回/更新密保
│   │   ├── memo.go           # 博文 CRUD、搜索、标签、置顶排序
│   │   ├── comment.go        # 评论 CRUD、评论设置、我回复过的主题
│   │   ├── admin.go          # 后台管理（设置/用户/博文/标签/置顶）
│   │   ├── ai.go             # AI 润色接口
│   │   └── upload.go         # 图片上传、头像上传
│   ├── middleware/           # Gin 中间件
│   │   ├── auth.go           # 强制 JWT 鉴权
│   │   ├── optional_auth.go  # 可选鉴权（公开接口识别登录用户）
│   │   └── admin.go          # admin 角色校验
│   ├── models/               # GORM 数据模型
│   │   ├── user.go           # 用户（含密保问答 JSON 字段）
│   │   ├── memo.go           # 博文（含图片 JSON、标签多对多、置顶字段）
│   │   ├── tag.go            # 标签 + 博文标签中间表
│   │   ├── comment.go        # 评论（游客/登录用户）
│   │   ├── setting.go        # 博客设置（含 AI 与自动任务配置）
│   │   └── cron_log.go       # 自动任务去重日志
│   └── utils/
│       ├── jwt.go            # JWT 生成/解析
│       └── tags.go           # #标签 正则提取
├── web/
│   ├── embed.go              # Go embed 前端 dist + SPA fallback
│   └── dist/                 # 前端构建产物（由 vite build 生成）
├── frontend/                 # Vue3 前端源码
│   ├── src/
│   │   ├── api/              # axios 封装 + 全部 API 调用
│   │   ├── components/       # 组件（编辑器/卡片/详情弹窗/侧栏等）
│   │   ├── views/            # 页面（首页/个人中心/后台）
│   │   └── router/           # 前端路由
│   └── vite.config.js        # 开发代理 + 构建输出到 web/dist
├── data/                     # 运行时数据（SQLite + 上传图片）
├── Dockerfile                # 多阶段构建（前端→后端→运行镜像）
└── docker-compose.yml        # 容器编排
```

## 3. 技术栈与依赖

### 3.1 后端依赖（go.mod）

| 依赖 | 版本 | 用途 |
|------|------|------|
| github.com/gin-gonic/gin | v1.12.0 | HTTP 框架 |
| github.com/golang-jwt/jwt/v5 | v5.3.1 | JWT 签发与校验 |
| golang.org/x/crypto | v0.55.0 | bcrypt 密码/密保答案哈希 |
| gorm.io/driver/sqlite | v1.6.0 | SQLite 驱动 |
| gorm.io/gorm | v1.31.2 | ORM |

### 3.2 前端依赖（frontend/package.json）

| 依赖 | 版本 | 用途 |
|------|------|------|
| vue | ^3.5.13 | 前端框架 |
| vue-router | ^4.6.4 | 前端路由 |
| axios | ^1.7.9 | HTTP 客户端 |
| marked | ^15.0.6 | Markdown 渲染 |
| dompurify | ^3.2.4 | HTML 消毒（防 XSS） |
| tailwindcss | ^3.4.17 | 样式（darkMode: 'class'） |
| vite | ^6.0.7 | 构建工具 |

## 4. 配置（internal/config/config.go）

所有配置通过环境变量注入，未设置时使用默认值：

| 环境变量 | 默认值 | 说明 |
|----------|--------|------|
| `NIKOBLOG_PORT` | `8080` | HTTP 监听端口 |
| `NIKOBLOG_DATA_DIR` | `./data` | 数据目录（SQLite + 上传图片） |
| `NIKOBLOG_JWT_SECRET` | `nikoblog-dev-secret-change-me` | JWT 签名密钥（生产必须修改） |

派生路径：
- 数据库文件：`{DataDir}/nikoblog.db`
- 上传目录：`{DataDir}/uploads`
- 单文件上传上限：固定 5MB（`MaxUploadSize`）

## 5. 数据模型

### 5.1 User（internal/models/user.go）

| 字段 | 类型 | 说明 |
|------|------|------|
| ID | uint | 主键 |
| Username | string | 用户名，唯一索引 |
| PasswordHash | string | bcrypt 哈希，JSON 不输出 |
| Nickname | string | 昵称 |
| Avatar | string | 头像 URL（`/uploads/...`） |
| Email | string | 邮箱，唯一索引 |
| SecurityQuestions | SecurityQAList | 密保问答（JSON 存储，1-3 个） |
| SecurityFailCount | int | 密保答错累计次数 |
| SecurityLockUntil | *time.Time | 密保锁定截止时间 |
| Role | string | `user` / `admin` |

角色常量：`RoleUser = "user"`、`RoleAdmin = "admin"`。

**首个注册用户自动成为 admin（博主）**，见 [`Register()`](internal/handlers/auth.go:46)。

### 5.2 Memo（internal/models/memo.go）

| 字段 | 类型 | 说明 |
|------|------|------|
| ID | uint | 主键 |
| UserID | uint | 作者 ID |
| User | User | 作者（外键关联） |
| Content | string | 正文（Markdown，含 `#标签`） |
| Images | StringList | 图片 URL 列表（JSON 存储） |
| Visibility | string | `public` / `private` |
| Tags | []Tag | 标签（many2many:memo_tags） |
| PinnedAt | *time.Time | 置顶时间，非空表示已置顶（用于排序） |
| PinExpireAt | *time.Time | 置顶截止时间；NULL 表示永久置顶，到期自动失效 |

可见性常量：`VisibilityPublic = "public"`、`VisibilityPrivate = "private"`。

**置顶排序**：`List` / `Search` / `ListAllMemos` 统一使用 SQLite 原生 `CURRENT_TIMESTAMP` 无参 CASE 表达式（见 §15），将有效置顶归为 0 组排前、普通博文归为 1 组排后，组内按 `id DESC`。**零定时任务**，过期自动失效靠查询时条件判断。

### 5.3 Tag 与 MemoTag（internal/models/tag.go）

- `Tag`：`ID`、`Name`（唯一索引）、`MemoCount`（博文计数）、时间戳
- `MemoTag`：`MemoID` + `TagID` 复合主键（GORM many2many 自动维护）

### 5.4 Comment（internal/models/comment.go）

| 字段 | 类型 | 说明 |
|------|------|------|
| ID | uint | 主键 |
| MemoID | uint | 所属博文 |
| UserID | *uint | 登录用户 ID；**NULL 表示游客评论** |
| User | *User | 关联用户（游客为 nil） |
| GuestName | string | 游客昵称 |
| Content | string | 评论内容 |
| Images | StringList | 评论附图（JSON 存储） |

### 5.5 Setting（internal/models/setting.go）

单行配置（ID=1），字段：

| 字段 | 类型 | 说明 |
|------|------|------|
| BlogName | string | 站点名称 |
| BlogDesc | string | 站点简介 |
| AllowRegister | bool | 是否开放注册 |
| AllowComment | bool | 是否开放评论 |
| AllowGuestComment | bool | 是否允许游客评论 |
| AiApiUrl | string | AI 服务 API 地址（OpenAI 兼容） |
| AiApiKey | string | AI 服务 API Key |
| AiModel | string | AI 模型名称 |
| DealSourceUrl | string | 自动任务数据源（RSS 链接） |
| DealCronExpr | string | 自动任务 cron 表达式（如 `0 10,16 * * *`） |
| AiSystemPrompt | string | AI 洗稿 System Prompt（后台可调教，为空时用代码内兜底 Prompt） |

### 5.6 CronLog（internal/models/cron_log.go）

自动任务去重日志：`SourceURL`（已处理条目的唯一标识，GUID/Link/Title）建立唯一索引，同一数据源条目只处理一次。

## 6. 认证与权限

### 6.1 JWT

- 签发：登录成功后 [`GenerateToken()`](internal/utils/jwt.go:19) 生成 HS256 Token，载荷含 `user_id`、`username`、`role`，有效期 24 小时。
- 校验：`AuthMiddleware`（强制）与 `OptionalAuthMiddleware`（可选）解析 Token 后将 `user_id`/`username`/`role` 写入 gin Context（**注意存入 `uint`，与 handler 中 `c.GetUint` 精确断言匹配**）。

### 6.2 中间件分层

| 中间件 | 行为 |
|--------|------|
| `AuthMiddleware` | 强制校验 Bearer Token，缺失/无效返回 401 |
| `OptionalAuthMiddleware` | 有有效 Token 则识别用户，否则视为未登录，**绝不拒绝请求** |
| `AdminMiddleware` | 校验 `role == admin`，否则 403（须在 AuthMiddleware 之后） |

### 6.3 权限规则

- **博文发布/修改/删除**：仅 admin（博主）可操作，且修改/删除仅限本人博文。
- **博文可见性**：未登录只查 `public`；已登录查 `public` + 自己的全部（在 SQL/GORM 层过滤）。
- **评论删除**：评论作者本人或 admin。
- **后台管理**：admin 角色专属。
- **头像上传**：需登录。
- **图片上传**：需登录。

## 7. 标签机制

标签为**半自动**：由博文正文中的 `#标签` 自动提取生成，无独立手动创建入口。

- 提取：发布/编辑博文时 [`syncTags()`](internal/handlers/memo.go:390) 调用 [`ExtractTags()`](internal/utils/tags.go:12)，用正则 `#([\p{L}\p{N}_]+)` 解析正文（支持中文/字母/数字/下划线），自动创建 Tag 并维护 `memo_tags` 中间表。
- 计数：`recountTags()` 重新统计每个标签的博文数量。
- 过滤：点击标签 → 前端带 `tag` 参数调用搜索接口 → 后端用子查询过滤出包含该标签的博文。
- 热门标签：`GetHotTags()` 通过 JOIN `memo_tags` + `memos` 统计每个标签被 **public** 博文引用的次数。

## 8. 上传机制

### 8.1 图片上传（POST /api/upload）

- 需登录；最大 5MB；仅允许 jpeg/png/gif/webp。
- 基于文件内容嗅探 MIME（`http.DetectContentType`），不信任扩展名。
- 文件名：`{时间戳}_{8位随机}.{ext}`，存入 `{UploadDir}`。
- 返回相对 URL `/uploads/{filename}`。

### 8.2 头像上传（POST /api/upload/avatar）

- 需登录；最大 2MB（`avatarMaxSize`）；仅允许 jpeg/png/gif/webp。
- 文件名：`avatar_{时间戳}_{8位随机}.{ext}`。
- 上传成功后**直接更新当前用户的 `avatar` 字段**。

## 9. 密保问答找回

- 注册/更新密保时，答案以 bcrypt 哈希存储（不存明文），问题与答案均非空，恰好 3 个。
- 找回流程：邮箱（+可选用户名）→ 返回随机一个密保问题 → 提交答案验证。
- 答错锁定：答错 2 次（`securityMaxFail`）锁定 24 小时（`securityLockDur`）；答对清零计数与锁定。
- 找回用户名：邮箱 + 密保答案。
- 找回密码：用户名 + 邮箱 + 密保答案 + 新密码。

## 10. REST API 一览

所有接口前缀 `/api`。认证方式：`Authorization: Bearer <token>`。

### 10.1 公开接口

| 方法 | 路径 | 说明 | 鉴权 |
|------|------|------|------|
| GET | /api/health | 健康检查 | 无 |
| POST | /api/auth/register | 注册 | 无 |
| POST | /api/auth/login | 登录 | 无 |
| POST | /api/auth/security/question | 获取密保问题 | 无 |
| POST | /api/auth/forgot/username | 找回用户名 | 无 |
| POST | /api/auth/forgot/password | 找回密码 | 无 |
| GET | /api/memos | 博文列表（分页） | 可选 |
| GET | /api/memos/:id | 博文详情 | 可选 |
| GET | /api/memos/search | 搜索（q 内容 / tag 标签） | 可选 |
| GET | /api/tags | 标签列表 | 无 |
| GET | /api/tags/hot | 热门标签 | 无 |
| GET | /api/settings/comments | 评论设置 | 无 |
| GET | /api/memos/:id/comments | 评论列表 | 可选 |
| POST | /api/memos/:id/comments | 发表评论 | 可选 |

### 10.2 受保护接口（需登录）

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | /api/memos | 发布博文（admin） |
| PUT | /api/memos/:id | 更新博文（作者本人） |
| DELETE | /api/memos/:id | 删除博文（作者本人） |
| GET | /api/memos/commented | 我评论过的博文（去重） |
| POST | /api/upload | 图片上传 |
| POST | /api/upload/avatar | 头像上传 |
| DELETE | /api/comments/:id | 删除评论（作者/admin） |
| PUT | /api/auth/security | 更新密保问答 |

### 10.3 后台管理接口（需登录 + admin）

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /api/admin/settings | 获取博客设置 |
| PUT | /api/admin/settings | 更新博客设置（含 AI 与自动任务配置） |
| GET | /api/admin/users | 用户列表（分页） |
| PUT | /api/admin/users/:id/role | 修改用户角色 |
| DELETE | /api/admin/users/:id | 删除用户（级联删博文） |
| GET | /api/admin/memos | 全量博文列表（分页，置顶优先） |
| DELETE | /api/admin/memos/:id | 删除任意博文 |
| PUT | /api/admin/memos/:id/pin | 置顶/取消置顶（body: `{pinned, expire_at?}`） |
| GET | /api/admin/tags | 标签列表 |
| DELETE | /api/admin/tags/:id | 删除标签 |
| POST | /api/admin/ai/polish | AI 润色博文（body: `{content}`） |

## 11. 前端架构

### 11.1 路由（frontend/src/router/index.js）

| 路径 | 视图 | 说明 |
|------|------|------|
| / | HomeView | 首页（博文流 + 标签侧栏 + 编辑器） |
| /profile | ProfileView | 个人中心（头像上传 + 我回复过的主题 + 密保设置） |
| /admin | AdminView | 后台管理（设置/用户/博文/标签） |

### 11.2 组件

- `MemoComposer`：编辑器（Markdown 工具栏含 `#` 标签按钮、图片上传、公开/私密切换、编辑/分屏/预览）
- `MemoCard`：博文卡片
- `MemoDetailModal`：博文详情弹窗（含评论）
- `TagSidebar`：用户信息 + 热门标签
- `AuthModal`：登录/注册弹窗
- `SecuritySettingsModal`：密保设置弹窗
- `RoleBadge`：角色徽章

### 11.3 状态与通信

- 登录态：`App.vue` 持有 `token`/`user`，存于 `localStorage`（`nikoblog_token`/`nikoblog_user`），通过 props 下发给各视图。
- 深色模式：`localStorage`（`nikoblog_dark`），`darkMode: 'class'`。
- 跨组件通信：`ProfileView` 上传头像后 `emit('avatar-updated', url)` → `App.vue` 的 `updateAvatar()` 同步 `user` 与 `localStorage`。
- 用户中心跳转：`ProfileView` 点击主题 → `router.push({ path: '/', query: { open: memoId } })` → `HomeView` 读取 `route.query.open` 自动打开详情弹窗。

### 11.4 HTTP 封装（frontend/src/api/http.js）

- axios 实例，`baseURL: ''`（开发走 vite 代理 `/api`）。
- 请求拦截器自动附加 `Authorization: Bearer <token>`。
- 响应拦截器统一返回 `res.data`，错误统一抛出 `error` 字段信息。

## 12. 构建与部署

### 12.1 本地开发

```bash
# 后端（8080）
go run ./cmd/server

# 前端（5173，代理 /api 与 /uploads 到 8080）
cd frontend && npm install && npm run dev
```

### 12.2 生产构建

```bash
# 构建前端 → web/dist
cd frontend && npm run build

# 编译后端（embed 打包 web/dist）
go build -o nikoblog ./cmd/server

# 运行
./nikoblog
```

### 12.3 Docker（Dockerfile）

多阶段构建：
1. `node:20-alpine`：安装前端依赖并 `npm run build`，产物输出到 `web/dist`。
2. `golang:1.26-alpine`：`CGO_ENABLED=0 go build`，embed 打包前端。
3. `alpine:3.20`：运行镜像，安装 ca-certificates 与 tzdata，`NIKOBLOG_DATA_DIR=/app/data`，暴露 8080，挂载卷 `/app/data`。

### 12.4 docker-compose.yml

- 单服务 `nikoblog`，映射 `8080:8080`。
- 环境变量：`NIKOBLOG_PORT`、`NIKOBLOG_DATA_DIR=/app/data`、`NIKOBLOG_JWT_SECRET`（默认 `please-change-me-in-production`）。
- 卷：`./data:/app/data` 持久化 SQLite 与上传图片。

## 13. 静态资源托管（web/embed.go）

- `//go:embed dist` 将前端构建产物嵌入二进制。
- `Handler()` 实现 SPA fallback：`/api` 与 `/uploads` 前缀放行（由其他路由处理）；其余路径尝试返回静态文件，找不到则回退到 `index.html`（支持前端路由）。

## 14. AI 能力（internal/ai + internal/handlers/ai.go）

- **客户端**：`ai.Client` 实现 OpenAI 兼容的 `POST {APIURL}/chat/completions` 对话补全，超时 60s。APIURL 为空时默认 `https://api.openai.com/v1`。
- **配置来源**：AI 配置（`AiApiUrl` / `AiApiKey` / `AiModel`）存于 `Setting` 表，后台可运行时修改，无需重启。
- **AI 润色**：`POST /api/admin/ai/polish`（仅 admin），将正文交给预设的润色 Prompt 处理，返回优化后的 Markdown。编辑器「AI 润色」按钮调用此接口。
- **未配置兜底**：`AiModel` 为空时接口返回 400「AI 服务未配置」，不会崩溃。

## 15. 自动任务引擎（internal/cronjob/cronjob.go）

- **职责**：定时抓取 RSS 数据源最新一条 → 交给 AI 洗稿 → 自动发布为公开博文。
- **调度**：基于 `robfig/cron/v3` 标准 5 段表达式（如 `0 10,16 * * *`），从 `Setting.DealCronExpr` 读取；未配置或表达式无效则不启动。
- **数据源**：`fetchLatest()` 兼容 RSS 2.0（`<channel><item>`）与 Atom（`<feed><entry>`），只取最新一条。
- **去重**：`CronLog.SourceURL` 唯一索引持久化去重（GUID/Link/Title），同一条目只处理一次。
- **洗稿 Prompt**：优先取后台配置的 `AiSystemPrompt`；为空时回退到代码内兜底 `dealPrompt`（毒舌羊毛导购专家），防止系统崩溃。
- **强制追加**：在 Go 代码层面强行追加原文链接与 `#自动抓取 #羊毛情报` 标签，不依赖 AI 拼接，保证链接不被吞并触发 `#标签` 提取。
- **发布**：以第一个 admin 用户身份调用 `CreateMemoAsUser` 发布为公开博文。
- **热更新**：`Reload()` 停止旧 cron、读取最新表达式重新注册，后台保存设置后自动调用，无需重启进程（通过 `Reloader` 接口注入，避免循环依赖）。

## 16. 置顶机制

- **字段**：`Memo.PinnedAt`（非空表示已置顶）+ `Memo.PinExpireAt`（NULL=永久置顶，到期自动失效）。
- **接口**：`PUT /api/admin/memos/:id/pin`，body `{pinned: bool, expire_at?: RFC3339}`。`pinned=false` 清空置顶；`pinned=true` 设置置顶（可带截止时间）。
- **排序**：`List` / `Search` / `ListAllMemos` 统一使用 SQLite 原生 `CURRENT_TIMESTAMP` 无参 CASE 表达式：
  ```sql
  ORDER BY CASE WHEN pinned_at IS NOT NULL AND (pin_expire_at IS NULL OR pin_expire_at > CURRENT_TIMESTAMP)
           THEN 0 ELSE 1 END ASC, id DESC
  ```
  有效置顶归为 0 组排前，普通博文归为 1 组排后，组内按 `id DESC`。
- **零定时任务**：过期自动失效完全靠查询时条件判断，不清理状态、不加定时任务。
- **前端**：`MemoCard` 展示 📌 置顶徽标；`AdminView` 提供置顶按钮 + `datetime-local` 截止时间输入。

## 17. 已知技术债与注意事项

- **SQLite 限制**：`Distinct + Order(聚合函数) + Pluck` 组合会产生无效 SQL，需拆分查询（见 `ListMyCommentedMemos`）。
- **GORM 丢参 Bug**：`Order(gorm.Expr("CASE ... ?", time.Now()))` 带 `?` 参数时，SQLite 方言会**丢弃整个 CASE 表达式**，导致排序退化。置顶排序因此改用 SQLite 原生 `CURRENT_TIMESTAMP` 常量（无参），彻底绕开该 Bug。
- **JWT 密钥**：默认值为开发密钥，生产环境必须通过 `NIKOBLOG_JWT_SECRET` 覆盖。
- **游客评论**：`Comment.UserID` 为 `*uint`，NULL 表示游客；游客不允许上传图片（后端强制 `Images = nil`），登录用户最多 5 张。
- **标签半自动**：标签只能由正文 `#标签` 生成，后台仅支持删除，不支持手动创建/重命名/合并。
- **自动任务依赖外部服务**：AI 洗稿依赖配置的 AI 服务可用；数据源抓取依赖目标 RSS 可达。任一失败仅记录日志并跳过本次，不影响主服务。
