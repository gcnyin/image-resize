package com.gcnyin.imagetool

import android.graphics.Bitmap
import android.graphics.Bitmap.CompressFormat
import android.os.Build
import java.io.ByteArrayOutputStream

/**
 * 恒定压缩率编码：目标字节数 = 源图每像素字节数 × 输出像素数，
 * 使缩放前后信息密度一致（例如缩小 50% 时体积约为原图的 1/4）。
 *
 * - JPEG / 有损 WebP：二分搜索编码质量逼近目标体积；
 * - PNG / 无损 WebP：保持无损直接编码（体积不可调，但内容零损失）；
 * - HEIF 源：Android 不支持 HEIF 编码，回退为 JPEG。
 */
object ImageEncoder {

    /** 质量搜索下限：再低画质明显劣化。 */
    const val MIN_QUALITY = 40

    /** 质量搜索上限：平台编码器可稳定达到的最高质量。 */
    const val MAX_QUALITY = 97

    data class Encoded(val bytes: ByteArray, val mime: String, val ext: String)

    fun encode(
        bitmap: Bitmap,
        sourceKind: ImageFormat.Kind,
        sourceFileBytes: Long,
        sourcePixels: Long,
        sourceWebpLossless: Boolean = false,
    ): Encoded {
        val outPixels = bitmap.width.toLong() * bitmap.height
        // 与源图保持相同的每像素字节（相同的压缩率 / 信息密度）
        val targetBytes = maxOf(4096L, sourceFileBytes * outPixels / sourcePixels)
        return when {
            sourceKind == ImageFormat.Kind.PNG ->
                encodeDirect(bitmap, CompressFormat.PNG, "image/png", "png")

            sourceKind == ImageFormat.Kind.WEBP && sourceWebpLossless && Build.VERSION.SDK_INT >= 30 ->
                encodeDirect(bitmap, CompressFormat.WEBP_LOSSLESS, "image/webp", "webp")

            sourceKind == ImageFormat.Kind.WEBP -> {
                val format =
                    if (Build.VERSION.SDK_INT >= 30) CompressFormat.WEBP_LOSSY
                    else @Suppress("DEPRECATION") CompressFormat.WEBP
                encodeByTarget(bitmap, format, targetBytes, "image/webp", "webp")
            }

            // JPEG、HEIF、未知格式 → JPEG
            else -> encodeByTarget(bitmap, CompressFormat.JPEG, targetBytes, "image/jpeg", "jpg")
        }
    }

    /** 无质量旋钮的格式：保持无损直接编码。 */
    private fun encodeDirect(bitmap: Bitmap, format: CompressFormat, mime: String, ext: String): Encoded {
        return Encoded(compress(bitmap, format, 100), mime, ext)
    }

    /** 恒定压缩率：二分搜索不超出目标体积的最大质量。 */
    private fun encodeByTarget(
        bitmap: Bitmap,
        format: CompressFormat,
        targetBytes: Long,
        mime: String,
        ext: String,
    ): Encoded {
        val quality = ImageMath.solveQuality(
            encode = { q -> compress(bitmap, format, q).size.toLong() },
            targetBytes = targetBytes,
            minQuality = MIN_QUALITY,
            maxQuality = MAX_QUALITY,
        )
        return Encoded(compress(bitmap, format, quality), mime, ext)
    }

    private fun compress(bitmap: Bitmap, format: CompressFormat, quality: Int): ByteArray {
        val out = ByteArrayOutputStream()
        check(bitmap.compress(format, quality, out)) { "编码失败" }
        return out.toByteArray()
    }
}
