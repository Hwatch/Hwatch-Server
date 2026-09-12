
# Notification

## Properties
| Name | Type | Description | Notes |
| ------------ | ------------- | ------------- | ------------- |
| **id** | **kotlin.Long** |  |  |
| **code** | [**inline**](#Code) |  |  |
| **severity** | [**inline**](#Severity) |  |  |
| **message** | **kotlin.String** |  |  |
| **createdAt** | [**java.time.OffsetDateTime**](java.time.OffsetDateTime.md) |  |  |
| **readAt** | **kotlin.String** | 未读时为空字符串 |  |


<a id="Code"></a>
## Enum: code
| Name | Value |
| ---- | ----- |
| code | storage_unavailable, storage_low, consecutive_download_failures, downloader_unavailable |


<a id="Severity"></a>
## Enum: severity
| Name | Value |
| ---- | ----- |
| severity | warning, error |



