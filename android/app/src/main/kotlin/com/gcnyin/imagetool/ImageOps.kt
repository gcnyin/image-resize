package com.gcnyin.imagetool

import android.graphics.Bitmap
import android.graphics.RectF
import kotlin.math.roundToInt

/** 剪裁与按百分比缩放操作。 */
object ImageOps {

    /**
     * 按百分比缩小图片。
     *
     * 为避免大倍率一步缩小时的走样（锯齿），先按 1/2 步进逐级下采样到
     * 目标尺寸的两倍以内，每级都启用双线性滤波，最后再缩放到精确目标尺寸。
     * 内容不变，只改变像素尺寸（例如 2048px → 1024px）。
     */
    fun scaleByPercent(src: Bitmap, percent: Int): Bitmap {
        require(percent in 1..100) { "比例需在 1~100 之间" }
        val targetW = ImageMath.targetSize(src.width, percent)
        val targetH = ImageMath.targetSize(src.height, percent)
        if (targetW == src.width && targetH == src.height) return src

        var cur = src
        while (cur.width / 2 >= targetW && cur.height / 2 >= targetH) {
            val next = Bitmap.createScaledBitmap(cur, cur.width / 2, cur.height / 2, true)
            if (next !== cur && cur !== src) cur.recycle()
            cur = next
        }
        if (cur.width != targetW || cur.height != targetH) {
            val next = Bitmap.createScaledBitmap(cur, targetW, targetH, true)
            if (next !== cur && cur !== src) cur.recycle()
            cur = next
        }
        return cur
    }

    /**
     * 按归一化 [0,1] 矩形剪裁位图。
     * 若选框已覆盖全图则原样返回。
     */
    fun crop(src: Bitmap, norm: RectF): Bitmap {
        val x = (norm.left * src.width).roundToInt().coerceIn(0, src.width - 1)
        val y = (norm.top * src.height).roundToInt().coerceIn(0, src.height - 1)
        val w = (norm.width() * src.width).roundToInt().coerceIn(1, src.width - x)
        val h = (norm.height() * src.height).roundToInt().coerceIn(1, src.height - y)
        if (x == 0 && y == 0 && w == src.width && h == src.height) return src
        return Bitmap.createBitmap(src, x, y, w, h)
    }
}
