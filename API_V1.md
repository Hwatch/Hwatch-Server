# HWatch HTTP API V1

本文档是 HWatch API 的人工阅读指南。机器可读的唯一契约为根目录 `openapi.yaml`，运行实例提供 `/openapi.yaml` 和 `/docs`；当前路由仍使用 `/api/...`，没有 `/api/v1` 前缀。

## 1. 连接约定

- 默认地址：`http://<NAS-IP>:8787`
- API 数据：`application/json; charset=utf-8`
- 时间：服务端使用 RFC 3339 UTC，例如 `2026-09-11T16:39:37Z`。客户端应按 `dashboard.timezone` 转换后显示。
- 成功状态通常为 `200`、`201` 或 `202`。
- 错误格式：`{"code":"stable_error_code","error":"错误说明"}`；旧客户端继续读取 `error` 即可。
- 除首次初始化和登录接口外，API、媒体流和缩略图均要求管理员会话。
- ID 和订阅 slug 可能包含冒号，拼接路径时必须进行 URL 编码。

### 管理员认证

浏览器使用 `hwatch_session` HttpOnly、SameSite=Strict Cookie。所有非 GET/HEAD/OPTIONS 请求还必须携带登录响应返回的 `X-CSRF-Token` 请求头。

- `GET /api/auth/status`：返回 `{configured, authenticated, username?, csrf_token?}`。
- `POST /api/auth/setup`：仅尚无用户时可用，请求 `{username,password}`，创建首位管理员并登录。
- `POST /api/auth/login`：请求 `{username,password}`；单个来源地址 15 分钟内最多连续失败 5 次。
- `GET /api/auth/me`：返回当前管理员和 CSRF token。
- `POST /api/auth/logout`：注销当前会话。

密码长度为 10–256，服务端仅保存 bcrypt 哈希；会话默认有效期为 7 天。HTTPS 反向代理应传递 `X-Forwarded-Proto: https` 以启用 Secure Cookie。

示例：

```text
source:3144962 -> source%3A3144962
```

## 2. 通用数据结构

### Video

```json
{
  "video_id": "source:3144962",
  "subscription_slug": "source:creator",
  "title": "Original title",
  "description": "Structured source description",
  "translated_title": "中文标题",
  "translated_tags": ["标签一", "标签二"],
  "translation_language": "zh-CN",
  "page_url": "",
  "published_at": "2026-06-10T00:00:00Z",
  "discovered_at": "2026-09-11T16:39:37Z",
  "downloaded_at": "2026-09-11T18:00:00Z",
  "updated_at": "2026-09-11T18:00:00Z",
  "thumbnail_url": "",
  "views": 63132,
  "likes": 1866,
  "like_rate": 99,
  "duration_seconds": 171,
  "creator_name": "Creator",
  "creator_url": "",
  "creator_avatar": "",
  "creators": [{"Name":"Creator","URL":"","Image":""}],
  "source_relation": {"type":"creator_listing","platform":"source","subscription_slug":"source:creator","subscription_url":"","creator_key":"creator"},
  "uploader_name": "uploader",
  "uploader_url": "",
  "uploader_avatar": "",
  "categories": [{"Name":"Animation","URL":"","Image":""}],
  "tags": [{"Name":"sound","URL":""}],
  "external_links": [],
  "status": "completed",
  "aria_gid": "0123456789abcdef",
  "downloader_id": "aria2",
  "external_task_id": "0123456789abcdef",
  "file_path": "/media/...mp4",
  "plex_path": "/media/...mp4",
  "bytes_downloaded": 123456,
  "total_bytes": 123456,
  "download_speed": 0,
  "connections": 0,
  "video_codec": "h264",
  "audio_codec": "aac",
  "container": "mp4",
  "width": 1920,
  "height": 1080,
  "frame_rate": 30,
  "bit_rate": 4000000,
  "bit_depth": 8,
  "media_probed_at": "2026-09-11T18:00:10Z",
  "transcode_status": "not_requested",
  "plex_season": 1,
  "plex_episode": 398,
  "error": ""
}
```

