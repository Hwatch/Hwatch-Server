# SubscriptionsApi

All URIs are relative to *http://localhost*

| Method | HTTP request | Description |
| ------------- | ------------- | ------------- |
| [**checkAll**](SubscriptionsApi.md#checkAll) | **POST** api/check |  |
| [**checkStatus**](SubscriptionsApi.md#checkStatus) | **GET** api/check/status |  |
| [**checkSubscription**](SubscriptionsApi.md#checkSubscription) | **POST** api/check/{slug} |  |
| [**createSubscription**](SubscriptionsApi.md#createSubscription) | **POST** api/subscriptions |  |
| [**deleteSubscription**](SubscriptionsApi.md#deleteSubscription) | **DELETE** api/subscriptions/{slug} |  |
| [**listPlatforms**](SubscriptionsApi.md#listPlatforms) | **GET** api/platforms |  |
| [**listSubscriptionLogs**](SubscriptionsApi.md#listSubscriptionLogs) | **GET** api/logs |  |
| [**listSubscriptions**](SubscriptionsApi.md#listSubscriptions) | **GET** api/subscriptions |  |
| [**searchCreators**](SubscriptionsApi.md#searchCreators) | **GET** api/creators/search |  |
| [**subscriptionAvatar**](SubscriptionsApi.md#subscriptionAvatar) | **GET** avatar/{slug} |  |
| [**updateSubscription**](SubscriptionsApi.md#updateSubscription) | **PATCH** api/subscriptions/{slug} |  |





### Example
```kotlin
// Import classes:
//import app.hwatch.client.*
//import app.hwatch.client.infrastructure.*
//import app.hwatch.client.models.*

val apiClient = ApiClient()
val webService = apiClient.createWebservice(SubscriptionsApi::class.java)
val xCSRFToken : kotlin.String = xCSRFToken_example // kotlin.String | 

val result : kotlin.collections.Map<kotlin.String, kotlin.Any> = webService.checkAll(xCSRFToken)
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




订阅检查进度（queued/active/running），供前端在批量检查完成后自动刷新。检查在服务端有并发上限（默认 3）。

### Example
```kotlin
// Import classes:
//import app.hwatch.client.*
//import app.hwatch.client.infrastructure.*
//import app.hwatch.client.models.*

val apiClient = ApiClient()
val webService = apiClient.createWebservice(SubscriptionsApi::class.java)

val result : kotlin.collections.Map<kotlin.String, kotlin.Any> = webService.checkStatus()
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
val webService = apiClient.createWebservice(SubscriptionsApi::class.java)
val slug : kotlin.String = slug_example // kotlin.String | 
val xCSRFToken : kotlin.String = xCSRFToken_example // kotlin.String | 

val result : kotlin.collections.Map<kotlin.String, kotlin.Any> = webService.checkSubscription(slug, xCSRFToken)
```

### Parameters
| **slug** | **kotlin.String**|  | |
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
val webService = apiClient.createWebservice(SubscriptionsApi::class.java)
val xCSRFToken : kotlin.String = xCSRFToken_example // kotlin.String | 
val createSubscriptionRequest : CreateSubscriptionRequest =  // CreateSubscriptionRequest | 

val result : kotlin.collections.Map<kotlin.String, kotlin.Any> = webService.createSubscription(xCSRFToken, createSubscriptionRequest)
```

### Parameters
| **xCSRFToken** | **kotlin.String**|  | |
| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **createSubscriptionRequest** | [**CreateSubscriptionRequest**](CreateSubscriptionRequest.md)|  | |

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
val webService = apiClient.createWebservice(SubscriptionsApi::class.java)
val slug : kotlin.String = slug_example // kotlin.String | 
val xCSRFToken : kotlin.String = xCSRFToken_example // kotlin.String | 

val result : kotlin.collections.Map<kotlin.String, kotlin.Any> = webService.deleteSubscription(slug, xCSRFToken)
```

### Parameters
| **slug** | **kotlin.String**|  | |
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
val webService = apiClient.createWebservice(SubscriptionsApi::class.java)

val result : kotlin.collections.Map<kotlin.String, kotlin.Any> = webService.listPlatforms()
```

### Parameters
This endpoint does not need any parameter.

### Return type

[**kotlin.collections.Map&lt;kotlin.String, kotlin.Any&gt;**](kotlin.Any.md)

### Authorization



### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json




订阅检查日志中心（只读）。记录每次检查的结果：发现/新增/入队数量或错误原因。按 id 倒序，最多保留最近 5000 条。

### Example
```kotlin
// Import classes:
//import app.hwatch.client.*
//import app.hwatch.client.infrastructure.*
//import app.hwatch.client.models.*

val apiClient = ApiClient()
val webService = apiClient.createWebservice(SubscriptionsApi::class.java)
val limit : kotlin.Int = 56 // kotlin.Int | 
val offset : kotlin.Int = 56 // kotlin.Int | 
val slug : kotlin.String = slug_example // kotlin.String | 
val level : kotlin.String = level_example // kotlin.String | 

val result : kotlin.collections.Map<kotlin.String, kotlin.Any> = webService.listSubscriptionLogs(limit, offset, slug, level)
```

### Parameters
| **limit** | **kotlin.Int**|  | [optional] [default to 200] |
| **offset** | **kotlin.Int**|  | [optional] |
| **slug** | **kotlin.String**|  | [optional] |
| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **level** | **kotlin.String**|  | [optional] [enum: info, error] |

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
val webService = apiClient.createWebservice(SubscriptionsApi::class.java)

val result : kotlin.collections.Map<kotlin.String, kotlin.Any> = webService.listSubscriptions()
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
val webService = apiClient.createWebservice(SubscriptionsApi::class.java)
val q : kotlin.String = q_example // kotlin.String | 
val platform : kotlin.String = platform_example // kotlin.String | 

val result : kotlin.collections.Map<kotlin.String, kotlin.Any> = webService.searchCreators(q, platform)
```

### Parameters
| **q** | **kotlin.String**|  | |
| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **platform** | **kotlin.String**|  | [optional] |

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
val webService = apiClient.createWebservice(SubscriptionsApi::class.java)
val slug : kotlin.String = slug_example // kotlin.String | 

webService.subscriptionAvatar(slug)
```

### Parameters
| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **slug** | **kotlin.String**|  | |

### Return type

null (empty response body)

### Authorization



### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: image/jpeg, image/png, image/svg+xml, application/json




### Example
```kotlin
// Import classes:
//import app.hwatch.client.*
//import app.hwatch.client.infrastructure.*
//import app.hwatch.client.models.*

val apiClient = ApiClient()
val webService = apiClient.createWebservice(SubscriptionsApi::class.java)
val slug : kotlin.String = slug_example // kotlin.String | 
val xCSRFToken : kotlin.String = xCSRFToken_example // kotlin.String | 
val updateSubscriptionRequest : UpdateSubscriptionRequest =  // UpdateSubscriptionRequest | 

val result : kotlin.collections.Map<kotlin.String, kotlin.Any> = webService.updateSubscription(slug, xCSRFToken, updateSubscriptionRequest)
```

### Parameters
| **slug** | **kotlin.String**|  | |
| **xCSRFToken** | **kotlin.String**|  | |
| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **updateSubscriptionRequest** | [**UpdateSubscriptionRequest**](UpdateSubscriptionRequest.md)|  | |

### Return type

[**kotlin.collections.Map&lt;kotlin.String, kotlin.Any&gt;**](kotlin.Any.md)

### Authorization



### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

