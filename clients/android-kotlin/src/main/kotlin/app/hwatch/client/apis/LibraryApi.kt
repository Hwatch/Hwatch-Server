package app.hwatch.client.apis

import app.hwatch.client.infrastructure.CollectionFormats.*
import retrofit2.http.*
import retrofit2.Call
import okhttp3.RequestBody
import com.google.gson.annotations.SerializedName

import app.hwatch.client.models.DeleteVideoRequest
import app.hwatch.client.models.Error
import app.hwatch.client.models.MediaIntegrityApplyRequest
import app.hwatch.client.models.ReviewDuplicateCandidatesBatchRequest
import app.hwatch.client.models.TransRunnerDeletionPlanPreviewRequest
import app.hwatch.client.models.VdfGUIExecuteRequest
import app.hwatch.client.models.VideoPage

interface LibraryApi {
    /**
     * GET api/dashboard
     * 
     * 
     * Responses:
     *  - 200: 成功
     *
     * @param language  (optional, default to "zh-CN")
     * @return [Call]<[kotlin.collections.Map<kotlin.String, kotlin.Any>]>
     */
    @GET("api/dashboard")
    fun dashboard(@Query("language") language: kotlin.String? = "zh-CN"): Call<kotlin.collections.Map<kotlin.String, kotlin.Any>>

    /**
     * POST api/videos/{id}/delete
     * 
     * 删除单个视频的媒体文件。媒体路径始终由服务端根据 video_id 从数据库解析并校验位于媒体根目录内；绝不接受客户端路径。
     * Responses:
     *  - 200: 成功
     *  - 409: 请求失败
     *  - 422: 请求失败
     *  - 503: 请求失败
     *
     * @param id 
     * @param xCSRFToken 
     * @param deleteVideoRequest 
     * @return [Call]<[kotlin.collections.Map<kotlin.String, kotlin.Any>]>
     */
    @POST("api/videos/{id}/delete")
    fun deleteVideo(@Path("id") id: kotlin.String, @Header("X-CSRF-Token") xCSRFToken: kotlin.String, @Body deleteVideoRequest: DeleteVideoRequest): Call<kotlin.collections.Map<kotlin.String, kotlin.Any>>

    /**
     * GET api/media-integrity/plan/{id}
     * 
     * 
     * Responses:
     *  - 200: 成功
     *  - 404: 请求失败
     *
     * @param id 
     * @return [Call]<[kotlin.collections.Map<kotlin.String, kotlin.Any>]>
     */
    @GET("api/media-integrity/plan/{id}")
    fun getDeletionPlan(@Path("id") id: kotlin.String): Call<kotlin.collections.Map<kotlin.String, kotlin.Any>>

    /**
     * POST api/import/html
     * 
     * 
     * Responses:
     *  - 200: 成功
     *
     * @param xCSRFToken 
     * @param requestBody 
     * @return [Call]<[kotlin.collections.Map<kotlin.String, kotlin.Any>]>
     */
    @POST("api/import/html")
    fun importHTML(@Header("X-CSRF-Token") xCSRFToken: kotlin.String, @Body requestBody: kotlin.collections.Map<kotlin.String, kotlin.Any>): Call<kotlin.collections.Map<kotlin.String, kotlin.Any>>


    /**
    * enum for parameter scope
    */
    enum class ScopeLibraryRandom(val value: kotlin.String) {
        @SerializedName(value = "all") all("all"),
        @SerializedName(value = "original") original("original"),
        @SerializedName(value = "translated") translated("translated")
    }


    /**
    * enum for parameter tagMode
    */
    enum class TagModeLibraryRandom(val value: kotlin.String) {
        @SerializedName(value = "all") all("all"),
        @SerializedName(value = "any") any("any")
    }

