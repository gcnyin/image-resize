package com.gcnyin.imagetool

import android.content.ContentValues
import android.content.Context
import android.provider.MediaStore
import java.text.SimpleDateFormat
import java.util.Date
import java.util.Locale

/** 通过 MediaStore 保存到系统相册（API 29+，无需存储权限）。 */
object ImageSaver {

    private const val REL_DIR = "Pictures/图片工具"

    /** 保存已编码的字节，返回展示用相对路径。 */
    fun save(context: Context, data: ByteArray, mime: String): String {
        val stamp = SimpleDateFormat("yyyyMMdd_HHmmss", Locale.US).format(Date())
        val name = "imagetool_$stamp.${extFor(mime)}"

        val values = ContentValues().apply {
            put(MediaStore.Images.Media.DISPLAY_NAME, name)
            put(MediaStore.Images.Media.MIME_TYPE, mime)
            put(MediaStore.Images.Media.RELATIVE_PATH, REL_DIR)
            put(MediaStore.Images.Media.IS_PENDING, 1)
        }
        val resolver = context.contentResolver
        val collection = MediaStore.Images.Media.getContentUri(MediaStore.VOLUME_EXTERNAL_PRIMARY)
        val uri = resolver.insert(collection, values) ?: error("无法创建相册文件")

        try {
            resolver.openOutputStream(uri)!!.use { out ->
                out.write(data)
                out.flush()
            }
            values.clear()
            values.put(MediaStore.Images.Media.IS_PENDING, 0)
            resolver.update(uri, values, null, null)
        } catch (t: Throwable) {
            resolver.delete(uri, null, null)
            throw t
        }
        return "$REL_DIR/$name"
    }

    private fun extFor(mime: String): String = when {
        mime.contains("png") -> "png"
        mime.contains("webp") -> "webp"
        else -> "jpg"
    }
}
