package com.gcnyin.imagetool

import android.content.Context
import android.graphics.Color
import android.graphics.drawable.ColorDrawable
import android.view.ContextThemeWrapper
import android.view.LayoutInflater
import android.view.View
import android.widget.TextView
import androidx.annotation.ColorRes
import androidx.core.content.ContextCompat
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.RuntimeEnvironment
import org.robolectric.annotation.Config
import kotlin.math.max
import kotlin.math.min
import kotlin.math.pow

/**
 * 深浅色适配的回归测试：
 * 同一套布局在 notnight / night 两种资源限定下必须解析出不同的配色，
 * 且正文、次要文字、提示文字相对各自背景的对比度不低于 WCAG AA（4.5:1）。
 */
@RunWith(RobolectricTestRunner::class)
@Config(sdk = [34])
class ThemeColorsTest {

    private fun app(): Context = RuntimeEnvironment.getApplication()

    private fun color(context: Context, @ColorRes id: Int): Int =
        ContextCompat.getColor(context, id)

    private fun luminance(argb: Int): Double {
        fun channel(value: Int): Double {
            val c = value / 255.0
            return if (c <= 0.03928) c / 12.92 else ((c + 0.055) / 1.055).pow(2.4)
        }
        return 0.2126 * channel(Color.red(argb)) +
            0.7152 * channel(Color.green(argb)) +
            0.0722 * channel(Color.blue(argb))
    }

    private fun contrast(foreground: Int, background: Int): Double {
        val a = luminance(foreground)
        val b = luminance(background)
        return (max(a, b) + 0.05) / (min(a, b) + 0.05)
    }

    @Test
    @Config(qualifiers = "+notnight")
    fun lightPalette_isLightAndReadable() {
        val ctx = app()
        val panel = color(ctx, R.color.panel_bg)
        val canvas = color(ctx, R.color.canvas_bg)

        assertTrue("浅色模式面板应为亮色", luminance(panel) > 0.8)
        assertTrue("浅色模式画布应为亮色", luminance(canvas) > 0.7)
        assertTrue("正文对比度不足", contrast(color(ctx, R.color.text_primary), panel) >= 7.0)
        assertTrue("次要文字对比度不足", contrast(color(ctx, R.color.text_secondary), panel) >= 4.5)
        assertTrue("提示文字对比度不足", contrast(color(ctx, R.color.text_hint), canvas) >= 4.5)
    }

    @Test
    @Config(qualifiers = "+night")
    fun darkPalette_isDarkAndReadable() {
        val ctx = app()
        val panel = color(ctx, R.color.panel_bg)
        val canvas = color(ctx, R.color.canvas_bg)

        assertTrue("深色模式面板应为暗色", luminance(panel) < 0.05)
        assertTrue("深色模式画布应为暗色", luminance(canvas) < 0.05)
        assertTrue("正文对比度不足", contrast(color(ctx, R.color.text_primary), panel) >= 7.0)
        assertTrue("次要文字对比度不足", contrast(color(ctx, R.color.text_secondary), panel) >= 4.5)
        assertTrue("提示文字对比度不足", contrast(color(ctx, R.color.text_hint), panel) >= 4.5)
    }

    @Test
    @Config(qualifiers = "+notnight")
    fun layout_usesLightPalette() {
        val ctx = ContextThemeWrapper(app(), R.style.Theme_ImageTool)
        val root = LayoutInflater.from(ctx).inflate(R.layout.activity_main, null)

        assertEquals(color(ctx, R.color.canvas_bg), (root.background as ColorDrawable).color)
        val panel = root.findViewById<View>(R.id.panel)
        assertEquals(color(ctx, R.color.panel_bg), (panel.background as ColorDrawable).color)
        assertEquals(
            color(ctx, R.color.text_primary),
            root.findViewById<TextView>(R.id.tvPercent).currentTextColor,
        )
        assertEquals(
            color(ctx, R.color.text_secondary),
            root.findViewById<TextView>(R.id.tvInfo).currentTextColor,
        )
        assertTrue("浅色模式状态栏应使用深色图标", lightStatusBarIcons(ctx))
    }

    @Test
    @Config(qualifiers = "+night")
    fun layout_usesDarkPalette() {
        val ctx = ContextThemeWrapper(app(), R.style.Theme_ImageTool)
        val root = LayoutInflater.from(ctx).inflate(R.layout.activity_main, null)

        assertEquals(color(ctx, R.color.canvas_bg), (root.background as ColorDrawable).color)
        val panel = root.findViewById<View>(R.id.panel)
        assertEquals(color(ctx, R.color.panel_bg), (panel.background as ColorDrawable).color)
        assertEquals(
            color(ctx, R.color.text_primary),
            root.findViewById<TextView>(R.id.tvPercent).currentTextColor,
        )
        assertEquals(
            color(ctx, R.color.text_secondary),
            root.findViewById<TextView>(R.id.tvInfo).currentTextColor,
        )
        assertFalse("深色模式状态栏应使用浅色图标", lightStatusBarIcons(ctx))
    }

    private fun lightStatusBarIcons(context: Context): Boolean {
        val values = context.theme.obtainStyledAttributes(
            intArrayOf(android.R.attr.windowLightStatusBar),
        )
        return try {
            values.getBoolean(0, false)
        } finally {
            values.recycle()
        }
    }
}
