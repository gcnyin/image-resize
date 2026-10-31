# 图片工具（image-resize）

一个轻量的 Android 图片编辑小工具，支持 **剪裁** 与 **按百分比缩放**。

- **剪裁**：在图片上拖动四角 / 整体拖动 / 空白处重新框选，应用后裁出选区像素。
- **缩放**：按 10%~100% 百分比缩小（如 2048px → 1024px），内容不变只降采样；
  采用"逐级 1/2 步进 + 双线性滤波"的高质量下采样策略，避免大倍率一步缩放产生锯齿。
- 编辑是循环式的：剪裁/缩放完成后可继续在上一步结果上操作，满意后一键保存。
- 保存写入系统相册的 `Pictures/图片工具/` 目录（MediaStore，无需存储权限）。
- 加载时自动按 EXIF 方向摆正照片。

## 压缩策略（恒定压缩率）

保存时**保持原格式**（按文件头魔数嗅探：JPEG / PNG / WebP / HEIF），并保证**体积与像素同比例收敛**：

- **目标字节数 = 源图每像素字节数 × 输出像素数**。即信息密度（压缩率）恒定：
  缩小 50% → 体积约为原图 1/4，绝不因重新编码而变大。
- **JPEG / 有损 WebP**：在该目标体积内二分搜索最大编码质量（下限 40，上限 97），
  优先保证不超出目标，其次尽量高画质。
- **PNG / 无损 WebP**：保持无损直接编码（无损没有质量旋钮，体积由内容决定；
  由于像素变为 1/4，通常同样显著缩小，见下方实测）。
- **HEIF/HEIC 源**：Android 不支持 HEIF 编码，回退为 JPEG（同样按恒定压缩率搜索质量）。
- 下采样采用"逐级 1/2 步进 + 双线性滤波"，避免大倍率一步缩放的锯齿。

实测（50% 缩放，`test_img/` 样例，桌面同算法参考值）：

| 样例 | 原体积 | 旧实现（固定 q92） | 新实现 |
| --- | --- | --- | --- |
| sample_q30.jpg（低质量 JPEG） | 40KB | 44KB（**+11% 反而变大**） | 16KB（39.9%） |
| sample_q85.jpg | 103KB | 44KB（44%） | 25KB（24.7%） |
| sample_lossy.webp | 44KB | （转 JPEG，更大） | 11KB（25.0%） |
| sample_lossless.webp | 418KB | （转 JPEG） | 128KB（30.6%，保持无损） |
| original.png | 717KB | — | 213KB（29.7%，保持无损） |

## 环境

| 组件 | 版本 |
| --- | --- |
| Gradle | 9.8.2（brew 安装，已生成 wrapper 可用 `./gradlew`） |
| Android Gradle Plugin | 9.5.0（内置 Kotlin 编译，无需 kotlin-android 插件） |
| compileSdk / targetSdk | 36 |
| minSdk | 29（Android 10+） |
| JDK | 17 |

## 构建

```sh
./gradlew assembleDebug          # 或：gradle :app:assembleDebug
./gradlew testDebugUnitTest      # 单元测试
```

产物：`app/build/outputs/apk/debug/app-debug.apk`（debug 签名，可直接安装）。

## 测试

```sh
./gradlew testDebugUnitTest
```

- `ImageMathTest`：尺寸换算与编码质量搜索的纯逻辑测试。
- `ImageEncoderPipelineTest`：Robolectric 跑真实 skia 解码/缩放/编码管线，
  用 `test_img/` 内的样例做端到端体积验证（50% 缩放后输出必须小于原图）。
  样例缺失时自动跳过；首次运行会经阿里云镜像下载 android-all 运行时。

## Release 构建

```sh
./gradlew assembleRelease
```

产物：`app/build/outputs/apk/release/app-release.apk`（正式签名，可直接安装）。

校验签名：

```sh
~/Library/Android/sdk/build-tools/36.0.0/apksigner verify --print-certs \
  app/build/outputs/apk/release/app-release.apk
```

### 签名说明

- 签名密钥为本仓库根目录的 `imagetool-release.jks`（alias `imagetool`，RSA 2048，有效期 10000 天），
  密码记录在 `keystore.properties`。两者均已 gitignore，**务必备份**——丢失后无法以同一签名发布更新。
- 签名配置在 `app/build.gradle.kts`：存在 `keystore.properties` 时自动签名 release；
  文件缺失时 release 不签名（不可直接安装，仅 debug 可装）。
- 更换机器 / 重建密钥：

  ```sh
  keytool -genkeypair -v -keystore imagetool-release.jks -alias imagetool \
    -keyalg RSA -keysize 2048 -validity 10000 \
    -storepass <你的密码> -keypass <你的密码> \
    -dname "CN=ImageTool, OU=Personal, O=gcnyin, C=CN"
  ```

  然后在 `keystore.properties` 中填入四项：`storeFile` / `storePassword` / `keyAlias` / `keyPassword`。

## 安装

```sh
adb install app/build/outputs/apk/release/app-release.apk   # release（推荐）
adb install app/build/outputs/apk/debug/app-debug.apk       # debug
```

或把 APK 传到手机上直接点击安装（需允许安装未知来源应用）。

## 使用

1. 「选择图片」→ 从相册/文件中选择一张图。
2. 底部切换 **剪裁 / 缩放** 模式：
   - 剪裁：在图上拖动选框，「应用剪裁」；
   - 缩放：拖动滑杆或点 25%/50%/75% 快捷按钮（顶部实时显示目标像素尺寸），「应用缩放」。
3. 可继续对结果再次剪裁/缩放，「保存到相册」写入 `Pictures/图片工具/`。

## 工程结构

```
app/src/main/kotlin/com/gcnyin/imagetool/
├── MainActivity.kt          # 界面与编辑流程编排
├── ImageLoader.kt           # Uri 解码 + EXIF 摆正 + 大图降采样（工作区上限 4096px）
├── ImageFormat.kt           # 文件头魔数识别格式（JPEG/PNG/WebP/HEIF），保存保持原格式
├── ImageOps.kt              # 逐级下采样缩放 / 矩形剪裁
├── ImageEncoder.kt          # 恒定压缩率编码（二分搜质量）+ 无损直通
├── ImageSaver.kt            # MediaStore 保存到相册
├── CropOverlayView.kt       # 剪裁选框覆盖层（归一化坐标，支持拖动/缩放/重新框选）
└── ImageMath.kt             # 纯尺寸/质量计算（被单元测试覆盖）
app/src/test/kotlin/com/gcnyin/imagetool/
├── ImageMathTest.kt             # 纯逻辑单元测试
└── ImageEncoderPipelineTest.kt  # Robolectric 端到端体积验证（读 test_img/）
test_img/                  # 端到端测试样例（PNG / JPEG / WebP）
```

## 网络 / 镜像配置

- 依赖仓库：阿里云镜像优先（`google` / `gradle-plugin` / `public`），配置在 `settings.gradle.kts`，
  `google()`、`mavenCentral()` 作为回退。
- Gradle 代理（`127.0.0.1:7890`）配置在用户级 `~/.gradle/gradle.properties`（`systemProp.http(s).proxyHost/Port`），
  同时该文件指定 `org.gradle.java.home` 为本机 JDK 17。
- `local.properties` 指向 SDK：`/Users/smith/Library/Android/sdk`（不入库）。
