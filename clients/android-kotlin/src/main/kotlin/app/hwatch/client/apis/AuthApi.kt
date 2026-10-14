package app.hwatch.client.apis

import app.hwatch.client.infrastructure.CollectionFormats.*
import retrofit2.http.*
import retrofit2.Call
import okhttp3.RequestBody
import com.google.gson.annotations.SerializedName

import app.hwatch.client.models.ChangeOwnPasswordRequest
import app.hwatch.client.models.CreateUserRequest
import app.hwatch.client.models.Credentials
import app.hwatch.client.models.Error
import app.hwatch.client.models.SetUserPasswordRequest
import app.hwatch.client.models.UpdateUserRoleRequest

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
     * POST api/auth/password
     * 
     * 修改当前登录账号的密码（任意已登录用户）。
     * Responses:
     *  - 200: 成功
     *  - 400: 请求失败
     *
     * @param xCSRFToken 
     * @param changeOwnPasswordRequest 
     * @return [Call]<[kotlin.collections.Map<kotlin.String, kotlin.Any>]>
     */
    @POST("api/auth/password")
    fun changeOwnPassword(@Header("X-CSRF-Token") xCSRFToken: kotlin.String, @Body changeOwnPasswordRequest: ChangeOwnPasswordRequest): Call<kotlin.collections.Map<kotlin.String, kotlin.Any>>

    /**
     * POST api/users
     * 
     * 创建用户（仅管理员）。role 为 admin 或 viewer，默认 viewer。
     * Responses:
     *  - 201: 成功
     *  - 400: 请求失败
     *  - 409: 请求失败
     *
     * @param xCSRFToken 
     * @param createUserRequest 
     * @return [Call]<[kotlin.collections.Map<kotlin.String, kotlin.Any>]>
     */
    @POST("api/users")
    fun createUser(@Header("X-CSRF-Token") xCSRFToken: kotlin.String, @Body createUserRequest: CreateUserRequest): Call<kotlin.collections.Map<kotlin.String, kotlin.Any>>

    /**
     * DELETE api/users/{id}
     * 
     * 删除用户（仅管理员，不能删除自己或最后一个管理员）。
     * Responses:
     *  - 200: 成功
     *  - 409: 请求失败
     *
     * @param id 
     * @param xCSRFToken 
     * @return [Call]<[kotlin.collections.Map<kotlin.String, kotlin.Any>]>
     */
    @DELETE("api/users/{id}")
    fun deleteUser(@Path("id") id: kotlin.Int, @Header("X-CSRF-Token") xCSRFToken: kotlin.String): Call<kotlin.collections.Map<kotlin.String, kotlin.Any>>

    /**
     * GET api/users
     * 
     * 列出用户（仅管理员）。
     * Responses:
     *  - 200: 成功
     *
     * @return [Call]<[kotlin.collections.Map<kotlin.String, kotlin.Any>]>
     */
    @GET("api/users")
    fun listUsers(): Call<kotlin.collections.Map<kotlin.String, kotlin.Any>>

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
     * POST api/users/{id}/password
     * 
     * 重置指定用户密码（仅管理员）。
     * Responses:
     *  - 200: 成功
     *  - 400: 请求失败
     *
     * @param id 
     * @param xCSRFToken 
     * @param setUserPasswordRequest 
     * @return [Call]<[kotlin.collections.Map<kotlin.String, kotlin.Any>]>
     */
    @POST("api/users/{id}/password")
    fun setUserPassword(@Path("id") id: kotlin.Int, @Header("X-CSRF-Token") xCSRFToken: kotlin.String, @Body setUserPasswordRequest: SetUserPasswordRequest): Call<kotlin.collections.Map<kotlin.String, kotlin.Any>>

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

    /**
     * PATCH api/users/{id}
     * 
     * 修改用户角色（仅管理员，不能降级最后一个管理员）。
     * Responses:
     *  - 200: 成功
     *  - 409: 请求失败
     *
     * @param id 
     * @param xCSRFToken 
     * @param updateUserRoleRequest 
     * @return [Call]<[kotlin.collections.Map<kotlin.String, kotlin.Any>]>
     */
    @PATCH("api/users/{id}")
    fun updateUserRole(@Path("id") id: kotlin.Int, @Header("X-CSRF-Token") xCSRFToken: kotlin.String, @Body updateUserRoleRequest: UpdateUserRoleRequest): Call<kotlin.collections.Map<kotlin.String, kotlin.Any>>

}
