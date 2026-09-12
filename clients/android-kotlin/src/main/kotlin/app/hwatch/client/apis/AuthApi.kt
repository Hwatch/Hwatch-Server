package app.hwatch.client.apis

import app.hwatch.client.infrastructure.CollectionFormats.*
import retrofit2.http.*
import retrofit2.Call
import okhttp3.RequestBody
import com.google.gson.annotations.SerializedName

import app.hwatch.client.models.Credentials
import app.hwatch.client.models.Error

interface AuthApi {
    /**
     * GET api/auth/me
     * 
     * 
     * Responses:
     *  - 200: 成功
     *
     * @return [Call]<[kotlin.collections.Map<kotlin.String, kotlin.Any>]>
     */
    @GET("api/auth/me")
    fun authMe(): Call<kotlin.collections.Map<kotlin.String, kotlin.Any>>

    /**
     * GET api/auth/status
     * 
     * 
     * Responses:
     *  - 200: 成功
     *
     * @return [Call]<[kotlin.collections.Map<kotlin.String, kotlin.Any>]>
     */
    @GET("api/auth/status")
    fun authStatus(): Call<kotlin.collections.Map<kotlin.String, kotlin.Any>>

    /**
     * POST api/auth/login
     * 
     * 
     * Responses:
     *  - 200: 成功
     *  - 401: 请求失败
     *  - 429: 请求失败
     *
     * @param credentials 
     * @return [Call]<[kotlin.collections.Map<kotlin.String, kotlin.Any>]>
     */
    @POST("api/auth/login")
    fun login(@Body credentials: Credentials): Call<kotlin.collections.Map<kotlin.String, kotlin.Any>>

    /**
     * POST api/auth/logout
     * 
     * 
     * Responses:
     *  - 200: 成功
     *
     * @param xCSRFToken 
     * @return [Call]<[kotlin.collections.Map<kotlin.String, kotlin.Any>]>
     */
    @POST("api/auth/logout")
    fun logout(@Header("X-CSRF-Token") xCSRFToken: kotlin.String): Call<kotlin.collections.Map<kotlin.String, kotlin.Any>>

    /**
     * POST api/auth/setup
     * 
     * 
     * Responses:
     *  - 200: 成功
     *  - 409: 请求失败
     *
     * @param credentials 
     * @return [Call]<[kotlin.collections.Map<kotlin.String, kotlin.Any>]>
     */
    @POST("api/auth/setup")
    fun setupAdmin(@Body credentials: Credentials): Call<kotlin.collections.Map<kotlin.String, kotlin.Any>>

}
