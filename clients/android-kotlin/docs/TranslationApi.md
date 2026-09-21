# TranslationApi

All URIs are relative to *http://localhost*

| Method | HTTP request | Description |
| ------------- | ------------- | ------------- |
| [**exportTranslations**](TranslationApi.md#exportTranslations) | **GET** api/translations/export |  |
| [**importTranslations**](TranslationApi.md#importTranslations) | **POST** api/translations/import |  |
| [**refreshNFO**](TranslationApi.md#refreshNFO) | **POST** api/plex/refresh-nfo |  |





### Example
```kotlin
// Import classes:
//import app.hwatch.client.*
//import app.hwatch.client.infrastructure.*
//import app.hwatch.client.models.*

val apiClient = ApiClient()
val webService = apiClient.createWebservice(TranslationApi::class.java)
val language : kotlin.String = language_example // kotlin.String | 
val platform : kotlin.String = platform_example // kotlin.String | 
val subscription : kotlin.String = subscription_example // kotlin.String | 

val result : kotlin.collections.Map<kotlin.String, kotlin.Any> = webService.exportTranslations(language, platform, subscription)
```

### Parameters
| **language** | **kotlin.String**|  | [optional] [default to &quot;zh-CN&quot;] |
| **platform** | **kotlin.String**|  | [optional] |
| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **subscription** | **kotlin.String**|  | [optional] |

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
val webService = apiClient.createWebservice(TranslationApi::class.java)
val xCSRFToken : kotlin.String = xCSRFToken_example // kotlin.String | 
val requestBody : kotlin.collections.Map<kotlin.String, kotlin.Any> = Object // kotlin.collections.Map<kotlin.String, kotlin.Any> | 

val result : kotlin.collections.Map<kotlin.String, kotlin.Any> = webService.importTranslations(xCSRFToken, requestBody)
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
val webService = apiClient.createWebservice(TranslationApi::class.java)
val xCSRFToken : kotlin.String = xCSRFToken_example // kotlin.String | 
val refreshNFORequest : RefreshNFORequest =  // RefreshNFORequest | 

val result : kotlin.collections.Map<kotlin.String, kotlin.Any> = webService.refreshNFO(xCSRFToken, refreshNFORequest)
```

### Parameters
| **xCSRFToken** | **kotlin.String**|  | |
| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **refreshNFORequest** | [**RefreshNFORequest**](RefreshNFORequest.md)|  | [optional] |

### Return type

[**kotlin.collections.Map&lt;kotlin.String, kotlin.Any&gt;**](kotlin.Any.md)

### Authorization



### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

