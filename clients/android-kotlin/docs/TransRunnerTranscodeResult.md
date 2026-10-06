
# TransRunnerTranscodeResult

## Properties
| Name | Type | Description | Notes |
| ------------ | ------------- | ------------- | ------------- |
| **videoId** | **kotlin.String** |  |  |
| **outputRelativePath** | **kotlin.String** | 必须位于暂存目录 .transrunner/ 内。 |  |
| **runnerJobId** | **kotlin.String** |  |  [optional] |
| **sourceSha256** | **kotlin.String** | 提交源文件的 SHA-256；与当前不一致时返回 409 source_changed。 |  [optional] |
| **profile** | **kotlin.String** |  |  [optional] |
| **applyMode** | [**inline**](#ApplyMode) | 省略时仅记录 pending；为 replace_source 时在校验通过后原子替换源文件。 |  [optional] |


<a id="ApplyMode"></a>
## Enum: apply_mode
| Name | Value |
| ---- | ----- |
| applyMode | replace_source |



