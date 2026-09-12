# DownloadsApi

All URIs are relative to *http://localhost*

| Method | HTTP request | Description |
| ------------- | ------------- | ------------- |
| [**downloadAll**](DownloadsApi.md#downloadAll) | **POST** api/videos/download-all |  |
| [**downloadEvents**](DownloadsApi.md#downloadEvents) | **GET** api/events |  |
| [**listTasks**](DownloadsApi.md#listTasks) | **GET** api/tasks |  |
| [**toggleDownloads**](DownloadsApi.md#toggleDownloads) | **POST** api/downloads/toggle |  |
| [**videoAction**](DownloadsApi.md#videoAction) | **POST** api/videos/{id}/{action} |  |





### Example
```kotlin
// Import classes:
//import app.hwatch.client.*
//import app.hwatch.client.infrastructure.*
//import app.hwatch.client.models.*

val apiClient = ApiClient()
val webService = apiClient.createWebservice(DownloadsApi::class.java)
val xCSRFToken : kotlin.String = xCSRFToken_example // kotlin.String | 
val subscription : kotlin.String = subscription_example // kotlin.String | 

val result : kotlin.collections.Map<kotlin.String, kotlin.Any> = webService.downloadAll(xCSRFToken, subscription)
```

### Parameters
| **xCSRFToken** | **kotlin.String**|  | |
| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **subscription** | **kotlin.String**|  | [optional] |

### Return type

[**kotlin.collections.Map&lt;kotlin.String, kotlin.Any&gt;**](kotlin.Any.md)

### Authorization



### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json




text/event-stream。客户端断线后自动重连，并通过 GET /api/dashboard 与 GET /api/tasks 恢复全量状态。

### Example
```kotlin
// Import classes:
//import app.hwatch.client.*
//import app.hwatch.client.infrastructure.*
//import app.hwatch.client.models.*

val apiClient = ApiClient()
val webService = apiClient.createWebservice(DownloadsApi::class.java)

val result : kotlin.String = webService.downloadEvents()
```

### Parameters
This endpoint does not need any parameter.

### Return type

**kotlin.String**

### Authorization



### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: text/event-stream




### Example
```kotlin
// Import classes:
//import app.hwatch.client.*
//import app.hwatch.client.infrastructure.*
//import app.hwatch.client.models.*

val apiClient = ApiClient()
val webService = apiClient.createWebservice(DownloadsApi::class.java)
val limit : kotlin.Int = 56 // kotlin.Int | 
val offset : kotlin.Int = 56 // kotlin.Int | 
val status : kotlin.String = status_example // kotlin.String | 
val language : kotlin.String = language_example // kotlin.String | 

val result : VideoPage = webService.listTasks(limit, offset, status, language)
```

### Parameters
| **limit** | **kotlin.Int**|  | [optional] [default to 48] |
| **offset** | **kotlin.Int**|  | [optional] [default to 0] |
| **status** | **kotlin.String**|  | [optional] |
| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **language** | **kotlin.String**|  | [optional] [default to &quot;zh-CN&quot;] |

### Return type

[**VideoPage**](VideoPage.md)

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
val webService = apiClient.createWebservice(DownloadsApi::class.java)
val xCSRFToken : kotlin.String = xCSRFToken_example // kotlin.String | 
val toggleDownloadsRequest : ToggleDownloadsRequest =  // ToggleDownloadsRequest | 

val result : kotlin.collections.Map<kotlin.String, kotlin.Any> = webService.toggleDownloads(xCSRFToken, toggleDownloadsRequest)
```

### Parameters
| **xCSRFToken** | **kotlin.String**|  | |
| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **toggleDownloadsRequest** | [**ToggleDownloadsRequest**](ToggleDownloadsRequest.md)|  | |

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
val webService = apiClient.createWebservice(DownloadsApi::class.java)
val id : kotlin.String = id_example // kotlin.String | 
val action : kotlin.String = action_example // kotlin.String | 
val xCSRFToken : kotlin.String = xCSRFToken_example // kotlin.String | 

val result : kotlin.collections.Map<kotlin.String, kotlin.Any> = webService.videoAction(id, action, xCSRFToken)
```

### Parameters
| **id** | **kotlin.String**|  | |
| **action** | **kotlin.String**|  | [enum: download, retry, pause, resume, delete] |
| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **xCSRFToken** | **kotlin.String**|  | |

### Return type

[**kotlin.collections.Map&lt;kotlin.String, kotlin.Any&gt;**](kotlin.Any.md)

### Authorization



### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