`creator_name`、`creator_url` 和 `creator_avatar` 保留为主要创作者兼容字段；`creators` 包含详情页识别到的全部关联创作者。`source_relation` 记录视频由哪个订阅列表发现，用于解释归属和避免联合创作内容被主要创作者字段误排除。`description` 是清洗后的详情正文；`likes` 为点赞数量，`like_rate` 为百分比数值（`99` 表示 99%）。

视频状态：

- `discovered`：已发现、尚未选择下载
- `queued`：持久队列中，正在解析下载源
- `submitting`：已持久化外部任务标识，正在原子提交给下载器
- `waiting`：已提交到任务绑定的下载器，等待下载
- `active`：正在下载
- `paused`：已暂停
- `failed`：失败，可重试
- `excluded`：命中自动排除规则，未提交下载
- `completed`：下载完成，可以播放

客户端不要假设状态只会单向变化。429 自动恢复会把任务从 `failed` 或 aria2 错误状态重新变为 `queued`。

### Subscription

```json
{
  "id": 1,
  "slug": "source:creator",
  "platform": "source",
  "subscription_type": "creator",
  "source_id": "creator",
  "name": "Creator",
  "url": "",
  "avatar_url": "",
  "external_links": [],
  "enabled": true,
  "baseline_complete": true,
  "created_at": "2026-09-11T15:32:48Z",
  "last_checked_at": "2026-09-11T16:13:04Z",
  "last_error": "",
  "progress": {
    "total": 737,
    "completed": 583,
    "active": 2,
    "waiting": 152,
    "failed": 0,
    "bytes_downloaded": 1000,
    "total_bytes": 2000
  }
}
```

## 3. 概览

### `GET /api/dashboard`

查询参数：

- `language`：译文语言，默认 `zh-CN`

返回订阅、最近入库、最近播放、下载统计、流量、磁盘和时区。

重要字段：

```json
{
  "subscriptions": [],
  "videos": [],
  "recently_played": [],
  "counts": {"videos":874,"active":2,"waiting":100,"completed":600,"discovered":172},
  "traffic_bytes": 123,
  "library_bytes": 456,
  "current_download_speed": 789,
  "current_connections": 8,
  "downloads_paused": false,
  "disk_total": 965554995200,
  "disk_free": 273582915584,
  "aria2_online": true,
  "downloader_mode": "auto",
  "downloaders": [
    {"ID":"aria2","Name":"aria2","Version":"","Error":"","Online":true,"SupportsMulti":true,"Active":2,"Connections":8,"DownloadSpeed":789,"UptimeSeconds":3600},
    {"ID":"wget","Name":"wget","Version":"GNU Wget 1.21.3","Error":"","Online":true,"SupportsMulti":false,"Active":0,"Connections":0,"DownloadSpeed":0,"UptimeSeconds":3600}
  ],
  "timezone": "Asia/Shanghai"
}
```

- `aria2_online` 只表示服务端能否连接配置的 aria2 JSON-RPC，并不保证每个历史 GID 仍然存在。
- `downloads_paused` 是持久化的 HWatch 全局下载开关；为 `true` 时不再消费 `queued` 项，并尝试暂停 aria2 与 HWatch 管理的 wget 任务。
- `current_download_speed` 单位为字节/秒，`current_connections` 是数据库中所有 `active` 任务连接数之和。
- `downloader_mode` 为 `aria2`、`wget` 或 `auto`；`downloaders` 使用当前实现的 Go 字段名（首字母大写），包含每个引擎的在线状态和传输统计。

## 4. 影视库与搜索

### `GET /api/videos`

分页响应：

```json
{"items":[],"total":0,"limit":48,"offset":0}
```

查询参数：

