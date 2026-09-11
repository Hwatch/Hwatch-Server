# HWatch Agent Guide

## Scope

HWatch 是面向 NAS 的 Go + Vue 媒体订阅、下载和 Plex 电视库管理服务。当前平台通过统一适配器接入 Rule34Video，并为以后接入其他平台保留接口。

## Repository layout

- `cmd/r34watch/main.go`：HTTP API、调度器、持久下载队列、SSE、媒体与缩略图服务。
- `internal/adapters/`：统一平台适配器接口与具体实现。
- `internal/collector/`：Rule34Video 页面采集和解析。
- `internal/aria2/`：aria2 JSON-RPC 客户端。
- `internal/store/`：SQLite schema、查询及持久化。
- `internal/plex/`：Plex 目录、NFO 和海报生成。
- `internal/mediainfo/`：ffprobe 媒体信息探测。
- `src/`：Vue 3 + TypeScript 前端。
- `public/`：由 Vite 原样复制的静态资源。
- `API_V1.md`：当前客户端 API 合约。

## Required behavior

- SQLite 中的时间保持 UTC/RFC 3339；显示层按设置的 IANA 时区转换。
- 影视库只展示 `completed` 视频，未下载内容使用独立页面。
- 下载使用持久状态机：`discovered -> queued -> waiting -> active -> completed`。
- 服务重启后必须继续处理 `queued` 项。
- HTTP 429 需要自动回队和指数退避，不能形成高频重试。
- 默认下载并发为 2，单任务连接数为 4。
- Plex 媒体直接写入配置的媒体根目录，不复制第二份视频。
- 当前语言存在译文时优先显示译文，缺失时回退原文。
- 强演示模式不能泄露真实视频标题或标签。
- 缩略图优先使用本地 Plex 海报，其次使用持久缓存，远端失败返回占位图。

## Adapter contract

新增平台应实现 `internal/adapters.Adapter`：

- `ID`、`Name`、`MatchURL`
- `SearchCreators`
- `ParseSubscription`
- `ListVideos`
- `FetchDetails`
- `ResolveDownload`
- `ParseImported`

平台特有逻辑不要写进 HTTP handler、store 或 Vue 页面。数据库 ID 应使用 `<platform>:<source_id>`，订阅 slug 应使用 `<platform>:<creator_key>`。

## API compatibility

- 修改现有响应字段时保持向后兼容；优先新增可选字段。
- 不得改变 `video_id`、订阅 slug 或状态字符串的语义。
- 新增接口或参数后同步更新 `API_V1.md`。
- 排序字段只能通过服务端白名单映射，不能把客户端值直接拼入 SQL。
- 所有普通值必须使用参数化 SQL。
- `202 Accepted` 表示任务已进入异步流程，不代表完成。

## Security and privacy

- 不提交数据库、下载媒体、aria2 临时文件、二进制、Node 依赖或前端构建目录。
- 不提交访问令牌、Cookie、代理密码、aria2 secret 或真实部署凭据。
- 不在日志中输出代理 URL 的凭据、下载签名 URL 或认证令牌。
- 翻译导出不得包含本地路径、代理配置、Cookie 或下载地址。
- 删除订阅、任务和媒体属于破坏性操作，UI 必须二次确认。
- 当前 API 尚无登录认证；不要声称它适合直接暴露到公网。

## Editing rules

- 保留用户现有 SQLite 数据、NFS 媒体和进行中的 aria2 任务。
- 不通过数据库迁移重置现有订阅或下载状态。
- 修改 schema 时使用向前兼容的 `CREATE TABLE IF NOT EXISTS` 或显式迁移。
- 不将平台 HTML 原样长期保存在自动采集记录中，除非功能明确需要。
- Vue 页面当前较紧凑；新增复杂区域时优先拆成组件，避免继续扩大单行模板。

## Verification

提交前至少执行：

```bash
gofmt -w <modified-go-files>
go test ./...
go vet ./...
npm run build
go build -o r34watch ./cmd/r34watch
git diff --check
```

涉及 API 时使用本地服务做最小联调。涉及下载状态时只查询或操作测试目标，不批量删除、重置用户数据。涉及前端时确认桌面和窄屏侧栏均可用。

## Runtime notes

- 默认监听 `:8787`，部署时开放在 `0.0.0.0`。
- aria2 JSON-RPC 通常位于 `127.0.0.1:6800`。
- 实际媒体目录由 SQLite 设置或 `R34WATCH_MEDIA_ROOT` 决定。
- 生产前端位于 `web/dist`，构建后嵌入/由 Go 静态服务提供。
- 服务可能由 systemd 管理；重启前确认持久队列设计不会丢任务。

## Git hygiene

- `.gitignore` 中根目录 Go 二进制必须写成 `/r34watch`，不能写成 `r34watch`，否则会误忽略 `cmd/r34watch/`。
- 提交前检查 `git status --short --untracked-files=all`，确认服务入口和新组件已纳入版本控制。
- 不把带凭据的推送 URL保存为 `origin`。
