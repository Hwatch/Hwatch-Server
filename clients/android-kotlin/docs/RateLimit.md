
# RateLimit

## Properties
| Name | Type | Description | Notes |
| ------------ | ------------- | ------------- | ------------- |
| **source** | **kotlin.String** | 平台/适配器标识，例如 rule34video |  |
| **backoffUntil** | [**java.time.OffsetDateTime**](java.time.OffsetDateTime.md) |  |  |
| **waiting** | **kotlin.Int** | 该来源当前等待中的 queued 任务数 |  |
| **host** | **kotlin.String** | 触发限速的真实主机 |  [optional] |
| **reason** | [**inline**](#Reason) | rate_limit&#x3D;HTTP 429；network&#x3D;连续连接级失败（TLS/EOF/reset）触发的来源恢复退避 |  [optional] |
| **retryAfterSeconds** | **kotlin.Int** | 服务端 Retry-After（若提供） |  [optional] |
| **consecutive429** | **kotlin.Int** |  |  [optional] |
| **transient** | **kotlin.Int** | 连续连接级失败次数 |  [optional] |


<a id="Reason"></a>
## Enum: reason
| Name | Value |
| ---- | ----- |
| reason | rate_limit, network |



