package app.hwatch.client.apis

import app.hwatch.client.infrastructure.CollectionFormats.*
import retrofit2.http.*
import retrofit2.Call
import okhttp3.RequestBody
import com.google.gson.annotations.SerializedName

import app.hwatch.client.models.Error
import app.hwatch.client.models.RecoveryStatus
import app.hwatch.client.models.RestoreBackupRequest

interface SettingsApi {
    /**
     * POST api/backups
     * 
     * 
     * Responses:
     *  - 201: 成功
     *
     * @param xCSRFToken 
     * @return [Call]<[kotlin.collections.Map<kotlin.String, kotlin.Any>]>
     */
    @POST("api/backups")
    fun createBackup(@Header("X-CSRF-Token") xCSRFToken: kotlin.String): Call<kotlin.collections.Map<kotlin.String, kotlin.Any>>

    /**
     * GET api/settings
     * 
     * 
     * Responses:
     *  - 200: 成功
     *
     * @return [Call]<[kotlin.collections.Map<kotlin.String, kotlin.Any>]>
     */
    @GET("api/settings")
    fun getSettings(): Call<kotlin.collections.Map<kotlin.String, kotlin.Any>>

    /**
     * GET api/health
     * 
     * 
     * Responses:
     *  - 200: 成功
     *
     * @return [Call]<[kotlin.collections.Map<kotlin.String, kotlin.Any>]>
     */
    @GET("api/health")
    fun health(): Call<kotlin.collections.Map<kotlin.String, kotlin.Any>>

    /**
     * GET api/backups
     * 
     * 
     * Responses:
     *  - 200: 成功
     *
     * @return [Call]<[kotlin.collections.Map<kotlin.String, kotlin.Any>]>
     */
    @GET("api/backups")
    fun listBackups(): Call<kotlin.collections.Map<kotlin.String, kotlin.Any>>

    /**
     * GET api/recovery/latest
     * 
     * 
     * Responses:
     *  - 200: 最近一次数据库恢复协调摘要
     *
     * @return [Call]<[RecoveryStatus]>
     */
    @GET("api/recovery/latest")
    fun recoverySummary(): Call<RecoveryStatus>

    /**
     * POST api/backups/{name}/restore
     * 
     * 
     * Responses:
     *  - 202: 成功
     *  - 422: 请求失败
     *
     * @param name 
     * @param xCSRFToken 
     * @param restoreBackupRequest 
     * @return [Call]<[kotlin.collections.Map<kotlin.String, kotlin.Any>]>
     */
    @POST("api/backups/{name}/restore")
    fun restoreBackup(@Path("name") name: kotlin.String, @Header("X-CSRF-Token") xCSRFToken: kotlin.String, @Body restoreBackupRequest: RestoreBackupRequest): Call<kotlin.collections.Map<kotlin.String, kotlin.Any>>

    /**
     * PUT api/settings
     * 
     * 
     * Responses:
     *  - 200: 成功
     *
     * @param xCSRFToken 
     * @param requestBody 
     * @return [Call]<[kotlin.collections.Map<kotlin.String, kotlin.Any>]>
     */
    @PUT("api/settings")
    fun updateSettings(@Header("X-CSRF-Token") xCSRFToken: kotlin.String, @Body requestBody: kotlin.collections.Map<kotlin.String, kotlin.Any>): Call<kotlin.collections.Map<kotlin.String, kotlin.Any>>

}
