package app.hwatch.client.apis

import app.hwatch.client.infrastructure.CollectionFormats.*
import retrofit2.http.*
import retrofit2.Call
import okhttp3.RequestBody
import com.google.gson.annotations.SerializedName

import app.hwatch.client.models.CreateSubscriptionRequest
import app.hwatch.client.models.Error
import app.hwatch.client.models.UpdateSubscriptionRequest

interface SubscriptionsApi {
    /**
     * POST api/check
     * 
     * 
     * Responses:
     *  - 202: 成功
     *
     * @param xCSRFToken 
     * @return [Call]<[kotlin.collections.Map<kotlin.String, kotlin.Any>]>
     */
    @POST("api/check")
    fun checkAll(@Header("X-CSRF-Token") xCSRFToken: kotlin.String): Call<kotlin.collections.Map<kotlin.String, kotlin.Any>>

    /**
     * GET api/check/status
     * 
     * 订阅检查进度（queued/active/running），供前端在批量检查完成后自动刷新。检查在服务端有并发上限（默认 3）。
     * Responses:
     *  - 200: 成功
     *
     * @return [Call]<[kotlin.collections.Map<kotlin.String, kotlin.Any>]>
     */
    @GET("api/check/status")
    fun checkStatus(): Call<kotlin.collections.Map<kotlin.String, kotlin.Any>>

    /**
     * POST api/check/{slug}
     * 
     * 
     * Responses:
     *  - 202: 成功
     *
     * @param slug 
     * @param xCSRFToken 
     * @return [Call]<[kotlin.collections.Map<kotlin.String, kotlin.Any>]>
     */
    @POST("api/check/{slug}")
    fun checkSubscription(@Path("slug") slug: kotlin.String, @Header("X-CSRF-Token") xCSRFToken: kotlin.String): Call<kotlin.collections.Map<kotlin.String, kotlin.Any>>

    /**
     * POST api/subscriptions
     * 
     * 
     * Responses:
     *  - 201: 成功
     *  - 400: 请求失败
     *  - 409: 请求失败
     *  - 502: 请求失败
     *
     * @param xCSRFToken 
     * @param createSubscriptionRequest 
     * @return [Call]<[kotlin.collections.Map<kotlin.String, kotlin.Any>]>
     */
    @POST("api/subscriptions")
    fun createSubscription(@Header("X-CSRF-Token") xCSRFToken: kotlin.String, @Body createSubscriptionRequest: CreateSubscriptionRequest): Call<kotlin.collections.Map<kotlin.String, kotlin.Any>>

    /**
     * DELETE api/subscriptions/{slug}
     * 
     * 
     * Responses:
     *  - 200: 成功
     *
     * @param slug 
     * @param xCSRFToken 
     * @return [Call]<[kotlin.collections.Map<kotlin.String, kotlin.Any>]>
     */
    @DELETE("api/subscriptions/{slug}")
    fun deleteSubscription(@Path("slug") slug: kotlin.String, @Header("X-CSRF-Token") xCSRFToken: kotlin.String): Call<kotlin.collections.Map<kotlin.String, kotlin.Any>>

    /**
     * DELETE api/subscriptions/{slug}/avatar
     * 
     * 清除订阅自定义头像，回退到采集头像。
     * Responses:
     *  - 200: 成功
     *
     * @param slug 
     * @param xCSRFToken 
     * @return [Call]<[kotlin.collections.Map<kotlin.String, kotlin.Any>]>
     */
    @DELETE("api/subscriptions/{slug}/avatar")
    fun deleteSubscriptionAvatar(@Path("slug") slug: kotlin.String, @Header("X-CSRF-Token") xCSRFToken: kotlin.String): Call<kotlin.collections.Map<kotlin.String, kotlin.Any>>

