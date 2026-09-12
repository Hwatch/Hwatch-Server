# SubscriptionsApi

All URIs are relative to *http://localhost*

| Method | HTTP request | Description |
| ------------- | ------------- | ------------- |
| [**checkAll**](SubscriptionsApi.md#checkAll) | **POST** api/check |  |
| [**checkSubscription**](SubscriptionsApi.md#checkSubscription) | **POST** api/check/{slug} |  |
| [**createSubscription**](SubscriptionsApi.md#createSubscription) | **POST** api/subscriptions |  |
| [**deleteSubscription**](SubscriptionsApi.md#deleteSubscription) | **DELETE** api/subscriptions/{slug} |  |
| [**listPlatforms**](SubscriptionsApi.md#listPlatforms) | **GET** api/platforms |  |
| [**listSubscriptions**](SubscriptionsApi.md#listSubscriptions) | **GET** api/subscriptions |  |
| [**searchCreators**](SubscriptionsApi.md#searchCreators) | **GET** api/creators/search |  |
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
val xCSRFToken : kotlin.String = xCSRFToken_example // kotlin.String | 

val result : kotlin.collections.Map<kotlin.String, kotlin.Any> = webService.updateSubscription(slug, xCSRFToken)
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

