# LibraryApi

All URIs are relative to *http://localhost*

| Method | HTTP request | Description |
| ------------- | ------------- | ------------- |
| [**dashboard**](LibraryApi.md#dashboard) | **GET** api/dashboard |  |
| [**deleteVideo**](LibraryApi.md#deleteVideo) | **POST** api/videos/{id}/delete |  |
| [**getDeletionPlan**](LibraryApi.md#getDeletionPlan) | **GET** api/media-integrity/plan/{id} |  |
| [**importHTML**](LibraryApi.md#importHTML) | **POST** api/import/html |  |
| [**libraryRandom**](LibraryApi.md#libraryRandom) | **GET** api/library/random |  |
| [**listDeletionPlans**](LibraryApi.md#listDeletionPlans) | **GET** api/media-integrity/plans |  |
| [**listDuplicateCandidates**](LibraryApi.md#listDuplicateCandidates) | **GET** api/duplicates |  |
| [**listMediaExclusions**](LibraryApi.md#listMediaExclusions) | **GET** api/media-exclusions |  |
| [**listVideos**](LibraryApi.md#listVideos) | **GET** api/videos |  |
| [**mediaIntegrityApply**](LibraryApi.md#mediaIntegrityApply) | **POST** api/media-integrity/apply |  |
| [**mediaIntegrityImport**](LibraryApi.md#mediaIntegrityImport) | **POST** api/media-integrity/import |  |
| [**mediaIntegrityScan**](LibraryApi.md#mediaIntegrityScan) | **GET** api/media-integrity/scan |  |
| [**recoverDeletionPlan**](LibraryApi.md#recoverDeletionPlan) | **POST** api/media-integrity/plan/{id}/recover |  |
| [**refreshVideoThumbnail**](LibraryApi.md#refreshVideoThumbnail) | **POST** api/videos/{id}/refresh-thumbnail |  |
| [**removeMediaExclusion**](LibraryApi.md#removeMediaExclusion) | **DELETE** api/media-exclusions/{video_id} |  |
| [**reviewDuplicateCandidate**](LibraryApi.md#reviewDuplicateCandidate) | **POST** api/duplicates/{id}/{action} |  |
| [**reviewDuplicateCandidatesBatch**](LibraryApi.md#reviewDuplicateCandidatesBatch) | **POST** api/duplicates/batch |  |
| [**streamMedia**](LibraryApi.md#streamMedia) | **GET** media/{id} |  |
| [**transRunnerDeletionPlanExecute**](LibraryApi.md#transRunnerDeletionPlanExecute) | **POST** api/media-integrity/plan/execute |  |
| [**transRunnerDeletionPlanPreview**](LibraryApi.md#transRunnerDeletionPlanPreview) | **POST** api/media-integrity/plan/preview |  |
| [**vdfGUIExecute**](LibraryApi.md#vdfGUIExecute) | **POST** api/media-integrity/vdf/execute |  |
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




删除单个视频的媒体文件。媒体路径始终由服务端根据 video_id 从数据库解析并校验位于媒体根目录内；绝不接受客户端路径。

### Example
```kotlin
// Import classes:
//import app.hwatch.client.*
//import app.hwatch.client.infrastructure.*
//import app.hwatch.client.models.*

val apiClient = ApiClient()
val webService = apiClient.createWebservice(LibraryApi::class.java)
val id : kotlin.String = id_example // kotlin.String | 
val xCSRFToken : kotlin.String = xCSRFToken_example // kotlin.String | 
val deleteVideoRequest : DeleteVideoRequest =  // DeleteVideoRequest | 

val result : kotlin.collections.Map<kotlin.String, kotlin.Any> = webService.deleteVideo(id, xCSRFToken, deleteVideoRequest)
```

### Parameters
| **id** | **kotlin.String**|  | |
| **xCSRFToken** | **kotlin.String**|  | |
| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **deleteVideoRequest** | [**DeleteVideoRequest**](DeleteVideoRequest.md)|  | |

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

val result : kotlin.collections.Map<kotlin.String, kotlin.Any> = webService.getDeletionPlan(id)
```

### Parameters
| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **id** | **kotlin.String**|  | |

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




从整个已下载影视库中均匀随机抽取一小批正式媒体。只读，不触发下载或状态变更；只返回 status&#x3D;completed 且 file_path 非空的数据库记录（不代表物理存在）。

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
val exclude : kotlin.String = exclude_example // kotlin.String | 
val q : kotlin.String = q_example // kotlin.String | 
val scope : kotlin.String = scope_example // kotlin.String | 
val tags : kotlin.String = tags_example // kotlin.String | 
val excludeTags : kotlin.String = excludeTags_example // kotlin.String | 
val tagMode : kotlin.String = tagMode_example // kotlin.String | 
val creator : kotlin.String = creator_example // kotlin.String | 
val codec : kotlin.String = codec_example // kotlin.String | 
val minHeight : kotlin.Int = 56 // kotlin.Int | 
val language : kotlin.String = language_example // kotlin.String | 

val result : kotlin.collections.Map<kotlin.String, kotlin.Any> = webService.libraryRandom(limit, exclude, q, scope, tags, excludeTags, tagMode, creator, codec, minHeight, language)
```

### Parameters
| **limit** | **kotlin.Int**|  | [optional] [default to 6] |
| **exclude** | **kotlin.String**|  | [optional] |
| **q** | **kotlin.String**|  | [optional] |
| **scope** | **kotlin.String**|  | [optional] [enum: all, original, translated] |
| **tags** | **kotlin.String**|  | [optional] |
| **excludeTags** | **kotlin.String**|  | [optional] |
| **tagMode** | **kotlin.String**|  | [optional] [enum: all, any] |
| **creator** | **kotlin.String**|  | [optional] |
| **codec** | **kotlin.String**|  | [optional] |
| **minHeight** | **kotlin.Int**|  | [optional] |
| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **language** | **kotlin.String**|  | [optional] [default to &quot;zh-CN&quot;] |

### Return type

[**kotlin.collections.Map&lt;kotlin.String, kotlin.Any&gt;**](kotlin.Any.md)

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
val limit : kotlin.Int = 56 // kotlin.Int | 
val offset : kotlin.Int = 56 // kotlin.Int | 

val result : kotlin.collections.Map<kotlin.String, kotlin.Any> = webService.listDeletionPlans(limit, offset)
```

### Parameters
| **limit** | **kotlin.Int**|  | [optional] [default to 48] |
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




### Example
```kotlin
// Import classes:
//import app.hwatch.client.*
//import app.hwatch.client.infrastructure.*
//import app.hwatch.client.models.*

val apiClient = ApiClient()
val webService = apiClient.createWebservice(LibraryApi::class.java)
val limit : kotlin.Int = 56 // kotlin.Int | 
val offset : kotlin.Int = 56 // kotlin.Int | 
val q : kotlin.String = q_example // kotlin.String | 
val source : kotlin.String = source_example // kotlin.String | 

val result : kotlin.collections.Map<kotlin.String, kotlin.Any> = webService.listMediaExclusions(limit, offset, q, source)
```

### Parameters
| **limit** | **kotlin.Int**|  | [optional] [default to 48] |
| **offset** | **kotlin.Int**|  | [optional] [default to 0] |
| **q** | **kotlin.String**|  | [optional] |
| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **source** | **kotlin.String**|  | [optional] |

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




对已确认缺失且由删除清单匹配的媒体执行“移除记录并加入已排除”；confirm&#x3D;false 为 Dry Run。每个视频独立事务，失败不影响其它条目。

### Example
```kotlin
// Import classes:
//import app.hwatch.client.*
//import app.hwatch.client.infrastructure.*
//import app.hwatch.client.models.*

val apiClient = ApiClient()
val webService = apiClient.createWebservice(LibraryApi::class.java)
val xCSRFToken : kotlin.String = xCSRFToken_example // kotlin.String | 
val mediaIntegrityApplyRequest : MediaIntegrityApplyRequest =  // MediaIntegrityApplyRequest | 

val result : kotlin.collections.Map<kotlin.String, kotlin.Any> = webService.mediaIntegrityApply(xCSRFToken, mediaIntegrityApplyRequest)
```

### Parameters
| **xCSRFToken** | **kotlin.String**|  | |
| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **mediaIntegrityApplyRequest** | [**MediaIntegrityApplyRequest**](MediaIntegrityApplyRequest.md)|  | |

### Return type

[**kotlin.collections.Map&lt;kotlin.String, kotlin.Any&gt;**](kotlin.Any.md)

### Authorization



### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json




自动识别删除清单格式并返回校验预览，不修改数据库。支持：外部已删除清单（数组或 {deletions:[...]}）、VDF GUI 原生导出（{CreatedAt,EstimatedTotalSavingsBytes,Groups:[{GroupId,RemoveItems,KeepItems}]}）。TransRunner 删除计划请使用模式二 /plan/_*。

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

val result : kotlin.collections.Map<kotlin.String, kotlin.Any> = webService.mediaIntegrityImport(xCSRFToken, requestBody)
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




只读扫描已完成媒体的文件完整性；存储不可用时不会判定缺失。

### Example
```kotlin
// Import classes:
//import app.hwatch.client.*
//import app.hwatch.client.infrastructure.*
//import app.hwatch.client.models.*

val apiClient = ApiClient()
val webService = apiClient.createWebservice(LibraryApi::class.java)
val xCSRFToken : kotlin.String = xCSRFToken_example // kotlin.String | 
val limit : kotlin.Int = 56 // kotlin.Int | 

val result : kotlin.collections.Map<kotlin.String, kotlin.Any> = webService.mediaIntegrityScan(xCSRFToken, limit)
```

### Parameters
| **xCSRFToken** | **kotlin.String**|  | |
| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **limit** | **kotlin.Int**|  | [optional] [default to 100000] |

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
val xCSRFToken : kotlin.String = xCSRFToken_example // kotlin.String | 

val result : kotlin.collections.Map<kotlin.String, kotlin.Any> = webService.recoverDeletionPlan(id, xCSRFToken)
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




重新抓取指定视频当前来源页面的封面，仅更新 thumbnail_url；不修改媒体文件、路径、编码、哈希、生命周期状态、排除或下载器状态。需要普通管理会话（CSRF），TransRunner Service Token 无权调用。返回 status（updated/unchanged/missing/failed）及 before/after。

### Example
```kotlin
// Import classes:
//import app.hwatch.client.*
//import app.hwatch.client.infrastructure.*
//import app.hwatch.client.models.*

val apiClient = ApiClient()
val webService = apiClient.createWebservice(LibraryApi::class.java)
val id : kotlin.String = id_example // kotlin.String | 
val xCSRFToken : kotlin.String = xCSRFToken_example // kotlin.String | 

val result : kotlin.collections.Map<kotlin.String, kotlin.Any> = webService.refreshVideoThumbnail(id, xCSRFToken)
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




删除指定 video_id 的排除 tombstone（恢复允许下载）；不会自动创建下载任务。

### Example
```kotlin
// Import classes:
//import app.hwatch.client.*
//import app.hwatch.client.infrastructure.*
//import app.hwatch.client.models.*

val apiClient = ApiClient()
val webService = apiClient.createWebservice(LibraryApi::class.java)
val videoId : kotlin.String = videoId_example // kotlin.String | 
val xCSRFToken : kotlin.String = xCSRFToken_example // kotlin.String | 

val result : kotlin.collections.Map<kotlin.String, kotlin.Any> = webService.removeMediaExclusion(videoId, xCSRFToken)
```

### Parameters
| **videoId** | **kotlin.String**|  | |
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




confirm&#x3D;false 为 Dry Run；confirm&#x3D;true 时对已验证条目把文件移入隔离区（同文件系统）并事务性写入排除，执行状态持久化，可重复执行与恢复。

### Example
```kotlin
// Import classes:
//import app.hwatch.client.*
//import app.hwatch.client.infrastructure.*
//import app.hwatch.client.models.*

val apiClient = ApiClient()
val webService = apiClient.createWebservice(LibraryApi::class.java)
val xCSRFToken : kotlin.String = xCSRFToken_example // kotlin.String | 
val vdfGUIExecuteRequest : VdfGUIExecuteRequest =  // VdfGUIExecuteRequest | 

val result : kotlin.collections.Map<kotlin.String, kotlin.Any> = webService.transRunnerDeletionPlanExecute(xCSRFToken, vdfGUIExecuteRequest)
```

### Parameters
| **xCSRFToken** | **kotlin.String**|  | |
| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **vdfGUIExecuteRequest** | [**VdfGUIExecuteRequest**](VdfGUIExecuteRequest.md)|  | |

### Return type

[**kotlin.collections.Map&lt;kotlin.String, kotlin.Any&gt;**](kotlin.Any.md)

### Authorization



### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json




解析并服务端重新校验 TransRunner 删除计划（kind&#x3D;transrunner-duplicate-deletion-plan, plan_version&#x3D;1），返回逐条校验结果；只读，不写库、不动文件。

### Example
```kotlin
// Import classes:
//import app.hwatch.client.*
//import app.hwatch.client.infrastructure.*
//import app.hwatch.client.models.*

val apiClient = ApiClient()
val webService = apiClient.createWebservice(LibraryApi::class.java)
val xCSRFToken : kotlin.String = xCSRFToken_example // kotlin.String | 
val transRunnerDeletionPlanPreviewRequest : TransRunnerDeletionPlanPreviewRequest =  // TransRunnerDeletionPlanPreviewRequest | 

val result : kotlin.collections.Map<kotlin.String, kotlin.Any> = webService.transRunnerDeletionPlanPreview(xCSRFToken, transRunnerDeletionPlanPreviewRequest)
```

### Parameters
| **xCSRFToken** | **kotlin.String**|  | |
| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **transRunnerDeletionPlanPreviewRequest** | [**TransRunnerDeletionPlanPreviewRequest**](TransRunnerDeletionPlanPreviewRequest.md)|  | |

### Return type

[**kotlin.collections.Map&lt;kotlin.String, kotlin.Any&gt;**](kotlin.Any.md)

### Authorization



### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json




执行 VDF GUI 原生删除导出：按路径匹配 HWatch 媒体（服务端路径→两段后缀→唯一 basename），校验同订阅/保留项存在；删除文件仍在则移入隔离区，已被外部删除则仅写入排除（复用 RemoveAndExclude）。confirm&#x3D;false 为 Dry Run；歧义或非受管路径拒绝执行。

### Example
```kotlin
// Import classes:
//import app.hwatch.client.*
//import app.hwatch.client.infrastructure.*
//import app.hwatch.client.models.*

val apiClient = ApiClient()
val webService = apiClient.createWebservice(LibraryApi::class.java)
val xCSRFToken : kotlin.String = xCSRFToken_example // kotlin.String | 
val vdfGUIExecuteRequest : VdfGUIExecuteRequest =  // VdfGUIExecuteRequest | 

val result : kotlin.collections.Map<kotlin.String, kotlin.Any> = webService.vdfGUIExecute(xCSRFToken, vdfGUIExecuteRequest)
```

### Parameters
| **xCSRFToken** | **kotlin.String**|  | |
| Name | Type | Description  | Notes |
| ------------- | ------------- | ------------- | ------------- |
| **vdfGUIExecuteRequest** | [**VdfGUIExecuteRequest**](VdfGUIExecuteRequest.md)|  | |

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