| 参数 | 说明 |
|---|---|
| `limit` | 1–200，默认 48 |
| `offset` | 分页偏移 |
| `status` | 影视库传 `completed`；未下载页传 `discovered` |
| `language` | 当前译文语言 |
| `q` | 标题、简介、译文、作者、上传者、分类、标签或 ID |
| `scope` | `all`、`translated`、`original` |
| `creator` | 作者精确名称或订阅 slug |
| `tags` | 逗号分隔的包含标签 |
| `tag_mode` | `all`（AND）或 `any`（OR） |
| `exclude_tags` | 逗号分隔的排除标签 |
| `codec` | 视频编码，例如 `h264`、`hevc` |
| `min_height` | 最低高度，例如 1080 |
| `sort` | 见下方白名单 |
| `order` | `asc` 或 `desc` |

排序白名单：`downloaded_at`、`published_at`、`views`、`duration`、`size`、`resolution`、`bit_rate`、`title`、`recent_played`、`play_count`。

示例：在简中译文和原文中搜索，同时满足两个标签，并排除一个标签：

```http
GET /api/videos?status=completed&language=zh-CN&q=银狼&scope=all&tags=iwara,sound&tag_mode=all&exclude_tags=compilation&sort=views&order=desc
```

### `GET /api/videos/facets`

返回已下载影视库中可用的筛选项及视频数量：

```json
{
  "creators": [{"value":"Akt","count":595}],
  "tags": [{"value":"iwara","count":516}],
  "codecs": [{"value":"h264","count":500}]
}
```

### `GET /media/{video_id}`

播放已下载媒体。支持 Go `http.ServeFile` 提供的 Range 请求，可供 Android Media3/ExoPlayer 使用。成功访问已完成视频时会记录最近播放。

### `GET /thumb/{video_id}`

返回本地 Plex 海报或缓存缩略图。图片不可用时返回 SVG 占位图，因此客户端应按响应 `Content-Type` 解码，不要强制当作 JPEG。

## 5. 下载任务

`GET /api/tasks/{id}/history` 读取单个任务最近的持久状态变更历史。

### `GET /api/tasks`

参数：`limit`（最大 1000）、`offset`、`status`、`language`。不传状态时返回除 `discovered` 外的所有任务。

`status` 可使用 `queued`、`submitting`、`waiting`、`active`、`paused`、`failed`、`excluded` 或 `completed`。响应结构：

```json
{"items":[Video],"total":168,"limit":200,"offset":0}
```

### `POST /api/videos/download-all`

- 无请求体。
- 可选 `subscription` 查询参数，只排队该订阅的 `discovered` 视频。
- 返回 `202`：`{"accepted":388}`。

### `POST /api/downloads/toggle`

请求体为 `{"paused":true}` 时先持久化全局暂停状态、停止消费队列，并尝试暂停 aria2 与当前 HWatch 实例管理的 wget 任务；传入 `false` 时尝试恢复两个下载器、清除全局暂停状态并继续处理 `queued` 项。成功返回 `202`：

```json
{"paused":true}
```

全局暂停状态保存在 SQLite，并通过 `GET /api/dashboard` 的 `downloads_paused` 字段返回。当前实现不会因为某个下载器的暂停/恢复调用失败而让接口失败；客户端应继续通过 `/api/health`、`/api/dashboard` 和 `/api/tasks` 检查各下载器与任务的实际状态。

### `POST /api/videos/{video_id}/{action}`

`action` 支持：

- `download`：加入持久队列
- `retry`：失败任务重新加入队列
- `pause`：使用任务持久化的 `downloader_id` 暂停任务
- `resume`：使用原下载器继续任务；已脱离进程的 wget 任务会重新进入 `queued` 并断点续传
- `delete`：移除任务和媒体/NFO/海报，并恢复为 `discovered`

成功返回 `202`：`{"status":"accepted"}`。

`delete` 是破坏性操作，Android 客户端必须二次确认。

### `GET /api/events`

Server-Sent Events 实时下载状态，响应类型 `text/event-stream`，约每秒一条：

