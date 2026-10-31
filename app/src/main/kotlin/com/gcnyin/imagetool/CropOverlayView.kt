package com.gcnyin.imagetool

import android.content.Context
import android.graphics.Canvas
import android.graphics.Paint
import android.graphics.PointF
import android.graphics.RectF
import android.util.AttributeSet
import android.view.MotionEvent
import android.view.View
import android.view.ViewConfiguration
import kotlin.math.max
import kotlin.math.min

/**
 * 剪裁选框覆盖层。
 *
 * 在 [imageRect]（图片显示区域，视图坐标）内维护一个可拖动、可缩放的选框，
 * 对外以相对图片的归一化坐标 [0,1] 暴露，与位图实际尺寸解耦。
 *
 * 手势：拖动四角缩放选框；在选框内拖动整体移动；在选框外按下并拖动重新框选。
 */
class CropOverlayView @JvmOverloads constructor(
    context: Context,
    attrs: AttributeSet? = null,
) : View(context, attrs) {

    private enum class DragMode { NONE, MOVE, TL, TR, BL, BR, NEW }

    /** 图片显示区域（视图坐标），空矩形表示没有图片。 */
    private val imageRect = RectF()

    /** 选框，归一化到 imageRect 的 [0,1] 区间。 */
    private val cropNorm = RectF(0f, 0f, 1f, 1f)
    private val dragStartNorm = RectF()

    private val downPoint = PointF()
    private val moveOffset = PointF()
    private var mode = DragMode.NONE

    private val density = resources.displayMetrics.density
    private val touchSlop = ViewConfiguration.get(context).scaledTouchSlop
    private val minSizePx = density * 48f
    private val cornerLenPx = density * 22f

    private val dimPaint = Paint().apply {
        color = 0xB3000000.toInt()
        style = Paint.Style.FILL
    }
    private val framePaint = Paint(Paint.ANTI_ALIAS_FLAG).apply {
        color = 0xFFFFFFFF.toInt()
        style = Paint.Style.STROKE
        strokeWidth = density * 1.5f
    }
    private val cornerPaint = Paint(Paint.ANTI_ALIAS_FLAG).apply {
        color = 0xFFFFFFFF.toInt()
        style = Paint.Style.STROKE
        strokeWidth = density * 3f
        strokeCap = Paint.Cap.ROUND
    }
    private val gridPaint = Paint(Paint.ANTI_ALIAS_FLAG).apply {
        color = 0x59FFFFFF
        style = Paint.Style.STROKE
        strokeWidth = density * 0.75f
    }

    /** 选框变化回调（归一化坐标）。 */
    var onCropChanged: ((RectF) -> Unit)? = null

    /** 更新图片显示区域；传 null 表示没有图片。 */
    fun setImageRect(rect: RectF?) {
        if (rect == null) imageRect.setEmpty() else imageRect.set(rect)
        invalidate()
    }

    /** 选框重置为全图。 */
    fun resetToFull() {
        cropNorm.set(0f, 0f, 1f, 1f)
        invalidate()
        notifyChanged()
    }

    /** 当前选框（归一化坐标）副本。 */
    fun cropRectNorm(): RectF = RectF(cropNorm)

    override fun onDraw(canvas: Canvas) {
        super.onDraw(canvas)
        if (imageRect.isEmpty) return
        val c = pixelRect()
        // 压暗选框外区域
        canvas.drawRect(imageRect.left, imageRect.top, imageRect.right, c.top, dimPaint)
        canvas.drawRect(imageRect.left, c.bottom, imageRect.right, imageRect.bottom, dimPaint)
        canvas.drawRect(imageRect.left, c.top, c.left, c.bottom, dimPaint)
        canvas.drawRect(c.right, c.top, imageRect.right, c.bottom, dimPaint)
        // 三分辅助线
        canvas.drawLine(c.left + c.width() / 3f, c.top, c.left + c.width() / 3f, c.bottom, gridPaint)
        canvas.drawLine(c.left + c.width() * 2f / 3f, c.top, c.left + c.width() * 2f / 3f, c.bottom, gridPaint)
        canvas.drawLine(c.left, c.top + c.height() / 3f, c.right, c.top + c.height() / 3f, gridPaint)
        canvas.drawLine(c.left, c.top + c.height() * 2f / 3f, c.right, c.top + c.height() * 2f / 3f, gridPaint)
        // 边框与四角手柄
        canvas.drawRect(c, framePaint)
        drawCorner(canvas, c, left = true, top = true)
        drawCorner(canvas, c, left = false, top = true)
        drawCorner(canvas, c, left = true, top = false)
        drawCorner(canvas, c, left = false, top = false)
    }

    private fun drawCorner(canvas: Canvas, c: RectF, left: Boolean, top: Boolean) {
        val x = if (left) c.left else c.right
        val y = if (top) c.top else c.bottom
        val dx = if (left) cornerLenPx else -cornerLenPx
        val dy = if (top) cornerLenPx else -cornerLenPx
        canvas.drawLine(x, y, x + dx, y, cornerPaint)
        canvas.drawLine(x, y, x, y + dy, cornerPaint)
    }

    private fun pixelRect(): RectF = RectF(
        imageRect.left + cropNorm.left * imageRect.width(),
        imageRect.top + cropNorm.top * imageRect.height(),
        imageRect.left + cropNorm.right * imageRect.width(),
        imageRect.top + cropNorm.bottom * imageRect.height(),
    )

    private fun normX(x: Float): Float = ((x - imageRect.left) / imageRect.width()).coerceIn(0f, 1f)

    private fun normY(y: Float): Float = ((y - imageRect.top) / imageRect.height()).coerceIn(0f, 1f)

    private fun minNormX(): Float = (minSizePx / imageRect.width()).coerceIn(0f, 1f)

    private fun minNormY(): Float = (minSizePx / imageRect.height()).coerceIn(0f, 1f)

    override fun onTouchEvent(event: MotionEvent): Boolean {
        if (imageRect.isEmpty) return false
        parent?.requestDisallowInterceptTouchEvent(true)
        when (event.actionMasked) {
            MotionEvent.ACTION_DOWN -> {
                downPoint.set(event.x, event.y)
                dragStartNorm.set(cropNorm)
                val c = pixelRect()
                mode = hitCorner(event.x, event.y, c)
                    ?: if (c.contains(event.x, event.y)) {
                        moveOffset.set(event.x - c.left, event.y - c.top)
                        DragMode.MOVE
                    } else {
                        DragMode.NEW
                    }
                if (mode == DragMode.NEW) {
                    val x = normX(event.x)
                    val y = normY(event.y)
                    cropNorm.set(x, y, x, y)
                }
                return true
            }

            MotionEvent.ACTION_MOVE -> {
                when (mode) {
                    DragMode.MOVE -> {
                        val dx = (event.x - downPoint.x) / imageRect.width()
                        val dy = (event.y - downPoint.y) / imageRect.height()
                        val s = dragStartNorm
                        val left = (s.left + dx).coerceIn(0f, 1f - s.width())
                        val top = (s.top + dy).coerceIn(0f, 1f - s.height())
                        cropNorm.set(left, top, left + s.width(), top + s.height())
                    }

                    DragMode.TL -> {
                        cropNorm.left = normX(event.x).coerceIn(0f, dragStartNorm.right - minNormX())
                        cropNorm.top = normY(event.y).coerceIn(0f, dragStartNorm.bottom - minNormY())
                    }

                    DragMode.TR -> {
                        cropNorm.right = normX(event.x).coerceIn(dragStartNorm.left + minNormX(), 1f)
                        cropNorm.top = normY(event.y).coerceIn(0f, dragStartNorm.bottom - minNormY())
                    }

                    DragMode.BL -> {
                        cropNorm.left = normX(event.x).coerceIn(0f, dragStartNorm.right - minNormX())
                        cropNorm.bottom = normY(event.y).coerceIn(dragStartNorm.top + minNormY(), 1f)
                    }

                    DragMode.BR -> {
                        cropNorm.right = normX(event.x).coerceIn(dragStartNorm.left + minNormX(), 1f)
                        cropNorm.bottom = normY(event.y).coerceIn(dragStartNorm.top + minNormY(), 1f)
                    }

                    DragMode.NEW -> {
                        val x0 = normX(downPoint.x)
                        val y0 = normY(downPoint.y)
                        val x1 = normX(event.x)
                        val y1 = normY(event.y)
                        cropNorm.set(min(x0, x1), min(y0, y1), max(x0, x1), max(y0, y1))
                    }

                    DragMode.NONE -> Unit
                }
                invalidate()
                notifyChanged()
                return true
            }

            MotionEvent.ACTION_UP, MotionEvent.ACTION_CANCEL -> {
                if (mode == DragMode.NEW &&
                    (cropNorm.width() < minNormX() || cropNorm.height() < minNormY())
                ) {
                    // 点击空白或拖动距离过小：恢复原选框
                    cropNorm.set(dragStartNorm)
                }
                mode = DragMode.NONE
                invalidate()
                notifyChanged()
                return true
            }
        }
        return super.onTouchEvent(event)
    }

    private fun hitCorner(x: Float, y: Float, c: RectF): DragMode? {
        val radius = maxOf(touchSlop * 2f, density * 18f)
        fun near(px: Float, py: Float): Boolean =
            (x - px) * (x - px) + (y - py) * (y - py) <= radius * radius
        return when {
            near(c.left, c.top) -> DragMode.TL
            near(c.right, c.top) -> DragMode.TR
            near(c.left, c.bottom) -> DragMode.BL
            near(c.right, c.bottom) -> DragMode.BR
            else -> null
        }
    }

    private fun notifyChanged() {
        onCropChanged?.invoke(RectF(cropNorm))
    }
}