    /**
     * GET api/platforms
     * 
     * 
     * Responses:
     *  - 200: 成功
     *
     * @return [Call]<[kotlin.collections.Map<kotlin.String, kotlin.Any>]>
     */
    @GET("api/platforms")
    fun listPlatforms(): Call<kotlin.collections.Map<kotlin.String, kotlin.Any>>


    /**
    * enum for parameter level
    */
    enum class LevelListSubscriptionLogs(val value: kotlin.String) {
        @SerializedName(value = "info") info("info"),
        @SerializedName(value = "error") error("error")
    }

    /**
     * GET api/logs
     * 
     * 订阅检查日志中心（只读）。记录每次检查的结果：发现/新增/入队数量或错误原因。按 id 倒序，最多保留最近 5000 条。
     * Responses:
     *  - 200: 成功
     *
     * @param limit  (optional, default to 200)
     * @param offset  (optional)
     * @param slug  (optional)
     * @param level  (optional)
     * @return [Call]<[kotlin.collections.Map<kotlin.String, kotlin.Any>]>
     */
    @GET("api/logs")
    fun listSubscriptionLogs(@Query("limit") limit: kotlin.Int? = 200, @Query("offset") offset: kotlin.Int? = null, @Query("slug") slug: kotlin.String? = null, @Query("level") level: LevelListSubscriptionLogs? = null): Call<kotlin.collections.Map<kotlin.String, kotlin.Any>>

    /**
     * GET api/subscriptions
     * 
     * 
     * Responses:
     *  - 200: 成功
     *
     * @return [Call]<[kotlin.collections.Map<kotlin.String, kotlin.Any>]>
     */
    @GET("api/subscriptions")
    fun listSubscriptions(): Call<kotlin.collections.Map<kotlin.String, kotlin.Any>>

    /**
     * GET api/creators/search
     * 
     * 
     * Responses:
     *  - 200: 成功
     *
     * @param q 
     * @param platform  (optional)
     * @return [Call]<[kotlin.collections.Map<kotlin.String, kotlin.Any>]>
     */
    @GET("api/creators/search")
    fun searchCreators(@Query("q") q: kotlin.String, @Query("platform") platform: kotlin.String? = null): Call<kotlin.collections.Map<kotlin.String, kotlin.Any>>

    /**
     * GET avatar/{slug}
     * 
     * 
     * Responses:
     *  - 200: 本地缓存的订阅头像或 SVG 占位图
     *  - 404: 请求失败
     *
     * @param slug 
     * @return [Call]<[Unit]>
     */
    @GET("avatar/{slug}")
    fun subscriptionAvatar(@Path("slug") slug: kotlin.String): Call<Unit>

    /**
     * PATCH api/subscriptions/{slug}
     * 
     * 
     * Responses:
     *  - 200: 成功
     *
     * @param slug 
     * @param xCSRFToken 
     * @param updateSubscriptionRequest 
     * @return [Call]<[kotlin.collections.Map<kotlin.String, kotlin.Any>]>
     */
    @PATCH("api/subscriptions/{slug}")
    fun updateSubscription(@Path("slug") slug: kotlin.String, @Header("X-CSRF-Token") xCSRFToken: kotlin.String, @Body updateSubscriptionRequest: UpdateSubscriptionRequest): Call<kotlin.collections.Map<kotlin.String, kotlin.Any>>

    /**
     * POST api/subscriptions/{slug}/avatar
     * 
     * 上传订阅（作者/番剧）自定义头像，覆盖采集到的头像。
     * Responses:
     *  - 200: 成功
     *  - 400: 请求失败
     *
     * @param slug 
     * @param xCSRFToken 
     * @return [Call]<[kotlin.collections.Map<kotlin.String, kotlin.Any>]>
     */
    @POST("api/subscriptions/{slug}/avatar")
    fun uploadSubscriptionAvatar(@Path("slug") slug: kotlin.String, @Header("X-CSRF-Token") xCSRFToken: kotlin.String): Call<kotlin.collections.Map<kotlin.String, kotlin.Any>>

}
