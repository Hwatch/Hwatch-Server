
# Video

## Properties
| Name | Type | Description | Notes |
| ------------ | ------------- | ------------- | ------------- |
| **videoId** | **kotlin.String** |  |  |
| **subscriptionSlug** | **kotlin.String** |  |  |
| **title** | **kotlin.String** |  |  |
| **status** | [**inline**](#Status) |  |  |
| **description** | **kotlin.String** |  |  [optional] |
| **translatedTitle** | **kotlin.String** |  |  [optional] |
| **views** | **kotlin.Long** |  |  [optional] |
| **likes** | **kotlin.Long** |  |  [optional] |
| **likeRate** | **kotlin.Double** | 点赞百分比，例如 99 表示 99% |  [optional] |
| **tags** | **kotlin.collections.List&lt;kotlin.collections.Map&lt;kotlin.String, kotlin.Any&gt;&gt;** |  |  [optional] |
| **downloaderId** | **kotlin.String** | 未分配时为空字符串，已分配时为 aria2 或 wget |  [optional] |
| **externalTaskId** | **kotlin.String** |  |  [optional] |
| **bytesDownloaded** | **kotlin.Long** |  |  [optional] |
| **totalBytes** | **kotlin.Long** |  |  [optional] |
| **downloadSpeed** | **kotlin.Long** |  |  [optional] |
| **connections** | **kotlin.Int** |  |  [optional] |
| **error** | **kotlin.String** |  |  [optional] |
| **errorCode** | **kotlin.String** | 稳定下载错误分类；无错误时为空字符串 |  [optional] |
| **errorRetryable** | **kotlin.Boolean** |  |  [optional] |


<a id="Status"></a>
## Enum: status
| Name | Value |
| ---- | ----- |
| status | discovered, queued, submitting, waiting, active, paused, failed, excluded, completed |



