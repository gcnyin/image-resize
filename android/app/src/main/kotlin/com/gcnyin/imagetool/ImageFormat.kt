package com.gcnyin.imagetool

import android.content.ContentResolver
import android.content.Context
import android.net.Uri
import java.io.IOException
import java.io.InputStream

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

    /**
     * WebP 是否为无损（首个图像数据块是 VP8L）。无损源保存时保持无损编码。
     *
     * 按 RIFF 结构逐个 chunk 遍历，不能只看固定偏移：ICCP / EXIF / XMP 等元数据块长度不固定，
     * 会把 VP8L 推到任意位置；动图还要看 ANMF 帧内的数据块。与 Windows 端实现保持一致
     * （windows/Services/ImageFormat.cs → IsWebpLossless）。
     */
    fun isWebpLossless(resolver: ContentResolver, uri: Uri): Boolean = try {
        resolver.openInputStream(uri)?.use { isWebpLossless(it) } ?: false
    } catch (_: IOException) {
        // 读失败按有损处理：保存走有损编码，结果不会比误当无损更糟
        false
    }

    /** 流式判定：只按 chunk 头前进，不把整个文件读进内存。 */
    internal fun isWebpLossless(input: InputStream): Boolean {
        fun readFully(buf: ByteArray): Boolean {
            var off = 0
            while (off < buf.size) {
                val n = input.read(buf, off, buf.size - off)
                if (n < 0) return false
                off += n
            }
            return true
        }

        fun skipFully(count: Long): Boolean {
            var left = count
            val buf = ByteArray(8 * 1024)
            while (left > 0) {
                val skipped = input.skip(left)
                if (skipped > 0) {
                    left -= skipped
                    continue
                }
                // skip 可能返回 0（如网络流），退回按块读
                val n = input.read(buf, 0, minOf(buf.size.toLong(), left).toInt())
                if (n < 0) return false
                left -= n
            }
            return true
        }

        val head = ByteArray(12)
        if (!readFully(head)) return false
        if (String(head, 0, 4, Charsets.US_ASCII) != "RIFF") return false
        if (String(head, 8, 4, Charsets.US_ASCII) != "WEBP") return false

        val chunk = ByteArray(8)
        while (readFully(chunk)) {
            val fourcc = String(chunk, 0, 4, Charsets.US_ASCII)
            val size = (chunk[4].toLong() and 0xFF) or
                ((chunk[5].toLong() and 0xFF) shl 8) or
                ((chunk[6].toLong() and 0xFF) shl 16) or
                ((chunk[7].toLong() and 0xFF) shl 24)
            when (fourcc) {
                "VP8L" -> return true // 无损图像数据
                "VP8 " -> return false // 有损图像数据
                "ANMF" -> {
                    // 动图帧：帧内前 16 字节是帧头，之后才是该帧的图像数据块
                    val frame = ByteArray(20)
                    if (!readFully(frame)) return false
                    return String(frame, 16, 4, Charsets.US_ASCII) == "VP8L"
                }
            }
            // 其余是元数据/容器块（VP8X / ICCP / EXIF / XMP / ALPH / ANIM …），整块跳过；RIFF 块按偶数字节对齐
            if (!skipFully(size + (size and 1L))) return false
        }
        return false
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