    /**
     * GET api/library/random
     * 
     * 从整个已下载影视库中均匀随机抽取一小批正式媒体。只读，不触发下载或状态变更；只返回 status&#x3D;completed 且 file_path 非空的数据库记录（不代表物理存在）。
     * Responses:
     *  - 200: 成功
     *  - 400: 请求失败
     *
     * @param limit  (optional, default to 6)
     * @param exclude  (optional)
     * @param q  (optional)
     * @param scope  (optional)
     * @param tags  (optional)
     * @param excludeTags  (optional)
     * @param tagMode  (optional)
     * @param creator  (optional)
     * @param codec  (optional)
     * @param minHeight  (optional)
     * @param excludeWatchedDays 大于 0 时排除最近 N 个自然日内（按设置时区）观看过的视频；0 表示不过滤 (optional, default to 0)
     * @param language  (optional, default to "zh-CN")
     * @return [Call]<[kotlin.collections.Map<kotlin.String, kotlin.Any>]>
     */
    @GET("api/library/random")
    fun libraryRandom(@Query("limit") limit: kotlin.Int? = 6, @Query("exclude") exclude: kotlin.String? = null, @Query("q") q: kotlin.String? = null, @Query("scope") scope: ScopeLibraryRandom? = null, @Query("tags") tags: kotlin.String? = null, @Query("exclude_tags") excludeTags: kotlin.String? = null, @Query("tag_mode") tagMode: TagModeLibraryRandom? = null, @Query("creator") creator: kotlin.String? = null, @Query("codec") codec: kotlin.String? = null, @Query("min_height") minHeight: kotlin.Int? = null, @Query("exclude_watched_days") excludeWatchedDays: kotlin.Int? = 0, @Query("language") language: kotlin.String? = "zh-CN"): Call<kotlin.collections.Map<kotlin.String, kotlin.Any>>

    /**
     * GET api/media-integrity/plans
     * 
     * 
     * Responses:
     *  - 200: 成功
     *
     * @param limit  (optional, default to 48)
     * @param offset  (optional, default to 0)
     * @return [Call]<[kotlin.collections.Map<kotlin.String, kotlin.Any>]>
     */
    @GET("api/media-integrity/plans")
    fun listDeletionPlans(@Query("limit") limit: kotlin.Int? = 48, @Query("offset") offset: kotlin.Int? = 0): Call<kotlin.collections.Map<kotlin.String, kotlin.Any>>

    /**
     * GET api/duplicates
     * 
     * 
     * Responses:
     *  - 200: 成功
     *
     * @return [Call]<[kotlin.collections.Map<kotlin.String, kotlin.Any>]>
     */
    @GET("api/duplicates")
    fun listDuplicateCandidates(): Call<kotlin.collections.Map<kotlin.String, kotlin.Any>>

    /**
     * GET api/media-exclusions
     * 
     * 
     * Responses:
     *  - 200: 成功
     *
     * @param limit  (optional, default to 48)
     * @param offset  (optional, default to 0)
     * @param q  (optional)
     * @param source  (optional)
     * @return [Call]<[kotlin.collections.Map<kotlin.String, kotlin.Any>]>
     */
    @GET("api/media-exclusions")
    fun listMediaExclusions(@Query("limit") limit: kotlin.Int? = 48, @Query("offset") offset: kotlin.Int? = 0, @Query("q") q: kotlin.String? = null, @Query("source") source: kotlin.String? = null): Call<kotlin.collections.Map<kotlin.String, kotlin.Any>>


    /**
    * enum for parameter scope
    */
    enum class ScopeListVideos(val value: kotlin.String) {
        @SerializedName(value = "all") all("all"),
        @SerializedName(value = "original") original("original"),
        @SerializedName(value = "translated") translated("translated")
    }


    /**
    * enum for parameter tagMode
    */
    enum class TagModeListVideos(val value: kotlin.String) {
        @SerializedName(value = "all") all("all"),
        @SerializedName(value = "any") any("any")
    }


    /**
    * enum for parameter order
    */
    enum class OrderListVideos(val value: kotlin.String) {
        @SerializedName(value = "asc") asc("asc"),
        @SerializedName(value = "desc") desc("desc")
    }