```text
data: {"tasks":[...],"download_speed":123456,"connections":8}
```

`tasks` 只包含 `waiting`、`active` 和 `paused` 项，不包含仍在解析下载源的 `queued` 项。Android 可使用 SSE 客户端；断线时建议 2–5 秒退避重连，并以 `/api/tasks` 做一次全量校准。SSE 中的 `download_speed` 单位为字节/秒。

## 6. 订阅

### `GET /api/subscriptions`

返回：`{"items":[Subscription...]}`。

`avatar_url` 保留上游原始地址；界面和客户端应优先使用同一对象中的
`avatar_proxy_url`，由 HWatch 代理并持久缓存头像，远端失效时返回占位图。

### `GET /avatar/{slug}`

返回订阅头像或 SVG 占位图。该接口需要登录会话，响应可由客户端缓存。

### `GET /api/platforms`

返回服务端已注册的适配器：`{"items":[{"id":"source","name":"Source"}]}`。

### `GET /api/creators/search`

参数：

- `q`：至少两个字符
- `platform`：适配器 ID；省略时使用服务端默认值

返回：

```json
{"items":[{"platform":"source","source_id":"3382","name":"Creator","url":"","avatar_url":"","video_count":877}]}
```

### `POST /api/subscriptions`

请求：

```json
{
  "url": "<subscription-url>",
  "type": "creator",
  "download_existing": false
}
```

- `download_existing=false`：只建立现有内容基线，以后发现的新视频自动下载。
- `download_existing=true`：订阅并将全部现有视频加入持久队列。
- `type` 可省略；显式提供时只允许与链接相符的 `creator` 或 `playlist`。
- 带时效签名的下载源会在任务开始前刷新，不在 API 或数据库中长期保存签名地址。

返回 `201`：

```json
{"slug":"source:creator","platform":"source","baselined":737,"queued":0}
```

### `PATCH /api/subscriptions/{slug}`

请求：`{"enabled":false}`。返回：`{"enabled":false}`。

### `DELETE /api/subscriptions/{slug}`

删除订阅及其数据库视频记录，并先停止该订阅进行中的下载。已完成媒体文件保留在磁盘，但不再出现在影视库。返回：`{"deleted":true}`。这是破坏性操作，客户端必须二次确认。

### `POST /api/check`

异步检查全部已启用订阅。返回 `202`：`{"status":"started"}`。

### `POST /api/check/{slug}`

异步检查指定订阅，响应同上。

## 7. 设置

新增运维接口：

- `POST /api/exclusions/preview` 使用未保存的规则草稿预览命中项及原因，不改变任务状态。
- `GET /api/audit?limit=&offset=` 分页读取结构化审计；服务不记录请求体、代理凭据、令牌或签名 URL。

### `GET /api/settings`

```json
{
  "http_proxy":"",
  "download_dir":"/media",
  "aria_rpc_url":"http://127.0.0.1:6800/jsonrpc",
  "aria_secret":"",
  "downloader_mode":"aria2",
  "check_time":"04:00",
  "timezone":"Asia/Shanghai",
  "plex_language":"zh-CN",
  "max_concurrent":2,
  "connections_per_download":4,
  "download_limit_kbps":0,
  "min_free_bytes":5368709120,
  "require_creator_match":true,
  "auto_transcode":false,
  "enable_media_duplicate":false,
  "media_duplicate_max_candidates":20,
  "media_duplicate_duration_percent":1,
  "exclude_title_keywords":"compilation,collection,合集,集锦",
  "exclude_uploaders":"",
  "exclude_categories":"",
  "exclude_tags":"",
  "max_duration_seconds":0,
  "backup_dir":"",
  "backup_time":"03:30",
  "backup_retention":7,
  "backup_last_success":"",
  "backup_last_error":""
}
```

### `PUT /api/settings`

发送需要修改的字段即可，服务端基于现有设置合并。成功返回完整设置。`download_limit_kbps=0` 表示不限速。

