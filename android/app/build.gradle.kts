import java.io.FileInputStream
import java.util.Properties

plugins {
    id("com.android.application")
}

// release 签名信息从根目录 keystore.properties 读取；文件不存在时 release 不签名。
val keystorePropsFile = rootProject.file("keystore.properties")
val keystoreProps = Properties().apply {
    if (keystorePropsFile.exists()) load(FileInputStream(keystorePropsFile))
}

android {
    namespace = "com.gcnyin.imagetool"
    compileSdk = 36

    defaultConfig {
        applicationId = "com.gcnyin.imagetool"
        minSdk = 29
        targetSdk = 36
        versionCode = 2
        versionName = "1.1"
    }

    signingConfigs {
        if (keystorePropsFile.exists()) {
            create("release") {
                storeFile = rootProject.file(keystoreProps.getProperty("storeFile"))
                storePassword = keystoreProps.getProperty("storePassword")
                keyAlias = keystoreProps.getProperty("keyAlias")
                keyPassword = keystoreProps.getProperty("keyPassword")
            }
        }
    }

    buildTypes {
        release {
            isMinifyEnabled = false
            proguardFiles(getDefaultProguardFile("proguard-android-optimize.txt"), "proguard-rules.pro")
            if (keystorePropsFile.exists()) {
                signingConfig = signingConfigs.getByName("release")
            }
        }
    }

    compileOptions {
        sourceCompatibility = JavaVersion.VERSION_17
        targetCompatibility = JavaVersion.VERSION_17
    }

    buildFeatures {
        viewBinding = true
    }

    testOptions {
        unitTests {
            isIncludeAndroidResources = true
        }
    }
}

dependencies {
    implementation("androidx.core:core-ktx:1.13.1")
    implementation("androidx.appcompat:appcompat:1.7.0")
    implementation("androidx.exifinterface:exifinterface:1.3.7")
    implementation("androidx.lifecycle:lifecycle-runtime-ktx:2.8.7")
    implementation("org.jetbrains.kotlinx:kotlinx-coroutines-android:1.8.1")

    testImplementation("junit:junit:4.13.2")
    testImplementation("org.robolectric:robolectric:4.17")
}

// Robolectric 运行时需下载 android-all 包：指向阿里云镜像并走代理
tasks.withType<Test>().configureEach {
    systemProperty("robolectric.dependency.repo.url", "https://maven.aliyun.com/repository/public")
    systemProperty("robolectric.dependency.repo.id", "aliyun")
    systemProperty("https.proxyHost", "127.0.0.1")
    systemProperty("https.proxyPort", "7890")
    systemProperty("http.proxyHost", "127.0.0.1")
    systemProperty("http.proxyPort", "7890")
    testLogging {
        events("passed", "failed", "skipped")
    }
}
