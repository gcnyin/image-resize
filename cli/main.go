// imgresize: 命令行图片缩放工具（不做裁剪，仅缩放，恒定压缩率对齐 android/windows/macos 三端）。
package main

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"image-resize/cli/internal/encode"
	"image-resize/cli/internal/exif"
	"image-resize/cli/internal/format"
	"image-resize/cli/internal/ops"
)

const headerBytes = 48

const usage = "用法: imgresize (-p 1~100 | -max 像素) [-o 输出目录(默认原目录)] 文件1 [文件2 ...]"

type scaleMode uint8

const (
	modeNone scaleMode = iota
	modePercent
	modeMaxDim
)

type cliConfig struct {
	mode     scaleMode
	value    int
	outDir   string
	files    []string
	showHelp bool
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	cfg, err := parseArgs(args)
	if err != nil {
		fmt.Fprintln(stderr, err)
		fmt.Fprintln(stderr, usage)
		return 2
	}
	if cfg.showHelp {
		fmt.Fprintln(stdout, usage)
		return 0
	}
	if cfg.outDir != "" {
		if err := os.MkdirAll(cfg.outDir, 0o755); err != nil {
			fmt.Fprintf(stderr, "创建输出目录失败: %v\n", err)
			return 1
		}
	}

	ok := true
	for _, path := range cfg.files {
		if !processOne(path, cfg, stdout, stderr) {
			ok = false
		}
	}
	if !ok {
		return 1
	}
	return 0
}

func parseArgs(args []string) (cliConfig, error) {
	var cfg cliConfig
	endOfFlags := false

	for i := 0; i < len(args); i++ {
		arg := args[i]
		if endOfFlags {
			cfg.files = append(cfg.files, arg)
			continue
		}
		if arg == "--" {
			endOfFlags = true
			continue
		}
		if arg == "-h" || arg == "--help" {
			cfg.showHelp = true
			return cfg, nil
		}
		if !strings.HasPrefix(arg, "-") || arg == "-" {
			cfg.files = append(cfg.files, arg)
			continue
		}

		name, val, hasVal := splitFlag(arg)
		nextVal := func() (string, error) {
			if hasVal {
				return val, nil
			}
			if i+1 >= len(args) {
				return "", fmt.Errorf("%s 需要一个参数", name)
			}
			i++
			return args[i], nil
		}

		switch name {
		case "-p", "--percent":
			if cfg.mode != modeNone {
				return cfg, fmt.Errorf("-p 与 -max 不能重复或同时使用")
			}
			raw, err := nextVal()
			if err != nil {
				return cfg, err
			}
			n, err := strconv.Atoi(raw)
			if err != nil || n < 1 || n > 100 {
				return cfg, fmt.Errorf("百分比须为 1~100 的整数: %q", raw)
			}
			cfg.mode = modePercent
			cfg.value = n

		case "-max", "--max":
			if cfg.mode != modeNone {
				return cfg, fmt.Errorf("-p 与 -max 不能重复或同时使用")
			}
			raw, err := nextVal()
			if err != nil {
				return cfg, err
			}
			n, err := strconv.Atoi(raw)
			if err != nil || n < 1 {
				return cfg, fmt.Errorf("最大边长须为大于 0 的整数: %q", raw)
			}
			cfg.mode = modeMaxDim
			cfg.value = n

		case "-o", "--output":
			raw, err := nextVal()
			if err != nil {
				return cfg, err
			}
			if raw == "" {
				return cfg, fmt.Errorf("输出目录不能为空")
			}
			cfg.outDir = raw

		default:
			return cfg, fmt.Errorf("未知参数: %s", name)
		}
	}

	if cfg.mode == modeNone {
		return cfg, fmt.Errorf("必须指定 -p 百分比或 -max 最大边长")
	}
	if len(cfg.files) == 0 {
		return cfg, fmt.Errorf("请至少提供一个输入文件")
	}
	return cfg, nil
}

func splitFlag(arg string) (name, val string, hasVal bool) {
	if idx := strings.IndexByte(arg, '='); idx >= 0 {
		return arg[:idx], arg[idx+1:], true
	}
	return arg, "", false
}