`downloader_mode` 可为 `aria2`、`wget` 或 `auto`。切换仅影响尚未启动的 `queued` 任务和新任务；已经分配的任务继续使用其持久化 `downloader_id`。wget 使用单连接及 `-c` 断点续传，服务重启后会重新解析临时下载地址并从现有文件继续。

自动排除规则在任务提交下载器前执行。多个文本规则使用逗号、中文逗号、分号或换行分隔；标题规则使用不区分大小写的包含匹配，上传者、分类和标签使用对应字段匹配。`require_creator_match=true` 会排除实际创作者与订阅作者不一致的视频，`max_duration_seconds=0` 表示不限制时长。已下载文件不会因新增规则而自动删除。

`auto_transcode=true` 时，新下载完成的视频会自动建立幂等的 H.265 转码任务，由已授权节点转码；成功后服务端用 `ffprobe` 校验并自动原子替换源文件，无需管理员确认，成功后删除暂存的原文件副本。

`enable_media_duplicate=true` 时，服务端会为已有内容哈希的完成媒体异步建立内容重复比较任务：先按 `abs(durationDelta) <= max(2s, duration * media_duplicate_duration_percent%)` 和排除 `SHA-256` 完全相同、媒体文件缺失、已确认/已忽略的项筛选候选，最多下发 `media_duplicate_max_candidates` 个。候选只在具备 `media_duplicate` 权限与能力的在线节点存在时排队。算法结果只生成重复候选，绝不自动删除媒体。

警告：当前接口会向已认证管理员返回 `aria_secret` 和可能含凭据的 `http_proxy`。客户端不得写入日志、崩溃报告或分析平台。

### SQLite 备份与恢复

- `GET /api/backups`：列出一致性快照、大小、SHA-256、完整性状态、实际目录和最近备份结果。
- `POST /api/backups`：立即创建快照，返回 `201`。备份使用 SQLite `VACUUM INTO`，不直接复制 WAL 文件。
- `POST /api/backups/{name}/restore`：请求体为 `{"confirm":true}`。恢复前创建回滚快照、暂停下载、校验完整性和必需 schema，原子替换数据库后返回 `202` 并重启服务。
- `GET /api/recovery/latest`：返回最近一次恢复协调摘要。恢复后保留当前部署的媒体与备份目录，将旧媒体根目录下的绝对路径映射到当前根目录；只在 aria2 明确返回 GID 不存在时重新排队。aria2 暂时离线或 RPC 异常时保留原 GID，并每 30 秒重试协调。

尚未进行过恢复时返回 `{"available":false}`。存在恢复记录时返回：

```json
{
  "available": true,
  "summary": {
    "completed_at": "2026-09-12T13:00:00Z",
    "source_media_root": "/old/media",
    "current_media_root": "/mnt/media",
    "subscriptions": 5,
    "completed_media": 737,
    "missing_media": 0,
    "path_mismatches": 0,
    "paths_remapped": 737,
    "aria_tasks_checked": 2,
    "aria_tasks_kept": 1,
    "aria_tasks_requeued": 1,
    "aria_unavailable": false,
    "reconciliation_done": true
  }
}
```

恢复完成后下载仍保持全局暂停，管理员应先检查摘要中的缺失媒体、根目录外路径和 aria2 重新排队数量，再手动继续下载。服务会对可访问但尚未探测的已下载媒体继续执行媒体信息探测。

`backup_dir` 留空时备份位于 `<download_dir>/.hwatch-backups`，当前部署即 NFS 媒体目录。数据库可能包含代理凭据、下载器密钥和管理员密码哈希，备份及校验和文件权限均为 `0600`。

## 8. 翻译与 Plex

### `GET /api/translations/export`

参数：`language`，可选 `platform`、`subscription`。返回附件 JSON，结构版本为 1。

### `POST /api/translations/import`

