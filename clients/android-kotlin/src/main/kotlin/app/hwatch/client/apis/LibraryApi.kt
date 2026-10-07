package app.hwatch.client.apis

import app.hwatch.client.infrastructure.CollectionFormats.*
import retrofit2.http.*
import retrofit2.Call
import okhttp3.RequestBody
import com.google.gson.annotations.SerializedName

import app.hwatch.client.models.DeleteVideoRequest
import app.hwatch.client.models.Error
import app.hwatch.client.models.ReviewDuplicateCandidatesBatchRequest
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

}