    /**
     * GET api/videos
     * 
     * 媒体分页列表。管理员会话可用；持有 read_media scope 的 Service Token（integrationBearer）也可只读调用以枚举媒体，此时响应中的 file_path/plex_path 会被清空。
     * Responses:
     *  - 200: 视频分页
     *  - 400: 请求失败
     *
     * @param limit  (optional, default to 48)
     * @param offset  (optional, default to 0)
     * @param status  (optional)
     * @param q  (optional)
     * @param scope  (optional)
     * @param tags  (optional)
     * @param excludeTags  (optional)
     * @param tagMode  (optional)
     * @param creator  (optional)
     * @param codec  (optional)
     * @param minHeight  (optional)
     * @param sort  (optional)
     * @param order  (optional)
     * @param language  (optional, default to "zh-CN")
     * @return [Call]<[VideoPage]>
     */
    @GET("api/videos")
    fun listVideos(@Query("limit") limit: kotlin.Int? = 48, @Query("offset") offset: kotlin.Int? = 0, @Query("status") status: kotlin.String? = null, @Query("q") q: kotlin.String? = null, @Query("scope") scope: ScopeListVideos? = null, @Query("tags") tags: kotlin.String? = null, @Query("exclude_tags") excludeTags: kotlin.String? = null, @Query("tag_mode") tagMode: TagModeListVideos? = null, @Query("creator") creator: kotlin.String? = null, @Query("codec") codec: kotlin.String? = null, @Query("min_height") minHeight: kotlin.Int? = null, @Query("sort") sort: kotlin.String? = null, @Query("order") order: OrderListVideos? = null, @Query("language") language: kotlin.String? = "zh-CN"): Call<VideoPage>

    /**
     * POST api/media-integrity/apply
     * 
     * 对已确认缺失且由删除清单匹配的媒体执行“移除记录并加入已排除”；confirm&#x3D;false 为 Dry Run。每个视频独立事务，失败不影响其它条目。
     * Responses:
     *  - 200: 成功
     *  - 422: 请求失败
     *  - 503: 请求失败
     *
     * @param xCSRFToken 
     * @param mediaIntegrityApplyRequest 
     * @return [Call]<[kotlin.collections.Map<kotlin.String, kotlin.Any>]>
     */
    @POST("api/media-integrity/apply")
    fun mediaIntegrityApply(@Header("X-CSRF-Token") xCSRFToken: kotlin.String, @Body mediaIntegrityApplyRequest: MediaIntegrityApplyRequest): Call<kotlin.collections.Map<kotlin.String, kotlin.Any>>

    /**
     * POST api/media-integrity/import
     * 
     * 自动识别删除清单格式并返回校验预览，不修改数据库。支持：外部已删除清单（数组或 {deletions:[...]}）、VDF GUI 原生导出（{CreatedAt,EstimatedTotalSavingsBytes,Groups:[{GroupId,RemoveItems,KeepItems}]}）。TransRunner 删除计划请使用模式二 /plan/_*。
     * Responses:
     *  - 200: 成功
     *  - 422: 请求失败
     *
     * @param xCSRFToken 
     * @param requestBody 
     * @return [Call]<[kotlin.collections.Map<kotlin.String, kotlin.Any>]>
     */
    @POST("api/media-integrity/import")
    fun mediaIntegrityImport(@Header("X-CSRF-Token") xCSRFToken: kotlin.String, @Body requestBody: kotlin.collections.Map<kotlin.String, kotlin.Any>): Call<kotlin.collections.Map<kotlin.String, kotlin.Any>>

    /**
     * GET api/media-integrity/scan
     * 
     * 只读扫描已完成媒体的文件完整性；存储不可用时不会判定缺失。
     * Responses:
     *  - 200: 成功
     *
     * @param xCSRFToken 
     * @param limit  (optional, default to 100000)
     * @return [Call]<[kotlin.collections.Map<kotlin.String, kotlin.Any>]>
     */
    @GET("api/media-integrity/scan")
    fun mediaIntegrityScan(@Header("X-CSRF-Token") xCSRFToken: kotlin.String, @Query("limit") limit: kotlin.Int? = 100000): Call<kotlin.collections.Map<kotlin.String, kotlin.Any>>