请求体是导出的 JSON 包。客户端填写每项的 `translations.title` 和 `translations.tags`，必须保留 `key`、`source_hash` 和 `fields`。

返回：`{"imported_fields":100,"skipped_items":2}`。原文哈希变化的条目会跳过。

### `POST /api/plex/refresh-nfo`

请求：`{"language":"zh-CN"}`。返回 `202`：`{"accepted":600,"language":"zh-CN"}`。

## 9. HTML 导入

### `POST /api/import/html`

请求：`{"html":"<div id=...>...</div>"}`。返回解析并保存后的 Video。该接口用于手动粘贴页面数据，Android 普通用户流程可以不实现。

## 10. 健康检查

### `GET /api/health`

返回数据库、媒体存储、持久队列和下载器状态。HTTP 请求成功并不等于所有组件健康；应检查顶层 `status`。

```json
{
  "status": "ok",
  "warnings": [],
  "database": {"online": true},
  "storage": {"online": true, "free_bytes": 273582915584, "minimum_free_bytes": 5368709120},
  "queue": {"paused": false},
  "downloaders": [
    {"ID":"aria2","Name":"aria2","Version":"","Error":"","Online":true,"SupportsMulti":true,"Active":2,"Connections":8,"DownloadSpeed":789,"UptimeSeconds":3600},
    {"ID":"wget","Name":"wget","Version":"GNU Wget 1.21.3","Error":"","Online":true,"SupportsMulti":false,"Active":0,"Connections":0,"DownloadSpeed":0,"UptimeSeconds":3600}
  ],
  "uptime_seconds": 3600
}
```

- `status=degraded`：数据库不可访问、媒体/NFS 目录不可用、空间低于最低保留值，或当前下载模式没有可用下载器。
- `warnings` 是结构化告警数组，包含稳定的 `code`、`severity` 和面向用户的 `message`。客户端逻辑应依赖 `code`，不要匹配中文文案。
- `storage.free_bytes` 和 `minimum_free_bytes` 单位均为字节。
- `downloaders[].DownloadSpeed` 单位为字节/秒。
- `downloaders[].UptimeSeconds` 当前表示 HWatch 本次进程运行时间，不是外部 aria2 进程启动时间。
- 固定下载器离线会令顶层状态降级；自动模式仅在 aria2 与 wget 都不可用时降级。

视频与任务对象额外返回 `error_code` 和 `error_retryable`。当前稳定分类包括 `capacity_reached`、`rate_limited`、`timeout`、`downloader_offline`、`detached_task`、`source_expired`、`storage_unavailable`、`permission_denied` 和 `download_failed`；无错误时 `error_code` 为空字符串。

### `GET /api/notifications`

返回持久系统通知及未读数量。当前通知覆盖媒体/NFS 不可用、空间低于保留值、当前下载器不可用和连续三个下载失败。相同故障采用 6 小时冷却，避免重复刷屏。

### `POST /api/notifications/read-all`

将当前全部通知标为已读，需要 CSRF 请求头。通知只保存稳定错误码和脱敏文案，不保存代理、下载地址或认证信息。

## 11. 重复内容审查

`POST /api/duplicates/batch` 可批量确认或忽略候选，响应包含本批候选涉及的预计空间；确认操作仍不会自动删除媒体。

- `GET /api/duplicates` 返回重复候选及两侧视频信息。
- `POST /api/duplicates/{id}/confirm` 确认重复并保持停止下载；不会删除任何已有媒体。
- `POST /api/duplicates/{id}/dismiss` 忽略该次匹配并将尚未下载的视频重新排队。

下载前先比较平台记录和去除查询参数、片段后的规范化详情 URL。随后计算封面 64 位感知哈希；只有相似度至少 92%，并且时长差不超过 5 秒时才停止下载并建立重复候选（视频沿用兼容的 `excluded` 状态）。下载完成后保存文件大小、首尾快速哈希和完整 SHA-256；完整哈希一致的已下载媒体只建立审查记录，不自动删除。

