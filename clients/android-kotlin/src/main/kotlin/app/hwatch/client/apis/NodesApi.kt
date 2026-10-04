package app.hwatch.client.apis

import app.hwatch.client.infrastructure.CollectionFormats.*
import retrofit2.http.*
import retrofit2.Call
import okhttp3.RequestBody
import com.google.gson.annotations.SerializedName

import app.hwatch.client.models.CommitTranscodeJobRequest
import app.hwatch.client.models.CreateComputeJob
import app.hwatch.client.models.CreateComputeNode
import app.hwatch.client.models.Error

interface NodesApi {
    /**
     * POST api/compute/jobs/{id}/cancel
     * 
     * 
     * Responses:
     *  - 200: 成功
     *
     * @param id 
     * @param xCSRFToken 
     * @return [Call]<[kotlin.collections.Map<kotlin.String, kotlin.Any>]>
     */
    @POST("api/compute/jobs/{id}/cancel")
    fun cancelComputeJob(@Path("id") id: kotlin.String, @Header("X-CSRF-Token") xCSRFToken: kotlin.String): Call<kotlin.collections.Map<kotlin.String, kotlin.Any>>

    /**
     * POST api/compute/jobs/{id}/commit
     * 
     * 兼容接口。节点回传的转码产物现在会在验证通过后自动原子提交，管理员无需确认；该端点仅用于仍处于 &#x60;verifying&#x60; 的旧任务。
     * Responses:
     *  - 200: 成功
     *  - 409: 请求失败
     *  - 422: 请求失败
     *
     * @param id 
     * @param xCSRFToken 
     * @param commitTranscodeJobRequest 
     * @return [Call]<[kotlin.collections.Map<kotlin.String, kotlin.Any>]>
     */
    @POST("api/compute/jobs/{id}/commit")
    fun commitTranscodeJob(@Path("id") id: kotlin.String, @Header("X-CSRF-Token") xCSRFToken: kotlin.String, @Body commitTranscodeJobRequest: CommitTranscodeJobRequest): Call<kotlin.collections.Map<kotlin.String, kotlin.Any>>

    /**
     * POST api/compute/jobs
     * 
     * 
     * Responses:
     *  - 200: 成功
     *  - 201: 成功
     *  - 409: 请求失败
     *  - 422: 请求失败
     *
     * @param xCSRFToken 
     * @param createComputeJob 
     * @return [Call]<[kotlin.collections.Map<kotlin.String, kotlin.Any>]>
     */
    @POST("api/compute/jobs")
    fun createComputeJob(@Header("X-CSRF-Token") xCSRFToken: kotlin.String, @Body createComputeJob: CreateComputeJob): Call<kotlin.collections.Map<kotlin.String, kotlin.Any>>

    /**
     * POST api/nodes
     * 
     * 
     * Responses:
     *  - 201: 成功
     *  - 422: 请求失败
     *
     * @param xCSRFToken 
     * @param createComputeNode 
     * @return [Call]<[kotlin.collections.Map<kotlin.String, kotlin.Any>]>
     */
    @POST("api/nodes")
    fun createComputeNode(@Header("X-CSRF-Token") xCSRFToken: kotlin.String, @Body createComputeNode: CreateComputeNode): Call<kotlin.collections.Map<kotlin.String, kotlin.Any>>

    /**
     * GET api/compute/jobs
     * 
     * 
     * Responses:
     *  - 200: 成功
     *
     * @param limit  (optional, default to 200)
     * @param offset  (optional, default to 0)
     * @return [Call]<[kotlin.collections.Map<kotlin.String, kotlin.Any>]>
     */
    @GET("api/compute/jobs")
    fun listComputeJobs(@Query("limit") limit: kotlin.Int? = 200, @Query("offset") offset: kotlin.Int? = 0): Call<kotlin.collections.Map<kotlin.String, kotlin.Any>>

    /**
     * GET api/nodes
     * 
     * 
     * Responses:
     *  - 200: 成功
     *
     * @return [Call]<[kotlin.collections.Map<kotlin.String, kotlin.Any>]>
     */
    @GET("api/nodes")
    fun listComputeNodes(): Call<kotlin.collections.Map<kotlin.String, kotlin.Any>>

    /**
     * DELETE api/nodes/{id}
     * 
     * 
     * Responses:
     *  - 200: 成功
     *
     * @param id 
     * @param xCSRFToken 
     * @return [Call]<[kotlin.collections.Map<kotlin.String, kotlin.Any>]>
     */
    @DELETE("api/nodes/{id}")
    fun revokeComputeNode(@Path("id") id: kotlin.String, @Header("X-CSRF-Token") xCSRFToken: kotlin.String): Call<kotlin.collections.Map<kotlin.String, kotlin.Any>>

}
