
# TransRunnerTranscodeResult

## Properties
| Name | Type | Description | Notes |
| ------------ | ------------- | ------------- | ------------- |
| **videoId** | **kotlin.String** |  |  |
| **outputRelativePath** | **kotlin.String** | 必须位于暂存目录 .transrunner/ 内。 |  |
| **runnerJobId** | **kotlin.String** |  |  [optional] |
| **sourceSha256** | **kotlin.String** | 提交源文件的 SHA-256；与当前不一致时返回 409 source_changed。 |  [optional] |
| **profile** | **kotlin.String** |  |  [optional] |



