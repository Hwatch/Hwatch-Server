
# Video

## Properties
| Name | Type | Description | Notes |
| ------------ | ------------- | ------------- | ------------- |
| **videoId** | **kotlin.String** |  |  |
| **subscriptionSlug** | **kotlin.String** |  |  |
| **title** | **kotlin.String** |  |  |
| **status** | [**inline**](#Status) |  |  |
| **translatedTitle** | **kotlin.String** |  |  [optional] |
| **downloaderId** | **kotlin.String** | 未分配时为空字符串，已分配时为 aria2 或 wget |  [optional] |
| **externalTaskId** | **kotlin.String** |  |  [optional] |
| **bytesDownloaded** | **kotlin.Long** |  |  [optional] |
| **totalBytes** | **kotlin.Long** |  |  [optional] |
| **downloadSpeed** | **kotlin.Long** |  |  [optional] |
| **connections** | **kotlin.Int** |  |  [optional] |


<a id="Status"></a>
## Enum: status
| Name | Value |
| ---- | ----- |
| status | discovered, queued, waiting, active, paused, failed, excluded, completed |



