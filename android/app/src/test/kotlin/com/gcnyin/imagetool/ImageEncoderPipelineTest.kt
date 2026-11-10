package com.gcnyin.imagetool

import android.graphics.Bitmap
import android.graphics.BitmapFactory
import org.junit.Assert.assertTrue
import org.junit.Assume.assumeTrue
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.annotation.Config
import java.io.File
import java.util.Locale

/**
 * 端到端验证（Robolectric 跑真实的 skia 解码/缩放/编码管线）：
 * 对 test_img 下的样例执行 50% 缩放 + 恒定压缩率编码，
 * 确认输出体积按像素比例收敛、绝不大于原图。
 */
@RunWith(RobolectricTestRunner::class)
@Config(sdk = [34])
class ImageEncoderPipelineTest {

    private fun sampleFiles(): List<File> =
        File("..", "test_img").listFiles()
            ?.filter { it.isFile && it.extension.lowercase() in setOf("jpg", "jpeg", "png", "webp") }
            .orEmpty()

    @Test
    fun resize50Percent_outputSmallerThanSource() {
        val files = sampleFiles()
        assumeTrue("test_img 下没有样例图片", files.isNotEmpty())

        println("RESULT 文件 | 原始尺寸 | 原体积 | 缩放后 | 输出体积 | 体积比 | 输出格式")
        for (f in files) {
            val srcBytes = f.length()
            val rawHeader = f.readBytes()
            val header = if (rawHeader.size > 48) rawHeader.copyOf(48) else rawHeader
            val kind = ImageFormat.sniff(header)

            val opts = BitmapFactory.Options().apply { inPreferredConfig = Bitmap.Config.ARGB_8888 }
            val src = BitmapFactory.decodeFile(f.absolutePath, opts)
            if (src == null) {
                println("SKIP ${f.name}（该环境无法解码此格式）")
                continue
            }

            val scaled = ImageOps.scaleByPercent(src, 50)
            val enc = ImageEncoder.encode(
                scaled,
                kind,
                srcBytes,
                src.width.toLong() * src.height,
                kind == ImageFormat.Kind.WEBP && ImageFormat.isWebpLossless(header),
            )
            val ratio = String.format(Locale.US, "%.1f%%", enc.bytes.size * 100.0 / srcBytes)
            println(
                "RESULT ${f.name} | ${src.width}×${src.height} | $srcBytes | " +
                    "${scaled.width}×${scaled.height} | ${enc.bytes.size} | $ratio | ${enc.ext}",
            )
            assertTrue(
                "${f.name}: 50% 缩放后输出（${enc.bytes.size}B）不应大于原图（${srcBytes}B）",
                enc.bytes.size < srcBytes,
            )
            src.recycle()
        }
    }
}
