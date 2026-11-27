// imgcompr：命令行图片压缩工具（恒定压缩率，算法与 android/windows/macos 三端一致）。
//
// 用法：imgcompr [-o 输出目录] 文件1 [文件2 ...]
package main

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"strings"

	"image-resize/cli/internal/compress"
	"image-resize/cli/internal/format"
)

const headerBytes = 48

const usage = "用法: imgcompr [-o 输出目录(默认原目录)] 文件1 [文件2 ...]"

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	outDir := ""
	var files []string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-o":
			if i+1 >= len(args) {
				fmt.Fprintln(os.Stderr, "-o 需要一个目录参数")
				fmt.Fprintln(os.Stderr, usage)
				return 2
			}
			outDir = args[i+1]
			i++
		case "-h", "--help":
			fmt.Println(usage)
			return 0
		default:
			files = append(files, args[i])
		}
	}
	if len(files) == 0 {
		fmt.Fprintln(os.Stderr, usage)
		return 2
	}
	if outDir != "" {
		if err := os.MkdirAll(outDir, 0o755); err != nil {
			fmt.Fprintf(os.Stderr, "创建输出目录失败: %v\n", err)
			return 1
		}
	}

	failed := false
	for _, path := range files {
		if !processOne(path, outDir) {
			failed = true
		}
	}
	if failed {
		return 1
	}
	return 0
}

// processOne 处理单个文件，返回是否成功（无跳过、无失败）。
func processOne(path, outDir string) bool {
	src, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "读取失败: %s: %v\n", path, err)
		return false
	}
	if len(src) == 0 {
		fmt.Fprintf(os.Stderr, "失败: %s (空文件)\n", path)
		return false
	}

	kind := format.Detect(path, src[:min(len(src), headerBytes)])
	if kind != format.JPEG && kind != format.PNG {
		fmt.Fprintf(os.Stderr, "跳过: %s (%s 暂不支持, 仅 JPEG/PNG)\n", path, kindName(kind))
		return false
	}

	var img image.Image
	var decodeErr error
	switch kind {
	case format.JPEG:
		img, decodeErr = jpeg.Decode(bytes.NewReader(src))
	case format.PNG:
		img, decodeErr = png.Decode(bytes.NewReader(src))
	}
	if decodeErr != nil {
		fmt.Fprintf(os.Stderr, "解码失败: %s: %v\n", path, decodeErr)
		return false
	}

	res, err := compress.Compress(img, kind, int64(len(src)))
	if err != nil {
		fmt.Fprintf(os.Stderr, "压缩失败: %s: %v\n", path, err)
		return false
	}

	outPath := outputPath(path, outDir)
	if err := os.WriteFile(outPath, res.Data, 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "写入失败: %s: %v\n", outPath, err)
		return false
	}

	pct := 0.0
	if len(src) > 0 {
		pct = (float64(len(res.Data)) - float64(len(src))) * 100 / float64(len(src))
	}
	note := ""
	if len(res.Data) >= len(src) {
		note = " (未变小)"
	}
	if res.Quality > 0 {
		fmt.Printf("%s → %s (%s → %s, q=%d, %+.1f%%%s)\n",
			path, outPath, human(int64(len(src))), human(int64(len(res.Data))), res.Quality, pct, note)
	} else {
		fmt.Printf("%s → %s (%s → %s, 无损, %+.1f%%%s)\n",
			path, outPath, human(int64(len(src))), human(int64(len(res.Data))), pct, note)
	}
	return true
}

// outputPath 输出路径：-o 目录下用原名；同目录加 _min 后缀，不覆盖源文件。
// -o 指向源文件所在目录时同样退回 _min 后缀，保证源文件永不被覆盖。
// 输出已存在时直接覆盖（重跑幂等）。
func outputPath(path, outDir string) string {
	base := filepath.Base(path)
	if outDir != "" {
		out := filepath.Join(outDir, base)
		outAbs, outErr := filepath.Abs(out)
		srcAbs, srcErr := filepath.Abs(path)
		if outErr == nil && srcErr == nil && outAbs != srcAbs {
			return out
		}
	}
	ext := filepath.Ext(base)
	stem := strings.TrimSuffix(base, ext)
	return filepath.Join(filepath.Dir(path), stem+"_min"+ext)
}

func kindName(k format.Kind) string {
	switch k {
	case format.JPEG:
		return "JPEG"
	case format.PNG:
		return "PNG"
	case format.WebP:
		return "WebP"
	case format.HEIF:
		return "HEIF"
	default:
		return "未知格式"
	}
}

func human(n int64) string {
	switch {
	case n >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(n)/(1<<20))
	case n >= 1<<10:
		return fmt.Sprintf("%.1f KB", float64(n)/(1<<10))
	default:
		return fmt.Sprintf("%d B", n)
	}
}
