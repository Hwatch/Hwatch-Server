
# MediaDuplicateMatch

## Properties
| Name | Type | Description | Notes |
| ------------ | ------------- | ------------- | ------------- |
| **videoId** | **kotlin.String** | 对应 MediaDuplicateOptions.candidates 中的候选 ID。 |  |
| **classification** | [**inline**](#Classification) |  |  |
| **backend** | **kotlin.String** |  |  [optional] |
| **algorithmVersion** | **kotlin.String** |  |  [optional] |
| **confidence** | [**java.math.BigDecimal**](java.math.BigDecimal.md) | 仅作参考；服务端不会据此自动删除媒体。 |  [optional] |
| **durationDelta** | **kotlin.Int** |  |  [optional] |
| **coverage** | [**java.math.BigDecimal**](java.math.BigDecimal.md) |  |  [optional] |
| **metrics** | [**kotlin.collections.Map&lt;kotlin.String, kotlin.Any&gt;**](kotlin.Any.md) | 结构化算法指标，原样保存，供后续校准回溯。 |  [optional] |


<a id="Classification"></a>
## Enum: classification
| Name | Value |
| ---- | ----- |
| classification | whole-video, trimmed-video, possible-match, different |



