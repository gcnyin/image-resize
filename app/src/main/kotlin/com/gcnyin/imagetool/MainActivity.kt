package com.gcnyin.imagetool

import android.graphics.Bitmap
import android.graphics.RectF
import android.net.Uri
import android.os.Bundle
import android.view.View
import android.widget.SeekBar
import android.widget.Toast
import androidx.activity.result.contract.ActivityResultContracts
import androidx.appcompat.app.AppCompatActivity
import androidx.lifecycle.lifecycleScope
import com.gcnyin.imagetool.databinding.ActivityMainBinding
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext

class MainActivity : AppCompatActivity() {

    private lateinit var binding: ActivityMainBinding

    /** 当前编辑中的位图（加载 → 剪裁/缩放循环替换）。 */
    private var current: Bitmap? = null

    /** 本次加载的源图片信息。 */
    private var source: LoadedImage? = null

    private var busy = false

    private val pickImage =
        registerForActivityResult(ActivityResultContracts.GetContent()) { uri: Uri? ->
            if (uri != null) loadImage(uri)
        }

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        binding = ActivityMainBinding.inflate(layoutInflater)
        setContentView(binding.root)

        binding.btnPick.setOnClickListener { pickImage.launch("image/*") }
        binding.btnSave.setOnClickListener { save() }
        binding.btnApplyCrop.setOnClickListener { applyCrop() }
        binding.btnApplyResize.setOnClickListener { applyResize() }
        binding.btnFull.setOnClickListener {
            binding.cropOverlay.resetToFull()
            updateInfo()
        }
        binding.btn25.setOnClickListener { setPercent(25) }
        binding.btn50.setOnClickListener { setPercent(50) }
        binding.btn75.setOnClickListener { setPercent(75) }
        binding.modeGroup.setOnCheckedChangeListener { _, _ -> updateModeUi() }
        binding.seekPercent.setOnSeekBarChangeListener(object : SeekBar.OnSeekBarChangeListener {
            override fun onProgressChanged(seekBar: SeekBar, progress: Int, fromUser: Boolean) {
                binding.tvPercent.text = "$progress%"
                updateInfo()
            }

            override fun onStartTrackingTouch(seekBar: SeekBar) = Unit

            override fun onStopTrackingTouch(seekBar: SeekBar) = Unit
        })
        binding.ivImage.addOnLayoutChangeListener { _, _, _, _, _, _, _, _, _ -> syncImageRect() }
        binding.cropOverlay.onCropChanged = { updateInfo() }

