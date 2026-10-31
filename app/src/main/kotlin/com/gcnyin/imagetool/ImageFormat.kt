package com.gcnyin.imagetool

import android.content.ContentResolver
import android.content.Context
import android.net.Uri

/** 图片格式识别（按文件头魔数），用于保存时保持原格式。 */
object ImageFormat {

    enum class Kind(val mime: String, val ext: String) {
        JPEG("image/jpeg", "jpg"),
        PNG("image/png", "png"),
        WEBP("image/webp", "webp"),
        /** iPhone 照片常见格式；Android 无法编码 HEIF，保存时回退为 JPEG。 */
        HEIF("image/heif", "heif"),
        UNKNOWN("image/jpeg", "jpg"),
    }

    private val HEIF_BRANDS = setOf(
        "heic", "heix", "hevc", "hevx", "heim", "heis", "hevm", "hevs", "mif1", "msf1",
    )

    /** 根据文件头魔数判断格式。 */
    fun sniff(header: ByteArray): Kind {
        fun eq(bytes: ByteArray, offset: Int, vararg expected: Int): Boolean {
            if (offset + expected.size > bytes.size) return false
            for (i in expected.indices) {
                if (bytes[offset + i] != expected[i].toByte()) return false
            }
            return true
        }
        return when {
            eq(header, 0, 0xFF, 0xD8, 0xFF) -> Kind.JPEG
            eq(header, 0, 0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A) -> Kind.PNG
            eq(header, 0, 0x52, 0x49, 0x46, 0x46) && eq(header, 8, 0x57, 0x45, 0x42, 0x50) -> Kind.WEBP
            eq(header, 4, 0x66, 0x74, 0x79, 0x70) -> {
                val brand = String(header, 8, 4, Charsets.US_ASCII)
                if (brand in HEIF_BRANDS) Kind.HEIF else Kind.UNKNOWN
            }

            else -> Kind.UNKNOWN
        }
    }

    /** WebP 是否为无损（存在 VP8L chunk）。无损源保存时保持无损编码。 */
    fun isWebpLossless(header: ByteArray): Boolean {
        fun vp8l(offset: Int): Boolean =
            offset + 4 <= header.size &&
                header[offset] == 0x56.toByte() &&
                header[offset + 1] == 0x50.toByte() &&
                header[offset + 2] == 0x38.toByte() &&
                header[offset + 3] == 0x4C.toByte()
        // 简单 WebP：chunk 四字符码在偏移 12；扩展 WebP：VP8X 头之后（约 30）
        return vp8l(12) || vp8l(30)
    }

    /** 嗅探 Content Uri 的格式：文件头魔数优先，MIME 类型兜底。 */
    fun detect(context: Context, uri: Uri): Pair<Kind, ByteArray> {
        val resolver = context.contentResolver
        val header = readHeader(resolver, uri, 48)
        val kind = sniff(header)
        if (kind != Kind.UNKNOWN) return kind to header

        resolver.getType(uri)?.let { mime ->
            val byMime = when {
                mime.contains("jpeg") || mime.contains("jpg") -> Kind.JPEG
                mime.contains("png") -> Kind.PNG
                mime.contains("webp") -> Kind.WEBP
                mime.contains("heic") || mime.contains("heif") -> Kind.HEIF
                else -> null
            }
            if (byMime != null) return byMime to header
        }
        return Kind.UNKNOWN to header
    }

    internal fun readHeader(resolver: ContentResolver, uri: Uri, count: Int): ByteArray {
        resolver.openInputStream(uri)!!.use { input ->
            val buf = ByteArray(count)
            var off = 0
            while (off < count) {
                val n = input.read(buf, off, count - off)
                if (n < 0) break
                off += n
            }
            return buf.copyOf(off)
        }
    }
}
