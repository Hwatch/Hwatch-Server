# HWatch HTTP API V1

本文档描述当前 HWatch Go 服务供 Web、Android 和其他客户端使用的接口。`V1` 是本接口文档的版本；当前路由仍使用 `/api/...`，没有 `/api/v1` 前缀。本文档只描述已经实现的能力，计划中的 OpenAPI、wget 和多下载器切换尚不可用。

## 1. 连接约定

- 默认地址：`http://<NAS-IP>:8787`
- API 数据：`application/json; charset=utf-8`
- 时间：服务端使用 RFC 3339 UTC，例如 `2026-09-11T16:39:37Z`。客户端应按 `dashboard.timezone` 转换后显示。
- 成功状态通常为 `200`、`201` 或 `202`。
- 错误格式：`{"error":"错误说明"}`。
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

`creator_name`、`creator_url` 和 `creator_avatar` 保留为主要创作者兼容字段；`creators` 包含详情页识别到的全部关联创作者。`source_relation` 记录视频由哪个订阅列表发现，用于解释归属和避免联合创作内容被主要创作者字段误排除。

视频状态：

- `discovered`：已发现、尚未选择下载
- `queued`：持久队列中，正在解析下载源
- `waiting`：已提交 aria2，等待下载
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
  "timezone": "Asia/Shanghai"
}
```

- `aria2_online` 只表示服务端能否连接配置的 aria2 JSON-RPC，并不保证每个历史 GID 仍然存在。
- `downloads_paused` 是持久化的 HWatch 全局下载开关；为 `true` 时不再消费 `queued` 项，并要求 aria2 暂停已有任务。
- `current_download_speed` 单位为字节/秒，`current_connections` 是数据库中所有 `active` 任务连接数之和。

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
| `q` | 标题、译文、作者、上传者、分类、标签或 ID |
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

### `GET /api/tasks`

参数：`limit`（最大 1000）、`offset`、`status`、`language`。不传状态时返回除 `discovered` 外的所有任务。

`status` 可使用 `queued`、`waiting`、`active`、`paused`、`failed`、`excluded` 或 `completed`。响应结构：

```json
{"items":[Video],"total":168,"limit":200,"offset":0}
```

### `POST /api/videos/download-all`

- 无请求体。
- 可选 `subscription` 查询参数，只排队该订阅的 `discovered` 视频。
- 返回 `202`：`{"accepted":388}`。

### `POST /api/downloads/toggle`

请求体为 `{"paused":true}` 时暂停所有 aria2 下载并停止消费持久队列；传入 `false` 时恢复全部下载并继续处理 `queued` 项。成功返回 `202`：

```json
{"paused":true}
```

全局暂停状态会保存在 SQLite，并通过 `GET /api/dashboard` 的 `downloads_paused` 字段返回。aria2 RPC 不可用时返回 `502`，暂停请求不会被保存为成功状态。当前实现只控制 aria2；TODO 中的 wget 和多下载器热切换不属于当前 API。

### `POST /api/videos/{video_id}/{action}`

`action` 支持：

- `download`：加入持久队列
- `retry`：失败任务重新加入队列
- `pause`：暂停 aria2 任务
- `resume`：继续 aria2 任务
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

返回 `201`：

```json
{"slug":"source:creator","platform":"source","baselined":737,"queued":0}
```

### `PATCH /api/subscriptions/{slug}`

请求：`{"enabled":false}`。返回：`{"enabled":false}`。

### `DELETE /api/subscriptions/{slug}`

删除订阅及其数据库视频记录。返回：`{"deleted":true}`。这是破坏性操作，客户端必须二次确认。

### `POST /api/check`

异步检查全部已启用订阅。返回 `202`：`{"status":"started"}`。

### `POST /api/check/{slug}`

异步检查指定订阅，响应同上。

## 7. 设置

### `GET /api/settings`

```json
{
  "http_proxy":"",
  "download_dir":"/media",
  "aria_rpc_url":"http://127.0.0.1:6800/jsonrpc",
  "aria_secret":"",
  "check_time":"04:00",
  "timezone":"Asia/Shanghai",
  "plex_language":"zh-CN",
  "max_concurrent":2,
  "connections_per_download":4,
  "download_limit_kbps":0,
  "min_free_bytes":5368709120,
  "require_creator_match":true,
  "exclude_title_keywords":"compilation,collection,合集,集锦",
  "exclude_uploaders":"",
  "exclude_categories":"",
  "exclude_tags":"",
  "max_duration_seconds":0
}
```

### `PUT /api/settings`

发送需要修改的字段即可，服务端基于现有设置合并。成功返回完整设置。`download_limit_kbps=0` 表示不限速。

自动排除规则在任务提交下载器前执行。多个文本规则使用逗号、中文逗号、分号或换行分隔；标题规则使用不区分大小写的包含匹配，上传者、分类和标签使用对应字段匹配。`require_creator_match=true` 会排除实际创作者与订阅作者不一致的视频，`max_duration_seconds=0` 表示不限制时长。已下载文件不会因新增规则而自动删除。

警告：当前接口会返回 `aria_secret` 和可能含凭据的 `http_proxy`。Android 客户端不得写入日志、崩溃报告或分析平台；发布公网版本前应等待服务端认证和敏感字段脱敏协议。

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

## 10. Android 实现建议

- 使用 Media3/ExoPlayer 播放 `/media/{id}`。
- 列表先加载 `/api/videos`，图片交给 Coil 加载 `/thumb/{id}`。
- 下载页先请求 `/api/tasks`，再连接 `/api/events` 增量更新。
- 所有服务端时间先解析为 `Instant`，显示时使用 `dashboard.timezone`。
- 原文/译文切换只影响显示；搜索范围通过 `scope` 明确传递。
- `video_id`、`slug` 和所有查询参数都必须 URL 编码。
- 不要依赖字段顺序；未知字段应忽略。
- 对 `202` 不要当作任务已完成，它只表示异步任务已接受。
