# LibraryApi

All URIs are relative to *http://localhost*

| Method | HTTP request | Description |
| ------------- | ------------- | ------------- |
| [**dashboard**](LibraryApi.md#dashboard) | **GET** api/dashboard |  |
| [**importHTML**](LibraryApi.md#importHTML) | **POST** api/import/html |  |
| [**listDuplicateCandidates**](LibraryApi.md#listDuplicateCandidates) | **GET** api/duplicates |  |
| [**listVideos**](LibraryApi.md#listVideos) | **GET** api/videos |  |
| [**reviewDuplicateCandidate**](LibraryApi.md#reviewDuplicateCandidate) | **POST** api/duplicates/{id}/{action} |  |
| [**reviewDuplicateCandidatesBatch**](LibraryApi.md#reviewDuplicateCandidatesBatch) | **POST** api/duplicates/batch |  |
| [**streamMedia**](LibraryApi.md#streamMedia) | **GET** media/{id} |  |
| [**videoFacets**](LibraryApi.md#videoFacets) | **GET** api/videos/facets |  |
| [**videoThumbnail**](LibraryApi.md#videoThumbnail) | **GET** thumb/{id} |  |





### Example
```kotlin
// Import classes:
//import app.hwatch.client.*
//import app.hwatch.client.infrastructure.*
//import app.hwatch.client.models.*

val apiClient = ApiClient()
val webService = apiClient.createWebservice(LibraryApi::class.java)
val language : kotlin.String = language_example // kotlin.String | 

val result : kotlin.collections.Map<kotlin.String, kotlin.Any> = webService.dashboard(language)
```

### Parameters
| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **language** | **kotlin.String**|  | [optional] [default to &quot;zh-CN&quot;] |

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
val webService = apiClient.createWebservice(LibraryApi::class.java)
val xCSRFToken : kotlin.String = xCSRFToken_example // kotlin.String | 
val requestBody : kotlin.collections.Map<kotlin.String, kotlin.Any> = Object // kotlin.collections.Map<kotlin.String, kotlin.Any> | 

val result : kotlin.collections.Map<kotlin.String, kotlin.Any> = webService.importHTML(xCSRFToken, requestBody)
```

### Parameters
| **xCSRFToken** | **kotlin.String**|  | |
| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **requestBody** | [**kotlin.collections.Map&lt;kotlin.String, kotlin.Any&gt;**](kotlin.Any.md)|  | |

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
val webService = apiClient.createWebservice(LibraryApi::class.java)

val result : kotlin.collections.Map<kotlin.String, kotlin.Any> = webService.listDuplicateCandidates()
```

### Parameters
This endpoint does not need any parameter.

### Return type

[**kotlin.collections.Map&lt;kotlin.String, kotlin.Any&gt;**](kotlin.Any.md)

### Authorization



### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json




媒体分页列表。管理员会话可用；持有 read_media scope 的 Service Token（integrationBearer）也可只读调用以枚举媒体，此时响应中的 file_path/plex_path 会被清空。

### Example
```kotlin
// Import classes:
//import app.hwatch.client.*
//import app.hwatch.client.infrastructure.*
//import app.hwatch.client.models.*

val apiClient = ApiClient()
apiClient.setBearerToken("TOKEN")
val webService = apiClient.createWebservice(LibraryApi::class.java)
val limit : kotlin.Int = 56 // kotlin.Int | 
val offset : kotlin.Int = 56 // kotlin.Int | 
val status : kotlin.String = status_example // kotlin.String | 
val q : kotlin.String = q_example // kotlin.String | 
val scope : kotlin.String = scope_example // kotlin.String | 
val tags : kotlin.String = tags_example // kotlin.String | 
val excludeTags : kotlin.String = excludeTags_example // kotlin.String | 
val tagMode : kotlin.String = tagMode_example // kotlin.String | 
val creator : kotlin.String = creator_example // kotlin.String | 
val codec : kotlin.String = codec_example // kotlin.String | 
val minHeight : kotlin.Int = 56 // kotlin.Int | 
val sort : kotlin.String = sort_example // kotlin.String | 
val order : kotlin.String = order_example // kotlin.String | 
val language : kotlin.String = language_example // kotlin.String | 

val result : VideoPage = webService.listVideos(limit, offset, status, q, scope, tags, excludeTags, tagMode, creator, codec, minHeight, sort, order, language)
```

### Parameters
| **limit** | **kotlin.Int**|  | [optional] [default to 48] |
| **offset** | **kotlin.Int**|  | [optional] [default to 0] |
| **status** | **kotlin.String**|  | [optional] |
| **q** | **kotlin.String**|  | [optional] |
| **scope** | **kotlin.String**|  | [optional] [enum: all, original, translated] |
| **tags** | **kotlin.String**|  | [optional] |
| **excludeTags** | **kotlin.String**|  | [optional] |
| **tagMode** | **kotlin.String**|  | [optional] [enum: all, any] |
| **creator** | **kotlin.String**|  | [optional] |
| **codec** | **kotlin.String**|  | [optional] |
| **minHeight** | **kotlin.Int**|  | [optional] |
| **sort** | **kotlin.String**|  | [optional] |
| **order** | **kotlin.String**|  | [optional] [enum: asc, desc] |
| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **language** | **kotlin.String**|  | [optional] [default to &quot;zh-CN&quot;] |

### Return type

[**VideoPage**](VideoPage.md)

### Authorization


Configure integrationBearer:
    ApiClient().setBearerToken("TOKEN")

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
val webService = apiClient.createWebservice(LibraryApi::class.java)
val id : kotlin.Long = 789 // kotlin.Long | 
val action : kotlin.String = action_example // kotlin.String | 
val xCSRFToken : kotlin.String = xCSRFToken_example // kotlin.String | 

val result : kotlin.collections.Map<kotlin.String, kotlin.Any> = webService.reviewDuplicateCandidate(id, action, xCSRFToken)
```

### Parameters
| **id** | **kotlin.Long**|  | |
| **action** | **kotlin.String**|  | [enum: confirm, dismiss] |
| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **xCSRFToken** | **kotlin.String**|  | |

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
val webService = apiClient.createWebservice(LibraryApi::class.java)
val xCSRFToken : kotlin.String = xCSRFToken_example // kotlin.String | 
val reviewDuplicateCandidatesBatchRequest : ReviewDuplicateCandidatesBatchRequest =  // ReviewDuplicateCandidatesBatchRequest | 

val result : kotlin.collections.Map<kotlin.String, kotlin.Any> = webService.reviewDuplicateCandidatesBatch(xCSRFToken, reviewDuplicateCandidatesBatchRequest)
```

### Parameters
| **xCSRFToken** | **kotlin.String**|  | |
| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **reviewDuplicateCandidatesBatchRequest** | [**ReviewDuplicateCandidatesBatchRequest**](ReviewDuplicateCandidatesBatchRequest.md)|  | |

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
val webService = apiClient.createWebservice(LibraryApi::class.java)
val id : kotlin.String = id_example // kotlin.String | 

webService.streamMedia(id)
```

### Parameters
| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **id** | **kotlin.String**|  | |

### Return type

null (empty response body)

### Authorization



### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: Not defined




### Example
```kotlin
// Import classes:
//import app.hwatch.client.*
//import app.hwatch.client.infrastructure.*
//import app.hwatch.client.models.*

val apiClient = ApiClient()
val webService = apiClient.createWebservice(LibraryApi::class.java)

val result : kotlin.collections.Map<kotlin.String, kotlin.Any> = webService.videoFacets()
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
val webService = apiClient.createWebservice(LibraryApi::class.java)
val id : kotlin.String = id_example // kotlin.String | 

webService.videoThumbnail(id)
```

### Parameters
| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **id** | **kotlin.String**|  | |

### Return type

null (empty response body)

### Authorization



### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: Not defined

