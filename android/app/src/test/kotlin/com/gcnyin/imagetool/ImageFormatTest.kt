package com.gcnyin.imagetool

import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test
import java.io.ByteArrayOutputStream

/** WebP 无损判定：按 RIFF 结构遍历 chunk，元数据块之后的 VP8L 也要认出来。 */
class ImageFormatTest {

    private fun chunk(fourcc: String, payload: ByteArray): ByteArray {
        val out = ByteArrayOutputStream()
        out.write(fourcc.toByteArray(Charsets.US_ASCII))
        val size = payload.size
        out.write(byteArrayOf(size.toByte(), (size shr 8).toByte(), (size shr 16).toByte(), (size shr 24).toByte()))
        out.write(payload)
        if (payload.size and 1 == 1) out.write(0) // RIFF 块按偶数字节对齐
        return out.toByteArray()
    }

    private fun webp(vararg chunks: Pair<String, ByteArray>): ByteArray {
        val body = ByteArrayOutputStream()
        chunks.forEach { (fourcc, payload) -> body.write(chunk(fourcc, payload)) }
        val payload = body.toByteArray()
        val out = ByteArrayOutputStream()
        out.write("RIFF".toByteArray(Charsets.US_ASCII))
        val riffSize = payload.size + 4
        out.write(
            byteArrayOf(
                riffSize.toByte(), (riffSize shr 8).toByte(),
                (riffSize shr 16).toByte(), (riffSize shr 24).toByte(),
            ),
        )
        out.write("WEBP".toByteArray(Charsets.US_ASCII))
        out.write(payload)
        return out.toByteArray()
    }

    private fun isLossless(bytes: ByteArray): Boolean = ImageFormat.isWebpLossless(bytes.inputStream())

    @Test
    fun detectsSimpleWebp() {
        assertTrue(isLossless(webp("VP8L" to ByteArray(8) { 0x2F })))
        assertFalse(isLossless(webp("VP8 " to ByteArray(8) { 0x30 })))
    }

    @Test
    fun detectsVp8lAfterMetadataChunks() {
        // ICCP / EXIF 长度不固定（这里 ICCP 还是奇数长度），会把 VP8L 推离固定偏移
        val bytes = webp(
            "VP8X" to ByteArray(10),
            "ICCP" to ByteArray(5) { 0x11 },
            "EXIF" to byteArrayOf(1, 2, 3, 4),
            "VP8L" to ByteArray(6) { 0x2F },
        )
        assertTrue(isLossless(bytes))
    }

    @Test
    fun detectsAnimatedFrameLosslessFlag() {
        fun frame(inner: String) = ByteArray(16) + chunk(inner, ByteArray(4))

        assertTrue(isLossless(webp("VP8X" to ByteArray(10), "ANIM" to ByteArray(6), "ANMF" to frame("VP8L"))))
        assertFalse(isLossless(webp("VP8X" to ByteArray(10), "ANIM" to ByteArray(6), "ANMF" to frame("VP8 "))))
    }

    @Test
    fun rejectsNonWebpAndTruncatedData() {
        assertFalse(isLossless(byteArrayOf(0xFF.toByte(), 0xD8.toByte(), 0xFF.toByte(), 0xE0.toByte())))
        assertFalse(isLossless(ByteArray(0)))
        // 只有 RIFF 头，没有 chunk
        assertFalse(isLossless(webp()))
        // 声明长度超出文件末尾，不崩、按有损处理
        val truncated = webp("VP8 " to ByteArray(4)).copyOf(16)
        assertFalse(isLossless(truncated))
    }
}