    /**
     * POST api/media-integrity/plan/{id}/recover
     * 
     * 
     * Responses:
     *  - 200: 成功
     *  - 503: 请求失败
     *
     * @param id 
     * @param xCSRFToken 
     * @return [Call]<[kotlin.collections.Map<kotlin.String, kotlin.Any>]>
     */
    @POST("api/media-integrity/plan/{id}/recover")
    fun recoverDeletionPlan(@Path("id") id: kotlin.String, @Header("X-CSRF-Token") xCSRFToken: kotlin.String): Call<kotlin.collections.Map<kotlin.String, kotlin.Any>>

    /**
     * POST api/videos/{id}/refresh-thumbnail
     * 
     * 重新抓取指定视频当前来源页面的封面，仅更新 thumbnail_url；不修改媒体文件、路径、编码、哈希、生命周期状态、排除或下载器状态。需要普通管理会话（CSRF），TransRunner Service Token 无权调用。返回 status（updated/unchanged/missing/failed）及 before/after。
     * Responses:
     *  - 200: 成功
     *  - 404: 请求失败
     *
     * @param id 
     * @param xCSRFToken 
     * @return [Call]<[kotlin.collections.Map<kotlin.String, kotlin.Any>]>
     */
    @POST("api/videos/{id}/refresh-thumbnail")
    fun refreshVideoThumbnail(@Path("id") id: kotlin.String, @Header("X-CSRF-Token") xCSRFToken: kotlin.String): Call<kotlin.collections.Map<kotlin.String, kotlin.Any>>

    /**
     * DELETE api/media-exclusions/{video_id}
     * 
     * 删除指定 video_id 的排除 tombstone（恢复允许下载）；不会自动创建下载任务。
     * Responses:
     *  - 200: 成功
     *
     * @param videoId 
     * @param xCSRFToken 
     * @return [Call]<[kotlin.collections.Map<kotlin.String, kotlin.Any>]>
     */
    @DELETE("api/media-exclusions/{video_id}")
    fun removeMediaExclusion(@Path("video_id") videoId: kotlin.String, @Header("X-CSRF-Token") xCSRFToken: kotlin.String): Call<kotlin.collections.Map<kotlin.String, kotlin.Any>>


    /**
    * enum for parameter action
    */
    enum class ActionReviewDuplicateCandidate(val value: kotlin.String) {
        @SerializedName(value = "confirm") confirm("confirm"),
        @SerializedName(value = "dismiss") dismiss("dismiss")
    }

    /**
     * POST api/duplicates/{id}/{action}
     * 
     * 
     * Responses:
     *  - 200: 成功
     *  - 400: 请求失败
     *
     * @param id 
     * @param action 
     * @param xCSRFToken 
     * @return [Call]<[kotlin.collections.Map<kotlin.String, kotlin.Any>]>
     */
    @POST("api/duplicates/{id}/{action}")
    fun reviewDuplicateCandidate(@Path("id") id: kotlin.Long, @Path("action") action: kotlin.String, @Header("X-CSRF-Token") xCSRFToken: kotlin.String): Call<kotlin.collections.Map<kotlin.String, kotlin.Any>>

    /**
     * POST api/duplicates/batch
     * 
     * 
     * Responses:
     *  - 200: 成功
     *  - 400: 请求失败
     *
     * @param xCSRFToken 
     * @param reviewDuplicateCandidatesBatchRequest 
     * @return [Call]<[kotlin.collections.Map<kotlin.String, kotlin.Any>]>
     */
    @POST("api/duplicates/batch")
    fun reviewDuplicateCandidatesBatch(@Header("X-CSRF-Token") xCSRFToken: kotlin.String, @Body reviewDuplicateCandidatesBatchRequest: ReviewDuplicateCandidatesBatchRequest): Call<kotlin.collections.Map<kotlin.String, kotlin.Any>>

    /**
     * GET media/{id}
     * 
     * 
     * Responses:
     *  - 200: 支持 Range 的媒体流
     *
     * @param id 
     * @return [Call]<[Unit]>
     */
    @GET("media/{id}")
    fun streamMedia(@Path("id") id: kotlin.String): Call<Unit>