        updateModeUi()
    }

    // ---------------------------------------------------------------- 加载

    private fun loadImage(uri: Uri) {
        if (busy) return
        setBusyState(true)
        binding.tvInfo.text = "加载中…"
        lifecycleScope.launch {
            try {
                val img = withContext(Dispatchers.Default) {
                    ImageLoader.load(applicationContext, uri)
                }
                source = img
                replaceBitmap(img.bitmap)
                binding.cropOverlay.resetToFull()
                syncImageRect()
            } catch (t: Throwable) {
                Toast.makeText(this@MainActivity, "加载失败：${t.message}", Toast.LENGTH_LONG).show()
            } finally {
                setBusyState(false)
                updateModeUi()
            }
        }
    }

    // ---------------------------------------------------------------- 剪裁 / 缩放

    private fun applyCrop() {
        val bmp = current ?: return
        val norm = binding.cropOverlay.cropRectNorm()
        if (norm.width() >= 0.999f && norm.height() >= 0.999f) {
            Toast.makeText(this, "选框已覆盖全图，无需剪裁", Toast.LENGTH_SHORT).show()
            return
        }
        runImageOp("剪裁") { ImageOps.crop(bmp, norm) }
    }

    private fun applyResize() {
        val bmp = current ?: return
        val percent = binding.seekPercent.progress
        if (percent >= 100) {
            Toast.makeText(this, "比例已是 100%，无需缩放", Toast.LENGTH_SHORT).show()
            return
        }
        runImageOp("缩放") { ImageOps.scaleByPercent(bmp, percent) }
    }

    private fun runImageOp(name: String, op: (Bitmap) -> Bitmap) {
        val bmp = current ?: return
        if (busy) return
        setBusyState(true)
        lifecycleScope.launch {
            try {
                val next = withContext(Dispatchers.Default) { op(bmp) }
                replaceBitmap(next)
                syncImageRect()
                binding.cropOverlay.resetToFull()
                Toast.makeText(
                    this@MainActivity,
                    "$name 完成：${next.width}×${next.height}",
                    Toast.LENGTH_SHORT,
                ).show()
            } catch (t: Throwable) {
                Toast.makeText(this@MainActivity, "$name 失败：${t.message}", Toast.LENGTH_LONG).show()
            } finally {
                setBusyState(false)
                updateModeUi()
            }
        }
    }

    // ---------------------------------------------------------------- 保存

    private fun save() {
        val bmp = current ?: return
        val img = source ?: return
        if (busy) return
        setBusyState(true)
        lifecycleScope.launch {
            try {
                // 恒定压缩率编码：与源图保持相同的每像素字节
                val enc = withContext(Dispatchers.Default) {
                    ImageEncoder.encode(
                        bmp,
                        img.format,
                        img.sourceFileBytes,
                        img.sourceWidth.toLong() * img.sourceHeight,
                        img.webpLossless,
                    )
                }
                val path = withContext(Dispatchers.IO) {
                    ImageSaver.save(applicationContext, enc.bytes, enc.mime)
                }
                val kb = (enc.bytes.size + 1023) / 1024
                Toast.makeText(
                    this@MainActivity,
                    "已保存到相册：$path（$kb KB）",
                    Toast.LENGTH_LONG,
                ).show()
            } catch (t: Throwable) {
                Toast.makeText(this@MainActivity, "保存失败：${t.message}", Toast.LENGTH_LONG).show()
            } finally {
                setBusyState(false)
            }
        }
    }

    // ---------------------------------------------------------------- UI 状态

    private fun replaceBitmap(next: Bitmap) {
        binding.ivImage.setImageDrawable(null)
        current?.let { old -> if (old !== next && !old.isRecycled) old.recycle() }
        current = next
        binding.ivImage.setImageBitmap(next)
    }

    /** 计算位图在 ImageView 中的实际显示区域（fitCenter），同步给剪裁覆盖层。 */
    private fun syncImageRect() {
        val bmp = current
        if (bmp == null) {
            binding.cropOverlay.setImageRect(null)
            return
        }
        val iv = binding.ivImage
        val dw = (iv.width - iv.paddingLeft - iv.paddingRight).toFloat()
        val dh = (iv.height - iv.paddingTop - iv.paddingBottom).toFloat()
        if (dw <= 0f || dh <= 0f) return
        val scale = minOf(dw / bmp.width, dh / bmp.height)
        val w = bmp.width * scale
        val h = bmp.height * scale
        val left = iv.paddingLeft + (dw - w) / 2f
        val top = iv.paddingTop + (dh - h) / 2f
        binding.cropOverlay.setImageRect(RectF(left, top, left + w, top + h))
    }

    private fun setPercent(percent: Int) {
        binding.seekPercent.progress = percent
        binding.tvPercent.text = "$percent%"
        updateInfo()
    }

    private fun updateModeUi() {
        val has = current != null
        val cropMode = binding.rbCrop.isChecked
        binding.tvEmpty.visibility = if (has) View.GONE else View.VISIBLE
        binding.cropOverlay.visibility = if (has && cropMode) View.VISIBLE else View.GONE
        binding.cropControls.visibility = if (has && cropMode) View.VISIBLE else View.GONE
        binding.resizeControls.visibility = if (has && !cropMode) View.VISIBLE else View.GONE
        binding.btnSave.isEnabled = has && !busy
        updateInfo()
    }

    private fun setBusyState(newBusy: Boolean) {
        busy = newBusy
        binding.btnPick.isEnabled = !newBusy
        binding.btnFull.isEnabled = !newBusy
        binding.btnApplyCrop.isEnabled = !newBusy
        binding.btnApplyResize.isEnabled = !newBusy
        binding.btn25.isEnabled = !newBusy
        binding.btn50.isEnabled = !newBusy
        binding.btn75.isEnabled = !newBusy
        binding.seekPercent.isEnabled = !newBusy
        binding.rbCrop.isEnabled = !newBusy
        binding.rbResize.isEnabled = !newBusy
        binding.btnSave.isEnabled = current != null && !newBusy
        updateInfo()
    }

    private fun updateInfo() {
        val img = source
        val bmp = current
        if (bmp == null || img == null) {
            binding.tvInfo.text = getString(R.string.no_image)
            return
        }
        val label = formatLabel(img)
        if (binding.rbCrop.isChecked) {
            val n = binding.cropOverlay.cropRectNorm()
            val cw = (n.width() * bmp.width).toInt().coerceAtLeast(1)
            val ch = (n.height() * bmp.height).toInt().coerceAtLeast(1)
            binding.tvInfo.text =
                "当前 ${bmp.width}×${bmp.height} · 剪裁区域 ${cw}×${ch} · 原图 ${img.sourceWidth}×${img.sourceHeight} $label"
        } else {
            val p = binding.seekPercent.progress
            val tw = ImageMath.targetSize(bmp.width, p)
            val th = ImageMath.targetSize(bmp.height, p)
            binding.tvInfo.text =
                "比例 $p%：${bmp.width}×${bmp.height} → $tw×$th · 原图 ${img.sourceWidth}×${img.sourceHeight} $label"
        }
    }

    private fun formatLabel(img: LoadedImage): String = when (img.format) {
        ImageFormat.Kind.JPEG -> "JPEG"
        ImageFormat.Kind.PNG -> "PNG"
        ImageFormat.Kind.WEBP -> if (img.webpLossless) "WebP(无损)" else "WebP"
        ImageFormat.Kind.HEIF -> "HEIF→JPEG"
        ImageFormat.Kind.UNKNOWN -> "JPEG"
    }
}