## 11. Android 实现建议

- 使用 Media3/ExoPlayer 播放 `/media/{id}`。
- 列表先加载 `/api/videos`，图片交给 Coil 加载 `/thumb/{id}`。
- 下载页先请求 `/api/tasks`，再连接 `/api/events` 增量更新。
- 所有服务端时间先解析为 `Instant`，显示时使用 `dashboard.timezone`。
- 原文/译文切换只影响显示；搜索范围通过 `scope` 明确传递。
- `video_id`、`slug` 和所有查询参数都必须 URL 编码。
- 不要依赖字段顺序；未知字段应忽略。
- 对 `202` 不要当作任务已完成，它只表示异步任务已接受。
## HWatch Node 外部节点

管理员使用现有 Cookie 会话和 CSRF 保护管理节点与任务：

- `GET/POST /api/nodes`：列出节点或创建节点。创建响应中的节点令牌只返回一次，数据库只保存 SHA-256 哈希。
- `DELETE /api/nodes/{id}`：撤销节点并回收其未完成租约。
- `GET/POST /api/compute/jobs`：列出任务或为已下载媒体建立 `dedupe`/`transcode` 任务。
- `POST /api/compute/jobs/{id}/cancel`：取消未完成任务。
- `POST /api/compute/jobs/{id}/commit`：兼容接口，用于提交仍处于 `verifying` 的旧转码任务。节点回传的转码产物现在会在验证通过后自动提交，默认删除暂存的原文件副本。

节点接口位于 `/api/node/v1/*`，使用创建节点时取得的 `Authorization: Bearer <token>`，不使用管理员 Cookie 或 CSRF：

- `POST /heartbeat`：上报版本、平台和能力。
- `POST /jobs/claim`：按节点能力领取任务；无任务时返回 `204`。
- `POST /jobs/{id}/progress`：上报进度并续期两分钟租约。
- `POST /jobs/{id}/complete`：回传去重结果或转码临时产物。
- `POST /jobs/{id}/fail`：失败或安全重新排队。

服务端只向节点返回 NFS 媒体相对路径。租约过期、节点撤销或节点失联后的任务可以重新领取；旧节点随后回传会收到 `409`。去重结果只能生成候选，节点无权删除媒体。转码完成回传后，服务端会立即用 `ffprobe` 校验并原子替换源路径；校验失败的任务进入 `failed`，成功则删除暂存原文件副本，管理员无需确认。

`dedupe` 任务现在使用结构化的 `media_duplicate` options，候选由服务端筛选后随任务下发，节点不得自行扫描媒体库：

```json
{
  "job_type": "dedupe",
  "subtype": "media_duplicate",
  "algorithm_version": "media-fingerprint-v1",
  "media_relative_path": "Platform/Creator/A.mp4",
  "input_json": "{\"mode\":\"media_duplicate\",\"candidates\":[{\"video_id\":\"...\",\"media_relative_path\":\"Platform/Creator/B.mkv\"}]}"
}
```

节点在 `POST /jobs/{id}/complete` 回传 `MediaDuplicateResult`：

```json
{
  "result": {
    "matches": [
      {
        "video_id": "候选 video_id",
        "classification": "whole-video",
        "backend": "media-fingerprint-v1",
        "algorithm_version": "media-fingerprint-v1",
        "confidence": 0.97,
        "duration_delta": 0,
        "metrics": {"sample_count": 32, "matched_samples": 32, "median_hamming_distance": 0, "duration_ratio": 0.999}
      }
    ]
  }
}
```

`classification` 仅接受 `whole-video`、`trimmed-video`、`possible-match`、`different`；前三者生成待审核的重复候选，`different` 不生成。服务端只接受任务下发的候选 ID，并按 `min/max(video_id)` 规范化配对，避免 A-B / B-A 重复。`confidence` 仅作展示，不用于自动删除。节点缺少 `media_duplicate` 权限或能力时不会被派发该子类型任务。
