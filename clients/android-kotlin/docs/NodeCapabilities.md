
# NodeCapabilities

## Properties
| Name | Type | Description | Notes |
| ------------ | ------------- | ------------- | ------------- |
| **dedupe** | **kotlin.Boolean** |  |  |
| **transcode** | **kotlin.Boolean** |  |  |
| **mediaDuplicate** | **kotlin.Boolean** | 支持视频内容重复比较（media-fingerprint-v1）。 |  [optional] |
| **videoSignature** | **kotlin.Boolean** | 额外支持 FFmpeg MPEG-7 signature，属可选能力，不作为任务硬性要求。 |  [optional] |
| **videoToolbox** | **kotlin.Boolean** |  |  [optional] |
| **encoders** | **kotlin.collections.List&lt;kotlin.String&gt;** |  |  [optional] |
| **maxDedupeJobs** | **kotlin.Int** |  |  [optional] |
| **maxTranscodeJobs** | **kotlin.Int** |  |  [optional] |



