package app.hwatch.client.apis

import app.hwatch.client.infrastructure.CollectionFormats.*
import retrofit2.http.*
import retrofit2.Call
import okhttp3.RequestBody
import com.google.gson.annotations.SerializedName

import app.hwatch.client.models.CreateTransRunnerTokenRequest
import app.hwatch.client.models.Error
import app.hwatch.client.models.TransRunnerDedupeCandidates
import app.hwatch.client.models.TransRunnerDuplicateResult
import app.hwatch.client.models.TransRunnerMedia
import app.hwatch.client.models.TransRunnerTranscodeResult

interface TransRunnerApi {
    /**
     * POST api/integrations/transrunner/token
     * 
     * 
     * Responses:
     *  - 201: 成功
     *  - 409: 请求失败
     *
     * @param xCSRFToken 
     * @param createTransRunnerTokenRequest  (optional)
     * @return [Call]<[kotlin.collections.Map<kotlin.String, kotlin.Any>]>
     */
    @POST("api/integrations/transrunner/token")
    fun createTransRunnerToken(@Header("X-CSRF-Token") xCSRFToken: kotlin.String, @Body createTransRunnerTokenRequest: CreateTransRunnerTokenRequest? = null): Call<kotlin.collections.Map<kotlin.String, kotlin.Any>>

    /**
     * GET api/integrations/transrunner/token
     * 
     * 
     * Responses:
     *  - 200: 成功
     *
     * @return [Call]<[kotlin.collections.Map<kotlin.String, kotlin.Any>]>
     */
    @GET("api/integrations/transrunner/token")
    fun getTransRunnerToken(): Call<kotlin.collections.Map<kotlin.String, kotlin.Any>>

    /**
     * DELETE api/integrations/transrunner/token
     * 
     * 
     * Responses:
     *  - 200: 成功
     *
     * @param xCSRFToken 
     * @return [Call]<[kotlin.collections.Map<kotlin.String, kotlin.Any>]>
     */
    @DELETE("api/integrations/transrunner/token")
    fun revokeTransRunnerToken(@Header("X-CSRF-Token") xCSRFToken: kotlin.String): Call<kotlin.collections.Map<kotlin.String, kotlin.Any>>

    /**
     * GET api/transrunner/v1/media/{video_id}/dedupe-candidates
     * 
     * 需要 read_dedupe_candidates。候选由 HWatch 服务端筛选，调用方不扫描全库。
     * Responses:
     *  - 200: 去重候选
     *  - 401: 请求失败
     *  - 403: 请求失败
     *  - 404: 请求失败
     *
     * @param videoId 
     * @param limit  (optional, default to 20)
     * @return [Call]<[TransRunnerDedupeCandidates]>
     */
    @GET("api/transrunner/v1/media/{video_id}/dedupe-candidates")
    fun transRunnerDedupeCandidates(@Path("video_id") videoId: kotlin.String, @Query("limit") limit: kotlin.Int? = 20): Call<TransRunnerDedupeCandidates>

    /**
     * POST api/transrunner/v1/duplicate-results
     * 
     * 需要 write_duplicate_results。仅写入重复候选，绝不删除或替换媒体。
     * Responses:
     *  - 200: 成功
     *  - 401: 请求失败
     *  - 403: 请求失败
     *  - 404: 请求失败
     *  - 422: 请求失败
     *
     * @param transRunnerDuplicateResult 
     * @return [Call]<[kotlin.collections.Map<kotlin.String, kotlin.Any>]>
     */
    @POST("api/transrunner/v1/duplicate-results")
    fun transRunnerDuplicateResults(@Body transRunnerDuplicateResult: TransRunnerDuplicateResult): Call<kotlin.collections.Map<kotlin.String, kotlin.Any>>

    /**
     * GET api/transrunner/v1/media/{video_id}
     * 
     * 需要 read_media。只返回相对媒体路径与媒体元数据，不返回绝对路径或凭据。
     * Responses:
     *  - 200: 媒体元数据
     *  - 401: 请求失败
     *  - 403: 请求失败
     *  - 404: 请求失败
     *  - 503: 请求失败
     *
     * @param videoId 
     * @return [Call]<[TransRunnerMedia]>
     */
    @GET("api/transrunner/v1/media/{video_id}")
    fun transRunnerMedia(@Path("video_id") videoId: kotlin.String): Call<TransRunnerMedia>

    /**
     * POST api/transrunner/v1/transcode-results
     * 
     * 需要 write_transcode_results。接收暂存产物并由 HWatch 自己 ffprobe 验证。默认只记录 pending 结果，不替换源文件；仅当 apply_mode&#x3D;replace_source 且校验通过（HEVC、分辨率/帧率/时长/位深/色彩保持、source_sha256 一致）时才由 HWatch 原子替换源文件并刷新元数据。
     * Responses:
     *  - 200: 成功
     *  - 202: 成功
     *  - 401: 请求失败
     *  - 403: 请求失败
     *  - 404: 请求失败
     *  - 409: 请求失败
     *  - 422: 请求失败
     *
     * @param transRunnerTranscodeResult 
     * @return [Call]<[kotlin.collections.Map<kotlin.String, kotlin.Any>]>
     */
    @POST("api/transrunner/v1/transcode-results")
    fun transRunnerTranscodeResults(@Body transRunnerTranscodeResult: TransRunnerTranscodeResult): Call<kotlin.collections.Map<kotlin.String, kotlin.Any>>

}
