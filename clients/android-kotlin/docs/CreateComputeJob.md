
# CreateComputeJob

## Properties
| Name | Type | Description | Notes |
| ------------ | ------------- | ------------- | ------------- |
| **videoId** | **kotlin.String** |  |  |
| **type** | [**inline**](#Type) |  |  |
| **algorithmVersion** | **kotlin.String** |  |  [optional] |
| **options** | [**kotlin.collections.Map&lt;kotlin.String, kotlin.Any&gt;**](kotlin.Any.md) | 通用 JSON。&#x60;dedupe&#x60; 任务使用 &#x60;MediaDuplicateOptions&#x60;；&#x60;transcode&#x60; 任务使用 &#x60;TranscodeOptions&#x60;。为兼容历史任务保持开放对象。 |  [optional] |


<a id="Type"></a>
## Enum: type
| Name | Value |
| ---- | ----- |
| type | dedupe, transcode |



