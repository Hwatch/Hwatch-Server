# SettingsApi

All URIs are relative to *http://localhost*

| Method | HTTP request | Description |
| ------------- | ------------- | ------------- |
| [**createBackup**](SettingsApi.md#createBackup) | **POST** api/backups |  |
| [**getSettings**](SettingsApi.md#getSettings) | **GET** api/settings |  |
| [**health**](SettingsApi.md#health) | **GET** api/health |  |
| [**listBackups**](SettingsApi.md#listBackups) | **GET** api/backups |  |
| [**recoverySummary**](SettingsApi.md#recoverySummary) | **GET** api/recovery/latest |  |
| [**restoreBackup**](SettingsApi.md#restoreBackup) | **POST** api/backups/{name}/restore |  |
| [**updateSettings**](SettingsApi.md#updateSettings) | **PUT** api/settings |  |





### Example
```kotlin
// Import classes:
//import app.hwatch.client.*
//import app.hwatch.client.infrastructure.*
//import app.hwatch.client.models.*

val apiClient = ApiClient()
val webService = apiClient.createWebservice(SettingsApi::class.java)
val xCSRFToken : kotlin.String = xCSRFToken_example // kotlin.String | 

val result : kotlin.collections.Map<kotlin.String, kotlin.Any> = webService.createBackup(xCSRFToken)
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
val webService = apiClient.createWebservice(SettingsApi::class.java)

val result : kotlin.collections.Map<kotlin.String, kotlin.Any> = webService.getSettings()
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
val webService = apiClient.createWebservice(SettingsApi::class.java)

val result : Health = webService.health()
```

### Parameters
This endpoint does not need any parameter.

### Return type

[**Health**](Health.md)

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
val webService = apiClient.createWebservice(SettingsApi::class.java)

val result : kotlin.collections.Map<kotlin.String, kotlin.Any> = webService.listBackups()
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
val webService = apiClient.createWebservice(SettingsApi::class.java)

val result : RecoveryStatus = webService.recoverySummary()
```

### Parameters
This endpoint does not need any parameter.

### Return type

[**RecoveryStatus**](RecoveryStatus.md)

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
val webService = apiClient.createWebservice(SettingsApi::class.java)
val name : kotlin.String = name_example // kotlin.String | 
val xCSRFToken : kotlin.String = xCSRFToken_example // kotlin.String | 
val restoreBackupRequest : RestoreBackupRequest =  // RestoreBackupRequest | 

val result : kotlin.collections.Map<kotlin.String, kotlin.Any> = webService.restoreBackup(name, xCSRFToken, restoreBackupRequest)
```

### Parameters
| **name** | **kotlin.String**|  | |
| **xCSRFToken** | **kotlin.String**|  | |
| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **restoreBackupRequest** | [**RestoreBackupRequest**](RestoreBackupRequest.md)|  | |

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
val webService = apiClient.createWebservice(SettingsApi::class.java)
val xCSRFToken : kotlin.String = xCSRFToken_example // kotlin.String | 
val requestBody : kotlin.collections.Map<kotlin.String, kotlin.Any> = Object // kotlin.collections.Map<kotlin.String, kotlin.Any> | 

val result : kotlin.collections.Map<kotlin.String, kotlin.Any> = webService.updateSettings(xCSRFToken, requestBody)
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

