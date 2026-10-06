# TransRunnerApi

All URIs are relative to *http://localhost*

| Method | HTTP request | Description |
| ------------- | ------------- | ------------- |
| [**createTransRunnerToken**](TransRunnerApi.md#createTransRunnerToken) | **POST** api/integrations/transrunner/token |  |
| [**getTransRunnerToken**](TransRunnerApi.md#getTransRunnerToken) | **GET** api/integrations/transrunner/token |  |
| [**revokeTransRunnerToken**](TransRunnerApi.md#revokeTransRunnerToken) | **DELETE** api/integrations/transrunner/token |  |
| [**submitDedupeToTransRunner**](TransRunnerApi.md#submitDedupeToTransRunner) | **POST** api/integrations/transrunner/submit/dedupe |  |
| [**submitTranscodeToTransRunner**](TransRunnerApi.md#submitTranscodeToTransRunner) | **POST** api/integrations/transrunner/submit/transcode |  |
| [**testTransRunnerConnection**](TransRunnerApi.md#testTransRunnerConnection) | **POST** api/integrations/transrunner/test |  |
| [**transRunnerDedupeCandidates**](TransRunnerApi.md#transRunnerDedupeCandidates) | **GET** api/transrunner/v1/media/{video_id}/dedupe-candidates |  |
| [**transRunnerDuplicateResults**](TransRunnerApi.md#transRunnerDuplicateResults) | **POST** api/transrunner/v1/duplicate-results |  |
| [**transRunnerMedia**](TransRunnerApi.md#transRunnerMedia) | **GET** api/transrunner/v1/media/{video_id} |  |
| [**transRunnerTranscodeResults**](TransRunnerApi.md#transRunnerTranscodeResults) | **POST** api/transrunner/v1/transcode-results |  |





### Example
```kotlin
// Import classes:
//import app.hwatch.client.*
//import app.hwatch.client.infrastructure.*
//import app.hwatch.client.models.*

val apiClient = ApiClient()
val webService = apiClient.createWebservice(TransRunnerApi::class.java)
val xCSRFToken : kotlin.String = xCSRFToken_example // kotlin.String | 
val createTransRunnerTokenRequest : CreateTransRunnerTokenRequest =  // CreateTransRunnerTokenRequest | 

val result : kotlin.collections.Map<kotlin.String, kotlin.Any> = webService.createTransRunnerToken(xCSRFToken, createTransRunnerTokenRequest)
```

### Parameters
| **xCSRFToken** | **kotlin.String**|  | |
| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **createTransRunnerTokenRequest** | [**CreateTransRunnerTokenRequest**](CreateTransRunnerTokenRequest.md)|  | [optional] |

### Return type

[**kotlin.collections.Map&lt;kotlin.String, kotlin.Any&gt;**](kotlin.Any.md)

### Authorization



### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json




### Example
```kotlin
// Import classes:
//import app.hwatch.client.*
//import app.hwatch.client.infrastructure.*
//import app.hwatch.client.models.*

val apiClient = ApiClient()
val webService = apiClient.createWebservice(TransRunnerApi::class.java)

val result : kotlin.collections.Map<kotlin.String, kotlin.Any> = webService.getTransRunnerToken()
```

### Parameters
This endpoint does not need any parameter.

### Return type

[**kotlin.collections.Map&lt;kotlin.String, kotlin.Any&gt;**](kotlin.Any.md)

### Authorization



### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json




### Example
```kotlin
// Import classes:
//import app.hwatch.client.*
//import app.hwatch.client.infrastructure.*
//import app.hwatch.client.models.*

val apiClient = ApiClient()
val webService = apiClient.createWebservice(TransRunnerApi::class.java)
val xCSRFToken : kotlin.String = xCSRFToken_example // kotlin.String | 

val result : kotlin.collections.Map<kotlin.String, kotlin.Any> = webService.revokeTransRunnerToken(xCSRFToken)
```

### Parameters
| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **xCSRFToken** | **kotlin.String**|  | |

### Return type

[**kotlin.collections.Map&lt;kotlin.String, kotlin.Any&gt;**](kotlin.Any.md)

### Authorization



### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json




提交内容去重任务到 TransRunner（kind&#x3D;media_duplicate，input.type&#x3D;hwatch）。TransRunner 自行调用 HWatch 的 dedupe-candidates 获取候选；HWatch 不发送候选、绝对路径或算法参数，也不轮询进度。

### Example
```kotlin
// Import classes:
//import app.hwatch.client.*
//import app.hwatch.client.infrastructure.*
//import app.hwatch.client.models.*

val apiClient = ApiClient()
val webService = apiClient.createWebservice(TransRunnerApi::class.java)
val xCSRFToken : kotlin.String = xCSRFToken_example // kotlin.String | 
val submitDedupeToTransRunnerRequest : SubmitDedupeToTransRunnerRequest =  // SubmitDedupeToTransRunnerRequest | 

val result : kotlin.collections.Map<kotlin.String, kotlin.Any> = webService.submitDedupeToTransRunner(xCSRFToken, submitDedupeToTransRunnerRequest)
```

### Parameters
| **xCSRFToken** | **kotlin.String**|  | |
| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **submitDedupeToTransRunnerRequest** | [**SubmitDedupeToTransRunnerRequest**](SubmitDedupeToTransRunnerRequest.md)|  | |

### Return type

[**kotlin.collections.Map&lt;kotlin.String, kotlin.Any&gt;**](kotlin.Any.md)

### Authorization



### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json




通过 TransRunner 的 /api/v1/jobs 提交转码任务，使用 HWatch 相对路径输入（绝不发送绝对路径）。HWatch 不镜像 TransRunner 的队列、进度或状态。

### Example
```kotlin
// Import classes:
//import app.hwatch.client.*
//import app.hwatch.client.infrastructure.*
//import app.hwatch.client.models.*

val apiClient = ApiClient()
val webService = apiClient.createWebservice(TransRunnerApi::class.java)
val xCSRFToken : kotlin.String = xCSRFToken_example // kotlin.String | 
val submitTranscodeToTransRunnerRequest : SubmitTranscodeToTransRunnerRequest =  // SubmitTranscodeToTransRunnerRequest | 

val result : kotlin.collections.Map<kotlin.String, kotlin.Any> = webService.submitTranscodeToTransRunner(xCSRFToken, submitTranscodeToTransRunnerRequest)
```

### Parameters
| **xCSRFToken** | **kotlin.String**|  | |
| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **submitTranscodeToTransRunnerRequest** | [**SubmitTranscodeToTransRunnerRequest**](SubmitTranscodeToTransRunnerRequest.md)|  | |

### Return type

[**kotlin.collections.Map&lt;kotlin.String, kotlin.Any&gt;**](kotlin.Any.md)

### Authorization



### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json




调用 TransRunner 的 /health 与 /api/v1/capabilities 验证地址与 Token；不暴露 Token。

### Example
```kotlin
// Import classes:
//import app.hwatch.client.*
//import app.hwatch.client.infrastructure.*
//import app.hwatch.client.models.*

val apiClient = ApiClient()
val webService = apiClient.createWebservice(TransRunnerApi::class.java)
val xCSRFToken : kotlin.String = xCSRFToken_example // kotlin.String | 

val result : kotlin.collections.Map<kotlin.String, kotlin.Any> = webService.testTransRunnerConnection(xCSRFToken)
```

### Parameters
| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **xCSRFToken** | **kotlin.String**|  | |

### Return type

[**kotlin.collections.Map&lt;kotlin.String, kotlin.Any&gt;**](kotlin.Any.md)

### Authorization



### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json




需要 read_dedupe_candidates。候选由 HWatch 服务端筛选，调用方不扫描全库。

### Example
```kotlin
// Import classes:
//import app.hwatch.client.*
//import app.hwatch.client.infrastructure.*
//import app.hwatch.client.models.*

val apiClient = ApiClient()
apiClient.setBearerToken("TOKEN")
val webService = apiClient.createWebservice(TransRunnerApi::class.java)
val videoId : kotlin.String = videoId_example // kotlin.String | 
val limit : kotlin.Int = 56 // kotlin.Int | 

val result : TransRunnerDedupeCandidates = webService.transRunnerDedupeCandidates(videoId, limit)
```

### Parameters
| **videoId** | **kotlin.String**|  | |
| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **limit** | **kotlin.Int**|  | [optional] [default to 20] |

### Return type

[**TransRunnerDedupeCandidates**](TransRunnerDedupeCandidates.md)

### Authorization


Configure integrationBearer:
    ApiClient().setBearerToken("TOKEN")

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json




需要 write_duplicate_results。仅写入重复候选，绝不删除或替换媒体。

### Example
```kotlin
// Import classes:
//import app.hwatch.client.*
//import app.hwatch.client.infrastructure.*
//import app.hwatch.client.models.*

val apiClient = ApiClient()
apiClient.setBearerToken("TOKEN")
val webService = apiClient.createWebservice(TransRunnerApi::class.java)
val transRunnerDuplicateResult : TransRunnerDuplicateResult =  // TransRunnerDuplicateResult | 

val result : kotlin.collections.Map<kotlin.String, kotlin.Any> = webService.transRunnerDuplicateResults(transRunnerDuplicateResult)
```

### Parameters
| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **transRunnerDuplicateResult** | [**TransRunnerDuplicateResult**](TransRunnerDuplicateResult.md)|  | |

### Return type

[**kotlin.collections.Map&lt;kotlin.String, kotlin.Any&gt;**](kotlin.Any.md)

### Authorization


Configure integrationBearer:
    ApiClient().setBearerToken("TOKEN")

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json




需要 read_media。只返回相对媒体路径与媒体元数据，不返回绝对路径或凭据。

### Example
```kotlin
// Import classes:
//import app.hwatch.client.*
//import app.hwatch.client.infrastructure.*
//import app.hwatch.client.models.*

val apiClient = ApiClient()
apiClient.setBearerToken("TOKEN")
val webService = apiClient.createWebservice(TransRunnerApi::class.java)
val videoId : kotlin.String = videoId_example // kotlin.String | 

val result : TransRunnerMedia = webService.transRunnerMedia(videoId)
```

### Parameters
| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **videoId** | **kotlin.String**|  | |

### Return type

[**TransRunnerMedia**](TransRunnerMedia.md)

### Authorization


Configure integrationBearer:
    ApiClient().setBearerToken("TOKEN")

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json




需要 write_transcode_results。接收暂存产物并由 HWatch 用 ffprobe 验证，只记录 pending 结果，不替换源文件。

### Example
```kotlin
// Import classes:
//import app.hwatch.client.*
//import app.hwatch.client.infrastructure.*
//import app.hwatch.client.models.*

val apiClient = ApiClient()
apiClient.setBearerToken("TOKEN")
val webService = apiClient.createWebservice(TransRunnerApi::class.java)
val transRunnerTranscodeResult : TransRunnerTranscodeResult =  // TransRunnerTranscodeResult | 

val result : kotlin.collections.Map<kotlin.String, kotlin.Any> = webService.transRunnerTranscodeResults(transRunnerTranscodeResult)
```

### Parameters
| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **transRunnerTranscodeResult** | [**TransRunnerTranscodeResult**](TransRunnerTranscodeResult.md)|  | |

### Return type

[**kotlin.collections.Map&lt;kotlin.String, kotlin.Any&gt;**](kotlin.Any.md)

### Authorization


Configure integrationBearer:
    ApiClient().setBearerToken("TOKEN")

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

