
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
| **idempotencyKey** | **kotlin.String** | 可选幂等键；也可用 Idempotency-Key 头。相同键但请求内容不同返回 409 idempotency_conflict；已成功应用的重复请求返回此前结果（idempotent&#x3D;true），不会重复替换。 |  [optional] |


<a id="ApplyMode"></a>
## Enum: apply_mode
| Name | Value |
| ---- | ----- |
| applyMode | replace_source |



