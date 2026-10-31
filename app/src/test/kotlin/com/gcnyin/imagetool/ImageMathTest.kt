package com.gcnyin.imagetool

import org.junit.Assert.assertEquals
import org.junit.Test

class ImageMathTest {

    @Test
    fun targetSize_at50Percent_halvesExactly() {
        assertEquals(1024, ImageMath.targetSize(2048, 50))
        assertEquals(768, ImageMath.targetSize(1536, 50))
    }

    @Test
    fun targetSize_roundsToNearestPixel() {
        assertEquals(1025, ImageMath.targetSize(2050, 50))
        assertEquals(205, ImageMath.targetSize(2048, 10))
    }

    @Test
    fun targetSize_neverBelowOnePixel() {
        assertEquals(1, ImageMath.targetSize(1, 10))
        assertEquals(1, ImageMath.targetSize(4, 1))
    }

    @Test
    fun targetSize_at100Percent_keepsOriginal() {
        assertEquals(2048, ImageMath.targetSize(2048, 100))
    }

    @Test
    fun inSampleSize_capNotExceeded() {
        assertEquals(1, ImageMath.inSampleSizeForCap(4096, 4096, 4096))
        assertEquals(2, ImageMath.inSampleSizeForCap(8192, 4096, 4096))
        assertEquals(4, ImageMath.inSampleSizeForCap(10000, 8000, 4096))
    }

    @Test
    fun inSampleSize_smallImageUntouched() {
        assertEquals(1, ImageMath.inSampleSizeForCap(1920, 1080, 4096))
    }

    @Test
    fun solveQuality_findsLargestQualityUnderTarget() {
        // 模拟质量→体积的单调函数：size = q × 100
        assertEquals(55, ImageMath.solveQuality({ it * 100L }, 5_500L, 40, 97))
    }

    @Test
    fun solveQuality_allExceedTarget_returnsMin() {
        assertEquals(40, ImageMath.solveQuality({ 10_000L }, 500L, 40, 97))
    }

    @Test
    fun solveQuality_allUnderTarget_returnsMax() {
        assertEquals(97, ImageMath.solveQuality({ 10L }, 5_000L, 40, 97))
    }
}
