package app.hwatch.client.apis

import app.hwatch.client.infrastructure.CollectionFormats.*
import retrofit2.http.*
import retrofit2.Call
import okhttp3.RequestBody
import com.google.gson.annotations.SerializedName


interface TranslationApi {
    /**
     * GET api/translations/export
     * 
     * 
     * Responses:
     *  - 200: 成功
     *
     * @param language  (optional, default to "zh-CN")
     * @return [Call]<[kotlin.collections.Map<kotlin.String, kotlin.Any>]>
     */
    @GET("api/translations/export")
    fun exportTranslations(@Query("language") language: kotlin.String? = "zh-CN"): Call<kotlin.collections.Map<kotlin.String, kotlin.Any>>

    /**
     * POST api/translations/import
     * 
     * 
     * Responses:
     *  - 200: 成功
     *
     * @param xCSRFToken 
     * @param requestBody 
     * @return [Call]<[kotlin.collections.Map<kotlin.String, kotlin.Any>]>
     */
    @POST("api/translations/import")
    fun importTranslations(@Header("X-CSRF-Token") xCSRFToken: kotlin.String, @Body requestBody: kotlin.collections.Map<kotlin.String, kotlin.Any>): Call<kotlin.collections.Map<kotlin.String, kotlin.Any>>

    /**
     * POST api/plex/refresh-nfo
     * 
     * 
     * Responses:
     *  - 202: 成功
     *
     * @param xCSRFToken 
     * @return [Call]<[kotlin.collections.Map<kotlin.String, kotlin.Any>]>
     */
    @POST("api/plex/refresh-nfo")
    fun refreshNFO(@Header("X-CSRF-Token") xCSRFToken: kotlin.String): Call<kotlin.collections.Map<kotlin.String, kotlin.Any>>

}
