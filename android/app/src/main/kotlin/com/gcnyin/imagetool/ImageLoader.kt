package com.gcnyin.imagetool

import android.content.ContentResolver
import android.content.Context
import android.graphics.Bitmap
import android.graphics.BitmapFactory
import android.graphics.Matrix
import android.net.Uri
import androidx.exifinterface.media.ExifInterface

/** 图片加载结果。 */
data class LoadedImage(
    val bitmap: Bitmap,
    /** 文件中的原始像素尺寸（应用 EXIF 方向之前）。 */
    val sourceWidth: Int,
    val sourceHeight: Int,
    /** 源文件字节数，用于输出码率对齐。 */
    val sourceFileBytes: Long,
    val format: ImageFormat.Kind,
    val webpLossless: Boolean,
)

/** 从 Content Uri 解码图片，识别原格式，处理 EXIF 方向并限制工作区尺寸。 */
object ImageLoader {

    /** 解码工作区最长边上限，避免超大图片占用过多内存。 */
    const val MAX_WORK_DIM = 4096

    fun load(context: Context, uri: Uri): LoadedImage {
        val resolver = context.contentResolver
        val (kind, header) = ImageFormat.detect(context, uri)

        val bounds = BitmapFactory.Options().apply { inJustDecodeBounds = true }
        resolver.openInputStream(uri)!!.use {
            BitmapFactory.decodeStream(it, null, bounds)
        }
        check(bounds.outWidth > 0 && bounds.outHeight > 0) { "无法解码该图片" }

        val opts = BitmapFactory.Options().apply {
            inSampleSize = ImageMath.inSampleSizeForCap(bounds.outWidth, bounds.outHeight, MAX_WORK_DIM)
            inPreferredConfig = Bitmap.Config.ARGB_8888
        }
        val raw = resolver.openInputStream(uri)!!.use {
            BitmapFactory.decodeStream(it, null, opts)
        } ?: error("无法解码该图片")

        // 按 EXIF 方向旋转到正立方向
        val rotation = resolver.openInputStream(uri)!!.use { stream ->
            when (ExifInterface(stream).getAttributeInt(ExifInterface.TAG_ORIENTATION, 1)) {
                ExifInterface.ORIENTATION_TRANSPOSE,
                ExifInterface.ORIENTATION_ROTATE_90 -> 90f

                ExifInterface.ORIENTATION_TRANSVERSE,
                ExifInterface.ORIENTATION_ROTATE_180 -> 180f

                ExifInterface.ORIENTATION_ROTATE_270 -> 270f

                else -> 0f
            }
        }
        val bitmap = if (rotation != 0f) {
            val matrix = Matrix().apply { postRotate(rotation) }
            Bitmap.createBitmap(raw, 0, 0, raw.width, raw.height, matrix, true).also {
                if (it !== raw) raw.recycle()
            }
        } else raw

        return LoadedImage(
            bitmap = bitmap,
            sourceWidth = bounds.outWidth,
            sourceHeight = bounds.outHeight,
            sourceFileBytes = fileSize(resolver, uri),
            format = kind,
            webpLossless = kind == ImageFormat.Kind.WEBP && ImageFormat.isWebpLossless(header),
        )
    }

    /** 源文件字节数：优先读描述符长度，失败则数一遍流。 */
    private fun fileSize(resolver: ContentResolver, uri: Uri): Long {
        try {
            resolver.openAssetFileDescriptor(uri, "r")?.use { afd ->
                val len = afd.length
                if (len >= 0) return len
            }
        } catch (_: Exception) {
            // 某些 provider 不支持，走兜底
        }
        resolver.openInputStream(uri)!!.use { input ->
            var total = 0L
            val buf = ByteArray(64 * 1024)
            while (true) {
                val n = input.read(buf)
                if (n < 0) break
                total += n
            }
            return total
        }
    }
}
