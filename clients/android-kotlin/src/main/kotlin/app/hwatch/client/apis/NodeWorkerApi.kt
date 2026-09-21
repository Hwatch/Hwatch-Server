package app.hwatch.client.apis

import app.hwatch.client.infrastructure.CollectionFormats.*
import retrofit2.http.*
import retrofit2.Call
import okhttp3.RequestBody
import com.google.gson.annotations.SerializedName

import app.hwatch.client.models.ClaimComputeJobRequest
import app.hwatch.client.models.CompleteComputeJobRequest
import app.hwatch.client.models.Error
import app.hwatch.client.models.NodeFailure
import app.hwatch.client.models.NodeHeartbeat
import app.hwatch.client.models.NodeProgress

interface NodeWorkerApi {
    /**
     * POST api/node/v1/jobs/claim
     * 
     * 
     * Responses:
     *  - 200: 成功
     *  - 204: 当前没有可领取任务
     *  - 401: 请求失败
     *
     * @param claimComputeJobRequest 
     * @return [Call]<[kotlin.collections.Map<kotlin.String, kotlin.Any>]>
     */
    @POST("api/node/v1/jobs/claim")
    fun claimComputeJob(@Body claimComputeJobRequest: ClaimComputeJobRequest): Call<kotlin.collections.Map<kotlin.String, kotlin.Any>>

    /**
     * POST api/node/v1/jobs/{id}/complete
     * 
     * 
     * Responses:
     *  - 200: 成功
     *  - 409: 请求失败
     *  - 422: 请求失败
     *
     * @param id 
     * @param completeComputeJobRequest 
     * @return [Call]<[kotlin.collections.Map<kotlin.String, kotlin.Any>]>
     */
    @POST("api/node/v1/jobs/{id}/complete")
    fun completeComputeJob(@Path("id") id: kotlin.String, @Body completeComputeJobRequest: CompleteComputeJobRequest): Call<kotlin.collections.Map<kotlin.String, kotlin.Any>>

    /**
     * POST api/node/v1/jobs/{id}/fail
     * 
     * 
     * Responses:
     *  - 200: 成功
     *  - 409: 请求失败
     *
     * @param id 
     * @param nodeFailure 
     * @return [Call]<[kotlin.collections.Map<kotlin.String, kotlin.Any>]>
     */
    @POST("api/node/v1/jobs/{id}/fail")
    fun failComputeJob(@Path("id") id: kotlin.String, @Body nodeFailure: NodeFailure): Call<kotlin.collections.Map<kotlin.String, kotlin.Any>>

    /**
     * POST api/node/v1/heartbeat
     * 
     * 
     * Responses:
     *  - 200: 成功
     *  - 401: 请求失败
     *
     * @param nodeHeartbeat 
     * @return [Call]<[kotlin.collections.Map<kotlin.String, kotlin.Any>]>
     */
    @POST("api/node/v1/heartbeat")
    fun nodeHeartbeat(@Body nodeHeartbeat: NodeHeartbeat): Call<kotlin.collections.Map<kotlin.String, kotlin.Any>>

    /**
     * POST api/node/v1/jobs/{id}/progress
     * 
     * 
     * Responses:
     *  - 200: 成功
     *  - 409: 请求失败
     *
     * @param id 
     * @param nodeProgress 
     * @return [Call]<[kotlin.collections.Map<kotlin.String, kotlin.Any>]>
     */
    @POST("api/node/v1/jobs/{id}/progress")
    fun updateComputeJobProgress(@Path("id") id: kotlin.String, @Body nodeProgress: NodeProgress): Call<kotlin.collections.Map<kotlin.String, kotlin.Any>>

}