    /**
     * POST api/media-integrity/plan/execute
     * 
     * confirm&#x3D;false 为 Dry Run；confirm&#x3D;true 时对已验证条目把文件移入隔离区（同文件系统）并事务性写入排除，执行状态持久化，可重复执行与恢复。
     * Responses:
     *  - 200: 成功
     *  - 422: 请求失败
     *  - 503: 请求失败
     *
     * @param xCSRFToken 
     * @param vdfGUIExecuteRequest 
     * @return [Call]<[kotlin.collections.Map<kotlin.String, kotlin.Any>]>
     */
    @POST("api/media-integrity/plan/execute")
    fun transRunnerDeletionPlanExecute(@Header("X-CSRF-Token") xCSRFToken: kotlin.String, @Body vdfGUIExecuteRequest: VdfGUIExecuteRequest): Call<kotlin.collections.Map<kotlin.String, kotlin.Any>>

    /**
     * POST api/media-integrity/plan/preview
     * 
     * 解析并服务端重新校验 TransRunner 删除计划（kind&#x3D;transrunner-duplicate-deletion-plan, plan_version&#x3D;1），返回逐条校验结果；只读，不写库、不动文件。
     * Responses:
     *  - 200: 成功
     *  - 422: 请求失败
     *
     * @param xCSRFToken 
     * @param transRunnerDeletionPlanPreviewRequest 
     * @return [Call]<[kotlin.collections.Map<kotlin.String, kotlin.Any>]>
     */
    @POST("api/media-integrity/plan/preview")
    fun transRunnerDeletionPlanPreview(@Header("X-CSRF-Token") xCSRFToken: kotlin.String, @Body transRunnerDeletionPlanPreviewRequest: TransRunnerDeletionPlanPreviewRequest): Call<kotlin.collections.Map<kotlin.String, kotlin.Any>>

    /**
     * POST api/media-integrity/vdf/execute
     * 
     * 执行 VDF GUI 原生删除导出：按路径匹配 HWatch 媒体（服务端路径→两段后缀→唯一 basename），校验同订阅/保留项存在；删除文件仍在则移入隔离区，已被外部删除则仅写入排除（复用 RemoveAndExclude）。confirm&#x3D;false 为 Dry Run；歧义或非受管路径拒绝执行。
     * Responses:
     *  - 200: 成功
     *  - 422: 请求失败
     *  - 503: 请求失败
     *
     * @param xCSRFToken 
     * @param vdfGUIExecuteRequest 
     * @return [Call]<[kotlin.collections.Map<kotlin.String, kotlin.Any>]>
     */
    @POST("api/media-integrity/vdf/execute")
    fun vdfGUIExecute(@Header("X-CSRF-Token") xCSRFToken: kotlin.String, @Body vdfGUIExecuteRequest: VdfGUIExecuteRequest): Call<kotlin.collections.Map<kotlin.String, kotlin.Any>>

    /**
     * GET api/videos/facets
     * 
     * 
     * Responses:
     *  - 200: 成功
     *
     * @return [Call]<[kotlin.collections.Map<kotlin.String, kotlin.Any>]>
     */
    @GET("api/videos/facets")
    fun videoFacets(): Call<kotlin.collections.Map<kotlin.String, kotlin.Any>>

    /**
     * GET thumb/{id}
     * 
     * 
     * Responses:
     *  - 200: 图片或 SVG 占位图
     *
     * @param id 
     * @return [Call]<[Unit]>
     */
    @GET("thumb/{id}")
    fun videoThumbnail(@Path("id") id: kotlin.String): Call<Unit>

    /**
     * GET api/timeline
     * 
     * 观看时间线（只读）：按时间倒序返回观看事件，含译文标题等媒体字段与 watched_at。播放 /media/{id} 时写入；同一视频 10 分钟内去重，最多保留 5000 条。
     * Responses:
     *  - 200: 成功
     *
     * @param language  (optional, default to "zh-CN")
     * @param limit  (optional, default to 200)
     * @param offset  (optional)
     * @return [Call]<[kotlin.collections.Map<kotlin.String, kotlin.Any>]>
     */
    @GET("api/timeline")
    fun watchTimeline(@Query("language") language: kotlin.String? = "zh-CN", @Query("limit") limit: kotlin.Int? = 200, @Query("offset") offset: kotlin.Int? = null): Call<kotlin.collections.Map<kotlin.String, kotlin.Any>>

}
