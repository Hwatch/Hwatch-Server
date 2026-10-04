# NodesApi

All URIs are relative to *http://localhost*

| Method | HTTP request | Description |
| ------------- | ------------- | ------------- |
| [**cancelComputeJob**](NodesApi.md#cancelComputeJob) | **POST** api/compute/jobs/{id}/cancel |  |
| [**commitTranscodeJob**](NodesApi.md#commitTranscodeJob) | **POST** api/compute/jobs/{id}/commit |  |
| [**createComputeJob**](NodesApi.md#createComputeJob) | **POST** api/compute/jobs |  |
| [**createComputeNode**](NodesApi.md#createComputeNode) | **POST** api/nodes |  |
| [**listComputeJobs**](NodesApi.md#listComputeJobs) | **GET** api/compute/jobs |  |
| [**listComputeNodes**](NodesApi.md#listComputeNodes) | **GET** api/nodes |  |
| [**revokeComputeNode**](NodesApi.md#revokeComputeNode) | **DELETE** api/nodes/{id} |  |





### Example
```kotlin
// Import classes:
//import app.hwatch.client.*
//import app.hwatch.client.infrastructure.*
//import app.hwatch.client.models.*

val apiClient = ApiClient()
val webService = apiClient.createWebservice(NodesApi::class.java)
val id : kotlin.String = id_example // kotlin.String | 
val xCSRFToken : kotlin.String = xCSRFToken_example // kotlin.String | 

val result : kotlin.collections.Map<kotlin.String, kotlin.Any> = webService.cancelComputeJob(id, xCSRFToken)
```

### Parameters
| **id** | **kotlin.String**|  | |
| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **xCSRFToken** | **kotlin.String**|  | |

### Return type

[**kotlin.collections.Map&lt;kotlin.String, kotlin.Any&gt;**](kotlin.Any.md)

### Authorization



### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json




兼容接口。节点回传的转码产物现在会在验证通过后自动原子提交，管理员无需确认；该端点仅用于仍处于 &#x60;verifying&#x60; 的旧任务。

### Example
```kotlin
// Import classes:
//import app.hwatch.client.*
//import app.hwatch.client.infrastructure.*
//import app.hwatch.client.models.*

val apiClient = ApiClient()
val webService = apiClient.createWebservice(NodesApi::class.java)
val id : kotlin.String = id_example // kotlin.String | 
val xCSRFToken : kotlin.String = xCSRFToken_example // kotlin.String | 
val commitTranscodeJobRequest : CommitTranscodeJobRequest =  // CommitTranscodeJobRequest | 

val result : kotlin.collections.Map<kotlin.String, kotlin.Any> = webService.commitTranscodeJob(id, xCSRFToken, commitTranscodeJobRequest)
```

### Parameters
| **id** | **kotlin.String**|  | |
| **xCSRFToken** | **kotlin.String**|  | |
| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **commitTranscodeJobRequest** | [**CommitTranscodeJobRequest**](CommitTranscodeJobRequest.md)|  | |

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
val webService = apiClient.createWebservice(NodesApi::class.java)
val xCSRFToken : kotlin.String = xCSRFToken_example // kotlin.String | 
val createComputeJob : CreateComputeJob =  // CreateComputeJob | 

val result : kotlin.collections.Map<kotlin.String, kotlin.Any> = webService.createComputeJob(xCSRFToken, createComputeJob)
```

### Parameters
| **xCSRFToken** | **kotlin.String**|  | |
| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **createComputeJob** | [**CreateComputeJob**](CreateComputeJob.md)|  | |

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
val webService = apiClient.createWebservice(NodesApi::class.java)
val xCSRFToken : kotlin.String = xCSRFToken_example // kotlin.String | 
val createComputeNode : CreateComputeNode =  // CreateComputeNode | 

val result : kotlin.collections.Map<kotlin.String, kotlin.Any> = webService.createComputeNode(xCSRFToken, createComputeNode)
```

### Parameters
| **xCSRFToken** | **kotlin.String**|  | |
| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **createComputeNode** | [**CreateComputeNode**](CreateComputeNode.md)|  | |

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
val webService = apiClient.createWebservice(NodesApi::class.java)
val limit : kotlin.Int = 56 // kotlin.Int | 
val offset : kotlin.Int = 56 // kotlin.Int | 

val result : kotlin.collections.Map<kotlin.String, kotlin.Any> = webService.listComputeJobs(limit, offset)
```

### Parameters
| **limit** | **kotlin.Int**|  | [optional] [default to 200] |
| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **offset** | **kotlin.Int**|  | [optional] [default to 0] |

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
val webService = apiClient.createWebservice(NodesApi::class.java)

val result : kotlin.collections.Map<kotlin.String, kotlin.Any> = webService.listComputeNodes()
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
val webService = apiClient.createWebservice(NodesApi::class.java)
val id : kotlin.String = id_example // kotlin.String | 
val xCSRFToken : kotlin.String = xCSRFToken_example // kotlin.String | 

val result : kotlin.collections.Map<kotlin.String, kotlin.Any> = webService.revokeComputeNode(id, xCSRFToken)
```

### Parameters
| **id** | **kotlin.String**|  | |
| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **xCSRFToken** | **kotlin.String**|  | |

### Return type

[**kotlin.collections.Map&lt;kotlin.String, kotlin.Any&gt;**](kotlin.Any.md)

### Authorization



### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

