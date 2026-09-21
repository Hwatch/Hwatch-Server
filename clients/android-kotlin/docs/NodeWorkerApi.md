# NodeWorkerApi

All URIs are relative to *http://localhost*

| Method | HTTP request | Description |
| ------------- | ------------- | ------------- |
| [**claimComputeJob**](NodeWorkerApi.md#claimComputeJob) | **POST** api/node/v1/jobs/claim |  |
| [**completeComputeJob**](NodeWorkerApi.md#completeComputeJob) | **POST** api/node/v1/jobs/{id}/complete |  |
| [**failComputeJob**](NodeWorkerApi.md#failComputeJob) | **POST** api/node/v1/jobs/{id}/fail |  |
| [**nodeHeartbeat**](NodeWorkerApi.md#nodeHeartbeat) | **POST** api/node/v1/heartbeat |  |
| [**updateComputeJobProgress**](NodeWorkerApi.md#updateComputeJobProgress) | **POST** api/node/v1/jobs/{id}/progress |  |





### Example
```kotlin
// Import classes:
//import app.hwatch.client.*
//import app.hwatch.client.infrastructure.*
//import app.hwatch.client.models.*

val apiClient = ApiClient()
apiClient.setBearerToken("TOKEN")
val webService = apiClient.createWebservice(NodeWorkerApi::class.java)
val claimComputeJobRequest : ClaimComputeJobRequest =  // ClaimComputeJobRequest | 

val result : kotlin.collections.Map<kotlin.String, kotlin.Any> = webService.claimComputeJob(claimComputeJobRequest)
```

### Parameters
| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **claimComputeJobRequest** | [**ClaimComputeJobRequest**](ClaimComputeJobRequest.md)|  | |

### Return type

[**kotlin.collections.Map&lt;kotlin.String, kotlin.Any&gt;**](kotlin.Any.md)

### Authorization


Configure nodeBearer:
    ApiClient().setBearerToken("TOKEN")

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
apiClient.setBearerToken("TOKEN")
val webService = apiClient.createWebservice(NodeWorkerApi::class.java)
val id : kotlin.String = id_example // kotlin.String | 
val completeComputeJobRequest : CompleteComputeJobRequest =  // CompleteComputeJobRequest | 

val result : kotlin.collections.Map<kotlin.String, kotlin.Any> = webService.completeComputeJob(id, completeComputeJobRequest)
```

### Parameters
| **id** | **kotlin.String**|  | |
| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **completeComputeJobRequest** | [**CompleteComputeJobRequest**](CompleteComputeJobRequest.md)|  | |

### Return type

[**kotlin.collections.Map&lt;kotlin.String, kotlin.Any&gt;**](kotlin.Any.md)

### Authorization


Configure nodeBearer:
    ApiClient().setBearerToken("TOKEN")

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
apiClient.setBearerToken("TOKEN")
val webService = apiClient.createWebservice(NodeWorkerApi::class.java)
val id : kotlin.String = id_example // kotlin.String | 
val nodeFailure : NodeFailure =  // NodeFailure | 

val result : kotlin.collections.Map<kotlin.String, kotlin.Any> = webService.failComputeJob(id, nodeFailure)
```

### Parameters
| **id** | **kotlin.String**|  | |
| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **nodeFailure** | [**NodeFailure**](NodeFailure.md)|  | |

### Return type

[**kotlin.collections.Map&lt;kotlin.String, kotlin.Any&gt;**](kotlin.Any.md)

### Authorization


Configure nodeBearer:
    ApiClient().setBearerToken("TOKEN")

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
apiClient.setBearerToken("TOKEN")
val webService = apiClient.createWebservice(NodeWorkerApi::class.java)
val nodeHeartbeat : NodeHeartbeat =  // NodeHeartbeat | 

val result : kotlin.collections.Map<kotlin.String, kotlin.Any> = webService.nodeHeartbeat(nodeHeartbeat)
```

### Parameters
| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **nodeHeartbeat** | [**NodeHeartbeat**](NodeHeartbeat.md)|  | |

### Return type

[**kotlin.collections.Map&lt;kotlin.String, kotlin.Any&gt;**](kotlin.Any.md)

### Authorization


Configure nodeBearer:
    ApiClient().setBearerToken("TOKEN")

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
apiClient.setBearerToken("TOKEN")
val webService = apiClient.createWebservice(NodeWorkerApi::class.java)
val id : kotlin.String = id_example // kotlin.String | 
val nodeProgress : NodeProgress =  // NodeProgress | 

val result : kotlin.collections.Map<kotlin.String, kotlin.Any> = webService.updateComputeJobProgress(id, nodeProgress)
```

### Parameters
| **id** | **kotlin.String**|  | |
| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **nodeProgress** | [**NodeProgress**](NodeProgress.md)|  | |

### Return type

[**kotlin.collections.Map&lt;kotlin.String, kotlin.Any&gt;**](kotlin.Any.md)

### Authorization


Configure nodeBearer:
    ApiClient().setBearerToken("TOKEN")

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

