# Android 集成约定

## 分页

列表接口使用 `limit` 与 `offset`。客户端应在每次刷新时从 `offset=0` 开始；当返回的 `items` 数量小于 `limit`，或 `offset + items.size >= total` 时停止。筛选或排序变化后必须清空旧列表并重新分页。

## SSE 重连

`GET /api/events` 使用当前登录 Cookie。连接断开后以 1、2、4、8、16、30 秒上限并带随机抖动重连；切换前后台时只保留一个连接。SSE 只用于提示状态变化，首次连接和每次重连成功后都应重新读取 `/api/dashboard` 与 `/api/tasks`。

## 错误处理

错误体固定为 `{ "code": string, "error": string }`。业务分支只依赖稳定的 `code` 与 HTTP 状态，不依赖可能翻译的 `error` 文本。`401` 进入登录页；`403` 更新 CSRF token 后最多重试一次；`409` 表示状态冲突；`429` 遵循 `Retry-After`；`503` 进入可恢复的维护或依赖离线状态。

所有写操作带 `X-CSRF-Token`。`202 Accepted` 仅表示异步任务已受理，最终状态通过任务列表和 SSE 确认。
