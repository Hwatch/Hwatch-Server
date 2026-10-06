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

}
