package app.hwatch.client.apis

import app.hwatch.client.infrastructure.CollectionFormats.*
import retrofit2.http.*
import retrofit2.Call
import okhttp3.RequestBody
import com.google.gson.annotations.SerializedName

import app.hwatch.client.models.Error
import app.hwatch.client.models.ToggleDownloadsRequest
import app.hwatch.client.models.VideoPage

interface DownloadsApi {
    /**
     * POST api/videos/download-all
     * 
     * 
     * Responses:
     *  - 202: 成功
     *
     * @param xCSRFToken 
     * @param subscription  (optional)
     * @return [Call]<[kotlin.collections.Map<kotlin.String, kotlin.Any>]>
     */
    @POST("api/videos/download-all")
    fun downloadAll(@Header("X-CSRF-Token") xCSRFToken: kotlin.String, @Query("subscription") subscription: kotlin.String? = null): Call<kotlin.collections.Map<kotlin.String, kotlin.Any>>

    /**
     * GET api/events
     * 
     * text/event-stream。客户端断线后自动重连，并通过 GET /api/dashboard 与 GET /api/tasks 恢复全量状态。
     * Responses:
     *  - 200: SSE 下载状态流
     *
     * @return [Call]<[kotlin.String]>
     */
    @GET("api/events")
    fun downloadEvents(): Call<kotlin.String>

    /**
     * GET api/tasks
     * 
     * 
     * Responses:
     *  - 200: 视频分页
     *
     * @param limit  (optional, default to 200)
     * @param offset  (optional, default to 0)
     * @param status  (optional)
     * @param language  (optional, default to "zh-CN")
     * @return [Call]<[VideoPage]>
     */
    @GET("api/tasks")
    fun listTasks(@Query("limit") limit: kotlin.Int? = 200, @Query("offset") offset: kotlin.Int? = 0, @Query("status") status: kotlin.String? = null, @Query("language") language: kotlin.String? = "zh-CN"): Call<VideoPage>

    /**
     * GET api/tasks/{id}/history
     * 
     * 
     * Responses:
     *  - 200: 成功
     *
     * @param id 
     * @return [Call]<[kotlin.collections.Map<kotlin.String, kotlin.Any>]>
     */
    @GET("api/tasks/{id}/history")
    fun taskHistory(@Path("id") id: kotlin.String): Call<kotlin.collections.Map<kotlin.String, kotlin.Any>>

    /**
     * POST api/downloads/toggle
     * 
     * 
     * Responses:
     *  - 202: 成功
     *
     * @param xCSRFToken 
     * @param toggleDownloadsRequest 
     * @return [Call]<[kotlin.collections.Map<kotlin.String, kotlin.Any>]>
     */
    @POST("api/downloads/toggle")
    fun toggleDownloads(@Header("X-CSRF-Token") xCSRFToken: kotlin.String, @Body toggleDownloadsRequest: ToggleDownloadsRequest): Call<kotlin.collections.Map<kotlin.String, kotlin.Any>>


    /**
    * enum for parameter action
    */
    enum class ActionVideoAction(val value: kotlin.String) {
        @SerializedName(value = "download") download("download"),
        @SerializedName(value = "retry") retry("retry"),
        @SerializedName(value = "pause") pause("pause"),
        @SerializedName(value = "resume") resume("resume"),
        @SerializedName(value = "delete") delete("delete")
    }

    /**
     * POST api/videos/{id}/{action}
     * 
     * 
     * Responses:
     *  - 202: 成功
     *  - 400: 请求失败
     *
     * @param id 
     * @param action 
     * @param xCSRFToken 
     * @return [Call]<[kotlin.collections.Map<kotlin.String, kotlin.Any>]>
     */
    @POST("api/videos/{id}/{action}")
    fun videoAction(@Path("id") id: kotlin.String, @Path("action") action: kotlin.String, @Header("X-CSRF-Token") xCSRFToken: kotlin.String): Call<kotlin.collections.Map<kotlin.String, kotlin.Any>>

}
