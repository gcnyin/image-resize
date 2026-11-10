package com.gcnyin.imagetool

import android.graphics.Bitmap
import android.graphics.Canvas
import android.view.ContextThemeWrapper
import android.view.LayoutInflater
import android.view.View
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.RuntimeEnvironment
import org.robolectric.annotation.Config
import org.robolectric.annotation.GraphicsMode
import java.io.File
import java.io.FileOutputStream

/**
 * 用 Robolectric 原生图形把主界面渲染成 PNG（浅色 / 深色各一张），
 * 输出到 app/build/reports/screenshots/，便于没有模拟器时人工核对配色。
 */
@RunWith(RobolectricTestRunner::class)
@GraphicsMode(GraphicsMode.Mode.NATIVE)
@Config(sdk = [34], qualifiers = "w393dp-h852dp-xxhdpi")
class ThemeScreenshotTest {

    @Test
    fun light() = render("light")

    @Test
    @Config(qualifiers = "+night")
    fun dark() = render("dark")

    private fun render(tag: String) {
        val ctx = ContextThemeWrapper(RuntimeEnvironment.getApplication(), R.style.Theme_ImageTool)
        val root = LayoutInflater.from(ctx).inflate(R.layout.activity_main, null)
        val width = 1080
        val height = 2340
        root.measure(
            View.MeasureSpec.makeMeasureSpec(width, View.MeasureSpec.EXACTLY),
            View.MeasureSpec.makeMeasureSpec(height, View.MeasureSpec.EXACTLY),
        )
        root.layout(0, 0, width, height)
        val bitmap = Bitmap.createBitmap(width, height, Bitmap.Config.ARGB_8888)
        root.draw(Canvas(bitmap))
        val out = File("build/reports/screenshots/$tag.png")
        out.parentFile?.mkdirs()
        FileOutputStream(out).use { bitmap.compress(Bitmap.CompressFormat.PNG, 100, it) }
        println("SCREENSHOT " + out.absolutePath)
    }
}
