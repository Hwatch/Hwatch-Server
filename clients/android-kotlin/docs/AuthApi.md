# AuthApi

All URIs are relative to *http://localhost*

| Method | HTTP request | Description |
| ------------- | ------------- | ------------- |
| [**authMe**](AuthApi.md#authMe) | **GET** api/auth/me |  |
| [**authStatus**](AuthApi.md#authStatus) | **GET** api/auth/status |  |
| [**login**](AuthApi.md#login) | **POST** api/auth/login |  |
| [**logout**](AuthApi.md#logout) | **POST** api/auth/logout |  |
| [**setupAdmin**](AuthApi.md#setupAdmin) | **POST** api/auth/setup |  |





### Example
```kotlin
// Import classes:
//import app.hwatch.client.*
//import app.hwatch.client.infrastructure.*
//import app.hwatch.client.models.*

val apiClient = ApiClient()
val webService = apiClient.createWebservice(AuthApi::class.java)

val result : kotlin.collections.Map<kotlin.String, kotlin.Any> = webService.authMe()
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
val webService = apiClient.createWebservice(AuthApi::class.java)

val result : kotlin.collections.Map<kotlin.String, kotlin.Any> = webService.authStatus()
```

### Parameters
This endpoint does not need any parameter.

### Return type

[**kotlin.collections.Map&lt;kotlin.String, kotlin.Any&gt;**](kotlin.Any.md)

### Authorization

No authorization required

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
val webService = apiClient.createWebservice(AuthApi::class.java)
val credentials : Credentials =  // Credentials | 

val result : kotlin.collections.Map<kotlin.String, kotlin.Any> = webService.login(credentials)
```

### Parameters
| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **credentials** | [**Credentials**](Credentials.md)|  | |

### Return type

[**kotlin.collections.Map&lt;kotlin.String, kotlin.Any&gt;**](kotlin.Any.md)

### Authorization

No authorization required

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
val webService = apiClient.createWebservice(AuthApi::class.java)
val xCSRFToken : kotlin.String = xCSRFToken_example // kotlin.String | 

val result : kotlin.collections.Map<kotlin.String, kotlin.Any> = webService.logout(xCSRFToken)
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
val webService = apiClient.createWebservice(AuthApi::class.java)
val credentials : Credentials =  // Credentials | 

val result : kotlin.collections.Map<kotlin.String, kotlin.Any> = webService.setupAdmin(credentials)
```

### Parameters
| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **credentials** | [**Credentials**](Credentials.md)|  | |

### Return type

[**kotlin.collections.Map&lt;kotlin.String, kotlin.Any&gt;**](kotlin.Any.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