func processOne(path string, cfg cliConfig, stdout, stderr io.Writer) bool {
	srcBytes, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(stderr, "读取失败: %s: %v\n", path, err)
		return false
	}
	if len(srcBytes) == 0 {
		fmt.Fprintf(stderr, "失败: %s (空文件)\n", path)
		return false
	}

	kind := format.Detect(path, srcBytes[:min(len(srcBytes), headerBytes)])
	if kind != format.JPEG && kind != format.PNG {
		fmt.Fprintf(stderr, "跳过: %s (%s 暂不支持, 仅 JPEG/PNG)\n", path, kind)
		return false
	}

	img, err := decodeImage(srcBytes, kind)
	if err != nil {
		fmt.Fprintf(stderr, "解码失败: %s: %v\n", path, err)
		return false
	}
	if kind == format.JPEG {
		img = exif.Apply(img, exif.ReadOrientation(srcBytes))
	}

	srcBounds := img.Bounds()
	srcW, srcH := srcBounds.Dx(), srcBounds.Dy()
	sourcePixels := int64(srcW) * int64(srcH)

	var scaled image.Image
	switch cfg.mode {
	case modePercent:
		scaled, err = ops.ScaleByPercent(img, cfg.value)
	case modeMaxDim:
		scaled, err = ops.ScaleToMaxDim(img, cfg.value)
	}
	if err != nil {
		fmt.Fprintf(stderr, "缩放失败: %s: %v\n", path, err)
		return false
	}

	res, err := encode.Encode(scaled, kind, int64(len(srcBytes)), sourcePixels)
	if err != nil {
		fmt.Fprintf(stderr, "编码失败: %s: %v\n", path, err)
		return false
	}

	outPath := outputPath(path, cfg, res.Ext)
	if err := writeFileAtomic(outPath, res.Data); err != nil {
		fmt.Fprintf(stderr, "写入失败: %s: %v\n", outPath, err)
		return false
	}

	outBounds := scaled.Bounds()
	pct := (float64(len(res.Data)) - float64(len(srcBytes))) * 100 / float64(len(srcBytes))
	if res.Quality > 0 {
		fmt.Fprintf(stdout, "%s -> %s (%dx%d -> %dx%d, %s -> %s, q=%d, %+.1f%%)\n",
			path, outPath, srcW, srcH, outBounds.Dx(), outBounds.Dy(),
			humanBytes(int64(len(srcBytes))), humanBytes(int64(len(res.Data))), res.Quality, pct)
	} else {
		fmt.Fprintf(stdout, "%s -> %s (%dx%d -> %dx%d, %s -> %s, 无损, %+.1f%%)\n",
			path, outPath, srcW, srcH, outBounds.Dx(), outBounds.Dy(),
			humanBytes(int64(len(srcBytes))), humanBytes(int64(len(res.Data))), pct)
	}
	return true
}

func decodeImage(data []byte, kind format.Kind) (image.Image, error) {
	switch kind {
	case format.JPEG:
		return jpeg.Decode(bytes.NewReader(data))
	case format.PNG:
		return png.Decode(bytes.NewReader(data))
	default:
		return nil, fmt.Errorf("不支持的格式: %s", kind)
	}
}

func outputPath(path string, cfg cliConfig, ext string) string {
	base := filepath.Base(path)
	if cfg.outDir != "" {
		candidate := filepath.Join(cfg.outDir, base)
		candAbs, candErr := filepath.Abs(candidate)
		srcAbs, srcErr := filepath.Abs(path)
		if candErr == nil && srcErr == nil && candAbs != srcAbs {
			return candidate
		}
	}
	fileExt := filepath.Ext(base)
	stem := strings.TrimSuffix(base, fileExt)
	if fileExt == "" {
		fileExt = "." + ext
	}
	var suffix string
	switch cfg.mode {
	case modePercent:
		suffix = fmt.Sprintf("_%dpct", cfg.value)
	case modeMaxDim:
		suffix = fmt.Sprintf("_max%d", cfg.value)
	}
	return filepath.Join(filepath.Dir(path), stem+suffix+fileExt)
}

func writeFileAtomic(outPath string, data []byte) error {
	dir := filepath.Dir(outPath)
	tmp, err := os.CreateTemp(dir, ".imgresize-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(tmpName)
		}
	}()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	_ = tmp.Chmod(0o644)
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpName, outPath); err != nil {
		return err
	}
	cleanup = false
	return nil
}

func humanBytes(n int64) string {
	switch {
	case n >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(n)/(1<<20))
	case n >= 1<<10:
		return fmt.Sprintf("%.1f KB", float64(n)/(1<<10))
	default:
		return fmt.Sprintf("%d B", n)
	}
}
