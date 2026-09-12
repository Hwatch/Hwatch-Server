package app.hwatch.client.apis

import app.hwatch.client.infrastructure.CollectionFormats.*
import retrofit2.http.*
import retrofit2.Call
import okhttp3.RequestBody
import com.google.gson.annotations.SerializedName

import app.hwatch.client.models.Error
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
     * 
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
