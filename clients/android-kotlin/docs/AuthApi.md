# AuthApi

All URIs are relative to *http://localhost*

| Method | HTTP request | Description |
| ------------- | ------------- | ------------- |
| [**authMe**](AuthApi.md#authMe) | **GET** api/auth/me |  |
| [**authStatus**](AuthApi.md#authStatus) | **GET** api/auth/status |  |
| [**changeOwnPassword**](AuthApi.md#changeOwnPassword) | **POST** api/auth/password |  |
| [**createUser**](AuthApi.md#createUser) | **POST** api/users |  |
| [**deleteMyAvatar**](AuthApi.md#deleteMyAvatar) | **DELETE** api/auth/avatar |  |
| [**deleteUser**](AuthApi.md#deleteUser) | **DELETE** api/users/{id} |  |
| [**deleteUserAvatar**](AuthApi.md#deleteUserAvatar) | **DELETE** api/users/{id}/avatar |  |
| [**listUsers**](AuthApi.md#listUsers) | **GET** api/users |  |
| [**login**](AuthApi.md#login) | **POST** api/auth/login |  |
| [**logout**](AuthApi.md#logout) | **POST** api/auth/logout |  |
| [**setUserPassword**](AuthApi.md#setUserPassword) | **POST** api/users/{id}/password |  |
| [**setupAdmin**](AuthApi.md#setupAdmin) | **POST** api/auth/setup |  |
| [**updateUserRole**](AuthApi.md#updateUserRole) | **PATCH** api/users/{id} |  |
| [**uploadMyAvatar**](AuthApi.md#uploadMyAvatar) | **POST** api/auth/avatar |  |
| [**uploadUserAvatar**](AuthApi.md#uploadUserAvatar) | **POST** api/users/{id}/avatar |  |
| [**userAvatar**](AuthApi.md#userAvatar) | **GET** avatar/user/{id} |  |





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




修改当前登录账号的密码（任意已登录用户）。

### Example
```kotlin
// Import classes:
//import app.hwatch.client.*
//import app.hwatch.client.infrastructure.*
//import app.hwatch.client.models.*

val apiClient = ApiClient()
val webService = apiClient.createWebservice(AuthApi::class.java)
val xCSRFToken : kotlin.String = xCSRFToken_example // kotlin.String | 
val changeOwnPasswordRequest : ChangeOwnPasswordRequest =  // ChangeOwnPasswordRequest | 

val result : kotlin.collections.Map<kotlin.String, kotlin.Any> = webService.changeOwnPassword(xCSRFToken, changeOwnPasswordRequest)
```

### Parameters
| **xCSRFToken** | **kotlin.String**|  | |
| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **changeOwnPasswordRequest** | [**ChangeOwnPasswordRequest**](ChangeOwnPasswordRequest.md)|  | |

### Return type

[**kotlin.collections.Map&lt;kotlin.String, kotlin.Any&gt;**](kotlin.Any.md)

### Authorization



### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json




创建用户（仅管理员）。role 为 admin 或 viewer，默认 viewer。

### Example
```kotlin
// Import classes:
//import app.hwatch.client.*
//import app.hwatch.client.infrastructure.*
//import app.hwatch.client.models.*

val apiClient = ApiClient()
val webService = apiClient.createWebservice(AuthApi::class.java)
val xCSRFToken : kotlin.String = xCSRFToken_example // kotlin.String | 
val createUserRequest : CreateUserRequest =  // CreateUserRequest | 

val result : kotlin.collections.Map<kotlin.String, kotlin.Any> = webService.createUser(xCSRFToken, createUserRequest)
```

### Parameters
| **xCSRFToken** | **kotlin.String**|  | |
| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **createUserRequest** | [**CreateUserRequest**](CreateUserRequest.md)|  | |

### Return type

[**kotlin.collections.Map&lt;kotlin.String, kotlin.Any&gt;**](kotlin.Any.md)

### Authorization



### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json




清除当前用户头像。

### Example
```kotlin
// Import classes:
//import app.hwatch.client.*
//import app.hwatch.client.infrastructure.*
//import app.hwatch.client.models.*

val apiClient = ApiClient()
val webService = apiClient.createWebservice(AuthApi::class.java)
val xCSRFToken : kotlin.String = xCSRFToken_example // kotlin.String | 

val result : kotlin.collections.Map<kotlin.String, kotlin.Any> = webService.deleteMyAvatar(xCSRFToken)
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




删除用户（仅管理员，不能删除自己或最后一个管理员）。

### Example
```kotlin
// Import classes:
//import app.hwatch.client.*
//import app.hwatch.client.infrastructure.*
//import app.hwatch.client.models.*

val apiClient = ApiClient()
val webService = apiClient.createWebservice(AuthApi::class.java)
val id : kotlin.Int = 56 // kotlin.Int | 
val xCSRFToken : kotlin.String = xCSRFToken_example // kotlin.String | 

val result : kotlin.collections.Map<kotlin.String, kotlin.Any> = webService.deleteUser(id, xCSRFToken)
```

### Parameters
| **id** | **kotlin.Int**|  | |
| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **xCSRFToken** | **kotlin.String**|  | |

### Return type

[**kotlin.collections.Map&lt;kotlin.String, kotlin.Any&gt;**](kotlin.Any.md)

### Authorization



### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json




清除指定用户头像（仅管理员）。

### Example
```kotlin
// Import classes:
//import app.hwatch.client.*
//import app.hwatch.client.infrastructure.*
//import app.hwatch.client.models.*

val apiClient = ApiClient()
val webService = apiClient.createWebservice(AuthApi::class.java)
val id : kotlin.Int = 56 // kotlin.Int | 
val xCSRFToken : kotlin.String = xCSRFToken_example // kotlin.String | 

val result : kotlin.collections.Map<kotlin.String, kotlin.Any> = webService.deleteUserAvatar(id, xCSRFToken)
```

### Parameters
| **id** | **kotlin.Int**|  | |
| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **xCSRFToken** | **kotlin.String**|  | |

### Return type

[**kotlin.collections.Map&lt;kotlin.String, kotlin.Any&gt;**](kotlin.Any.md)

### Authorization



### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json




列出用户（仅管理员）。

### Example
```kotlin
// Import classes:
//import app.hwatch.client.*
//import app.hwatch.client.infrastructure.*
//import app.hwatch.client.models.*

val apiClient = ApiClient()
val webService = apiClient.createWebservice(AuthApi::class.java)

val result : kotlin.collections.Map<kotlin.String, kotlin.Any> = webService.listUsers()
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




重置指定用户密码（仅管理员）。

### Example
```kotlin
// Import classes:
//import app.hwatch.client.*
//import app.hwatch.client.infrastructure.*
//import app.hwatch.client.models.*

val apiClient = ApiClient()
val webService = apiClient.createWebservice(AuthApi::class.java)
val id : kotlin.Int = 56 // kotlin.Int | 
val xCSRFToken : kotlin.String = xCSRFToken_example // kotlin.String | 
val setUserPasswordRequest : SetUserPasswordRequest =  // SetUserPasswordRequest | 

val result : kotlin.collections.Map<kotlin.String, kotlin.Any> = webService.setUserPassword(id, xCSRFToken, setUserPasswordRequest)
```

### Parameters
| **id** | **kotlin.Int**|  | |
| **xCSRFToken** | **kotlin.String**|  | |
| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **setUserPasswordRequest** | [**SetUserPasswordRequest**](SetUserPasswordRequest.md)|  | |

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




修改用户角色（仅管理员，不能降级最后一个管理员）。

### Example
```kotlin
// Import classes:
//import app.hwatch.client.*
//import app.hwatch.client.infrastructure.*
//import app.hwatch.client.models.*

val apiClient = ApiClient()
val webService = apiClient.createWebservice(AuthApi::class.java)
val id : kotlin.Int = 56 // kotlin.Int | 
val xCSRFToken : kotlin.String = xCSRFToken_example // kotlin.String | 
val updateUserRoleRequest : UpdateUserRoleRequest =  // UpdateUserRoleRequest | 

val result : kotlin.collections.Map<kotlin.String, kotlin.Any> = webService.updateUserRole(id, xCSRFToken, updateUserRoleRequest)
```

### Parameters
| **id** | **kotlin.Int**|  | |
| **xCSRFToken** | **kotlin.String**|  | |
| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **updateUserRoleRequest** | [**UpdateUserRoleRequest**](UpdateUserRoleRequest.md)|  | |

### Return type

[**kotlin.collections.Map&lt;kotlin.String, kotlin.Any&gt;**](kotlin.Any.md)

### Authorization



### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json




上传当前用户头像（multipart 字段 file，PNG/JPEG/WebP/GIF，≤6MB）。

### Example
```kotlin
// Import classes:
//import app.hwatch.client.*
//import app.hwatch.client.infrastructure.*
//import app.hwatch.client.models.*

val apiClient = ApiClient()
val webService = apiClient.createWebservice(AuthApi::class.java)
val xCSRFToken : kotlin.String = xCSRFToken_example // kotlin.String | 

val result : kotlin.collections.Map<kotlin.String, kotlin.Any> = webService.uploadMyAvatar(xCSRFToken)
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




上传指定用户头像（仅管理员）。

### Example
```kotlin
// Import classes:
//import app.hwatch.client.*
//import app.hwatch.client.infrastructure.*
//import app.hwatch.client.models.*

val apiClient = ApiClient()
val webService = apiClient.createWebservice(AuthApi::class.java)
val id : kotlin.Int = 56 // kotlin.Int | 
val xCSRFToken : kotlin.String = xCSRFToken_example // kotlin.String | 

val result : kotlin.collections.Map<kotlin.String, kotlin.Any> = webService.uploadUserAvatar(id, xCSRFToken)
```

### Parameters
| **id** | **kotlin.Int**|  | |
| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **xCSRFToken** | **kotlin.String**|  | |

### Return type

[**kotlin.collections.Map&lt;kotlin.String, kotlin.Any&gt;**](kotlin.Any.md)

### Authorization



### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json




用户头像图片（无自定义头像时返回首字母占位图）。

### Example
```kotlin
// Import classes:
//import app.hwatch.client.*
//import app.hwatch.client.infrastructure.*
//import app.hwatch.client.models.*

val apiClient = ApiClient()
val webService = apiClient.createWebservice(AuthApi::class.java)
val id : kotlin.Int = 56 // kotlin.Int | 

val result : kotlin.collections.Map<kotlin.String, kotlin.Any> = webService.userAvatar(id)
```

### Parameters
| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **id** | **kotlin.Int**|  | |

### Return type

[**kotlin.collections.Map&lt;kotlin.String, kotlin.Any&gt;**](kotlin.Any.md)

### Authorization



### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

