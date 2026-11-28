package main

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunArgValidationAndExitCodes(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		wantCode int
	}{
		{"no_args", nil, 2},
		{"missing_scale_mode", []string{"photo.jpg"}, 2},
		{"missing_files", []string{"-p", "50"}, 2},
		{"conflict_p_and_max", []string{"-p", "50", "-max", "800", "photo.jpg"}, 2},
		{"percent_zero", []string{"-p", "0", "photo.jpg"}, 2},
		{"percent_over_100", []string{"-p", "101", "photo.jpg"}, 2},
		{"max_zero", []string{"-max", "0", "photo.jpg"}, 2},
		{"unknown_flag", []string{"--crop", "10", "-p", "50", "photo.jpg"}, 2},
		{"help_short", []string{"-h"}, 0},
		{"help_long", []string{"--help"}, 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			if got := run(tc.args, &stdout, &stderr); got != tc.wantCode {
				t.Fatalf("run(%v) = %d, want %d; stderr=%s", tc.args, got, tc.wantCode, stderr.String())
			}
		})
	}
}

func TestParseArgsFlexibleOrderAndDoubleDash(t *testing.T) {
	cfg, err := parseArgs([]string{"a.jpg", "--percent=50", "-o", "out", "--", "-weird.png"})
	if err != nil {
		t.Fatalf("parseArgs failed: %v", err)
	}
	if cfg.mode != modePercent || cfg.value != 50 || cfg.outDir != "out" {
		t.Fatalf("unexpected cfg: %+v", cfg)
	}
	if len(cfg.files) != 2 || cfg.files[0] != "a.jpg" || cfg.files[1] != "-weird.png" {
		t.Fatalf("unexpected files: %v", cfg.files)
	}
}

func TestOutputPathAndNoSourceOverwrite(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "photo.jpg")

	pSame := outputPath(src, cliConfig{mode: modePercent, value: 50}, "jpg")
	if pSame != filepath.Join(dir, "photo_50pct.jpg") {
		t.Fatalf("same dir = %q", pSame)
	}

	pSameOut := outputPath(src, cliConfig{mode: modeMaxDim, value: 1200, outDir: dir}, "jpg")
	if pSameOut != filepath.Join(dir, "photo_max1200.jpg") {
		t.Fatalf("-o pointing to source dir must not overwrite source, got %q", pSameOut)
	}

	other := filepath.Join(dir, "out")
	pOther := outputPath(src, cliConfig{mode: modePercent, value: 50, outDir: other}, "jpg")
	if pOther != filepath.Join(other, "photo.jpg") {
		t.Fatalf("different -o dir = %q", pOther)
	}

	noExt := outputPath(filepath.Join(dir, "raw"), cliConfig{mode: modePercent, value: 50}, "png")
	if noExt != filepath.Join(dir, "raw_50pct.png") {
		t.Fatalf("no ext = %q", noExt)
	}
}

func TestEndToEndSyntheticJPEGWithExifAndIdempotency(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "rotated.jpg")

	// 200x100 source encoded with EXIF orientation 6 (90 CW) -> upright 100x200.
	img := image.NewRGBA(image.Rect(0, 0, 200, 100))
	for y := 0; y < 100; y++ {
		for x := 0; x < 200; x++ {
			img.SetRGBA(x, y, color.RGBA{R: uint8(x), G: uint8(y * 2), B: 128, A: 255})
		}
	}
	var rawJPEG bytes.Buffer
	if err := jpeg.Encode(&rawJPEG, img, &jpeg.Options{Quality: 90}); err != nil {
		t.Fatalf("jpeg encode: %v", err)
	}
	withExif := injectExifOrientation(rawJPEG.Bytes(), 6)
	if err := os.WriteFile(srcPath, withExif, 0o644); err != nil {
		t.Fatalf("write src: %v", err)
	}

	var stdout, stderr bytes.Buffer
	if code := run([]string{"-p", "50", srcPath}, &stdout, &stderr); code != 0 {
		t.Fatalf("first run exit = %d, stderr=%s", code, stderr.String())
	}

	outPath := filepath.Join(dir, "rotated_50pct.jpg")
	outBytes, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	decoded, err := jpeg.Decode(bytes.NewReader(outBytes))
	if err != nil {
		t.Fatalf("decode output: %v", err)
	}
	// Upright 100x200 at 50% -> 50x100.
	if decoded.Bounds().Dx() != 50 || decoded.Bounds().Dy() != 100 {
		t.Fatalf("output size = %dx%d, want 50x100", decoded.Bounds().Dx(), decoded.Bounds().Dy())
	}

	// Second run in the same directory must succeed idempotently with no temp leftovers.
	stdout.Reset()
	stderr.Reset()
	if code := run([]string{"-p", "50", srcPath}, &stdout, &stderr); code != 0 {
		t.Fatalf("second run exit = %d, stderr=%s", code, stderr.String())
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("readdir: %v", err)
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".imgresize-") {
			t.Fatalf("stale temp file left behind: %s", e.Name())
		}
	}
}

func TestEndToEndRealRepoSamplesAndErrorBatch(t *testing.T) {
	samplesDir := filepath.Join("..", "windows", "samples")
	outDir := t.TempDir()

	realSamples := []struct {
		file string
		kind string
	}{
		{"sample_q85.jpg", "jpeg"},
		{"sample_q30.jpg", "jpeg"},
		{"original.png", "png"},
	}

	var args []string
	args = append(args, "-p", "50", "-o", outDir)
	for _, s := range realSamples {
		args = append(args, filepath.Join(samplesDir, s.file))
	}

	var stdout, stderr bytes.Buffer
	if code := run(args, &stdout, &stderr); code != 0 {
		t.Fatalf("real samples run exit = %d, stderr=%s", code, stderr.String())
	}

	for _, s := range realSamples {
		srcPath := filepath.Join(samplesDir, s.file)
		outPath := filepath.Join(outDir, s.file)

		srcInfo, err := os.Stat(srcPath)
		if err != nil {
			t.Fatalf("stat src %s: %v", s.file, err)
		}
		outBytes, err := os.ReadFile(outPath)
		if err != nil {
			t.Fatalf("read out %s: %v", s.file, err)
		}
		if int64(len(outBytes)) >= srcInfo.Size() {
			t.Fatalf("%s 50%% output (%d B) did not shrink below source (%d B)",
				s.file, len(outBytes), srcInfo.Size())
		}

		srcFile, err := os.Open(srcPath)
		if err != nil {
			t.Fatalf("open src %s: %v", s.file, err)
		}
		srcCfg, _, err := image.DecodeConfig(srcFile)
		_ = srcFile.Close()
		if err != nil {
			t.Fatalf("decodeConfig src %s: %v", s.file, err)
		}

		var decoded image.Image
		if s.kind == "jpeg" {
			decoded, err = jpeg.Decode(bytes.NewReader(outBytes))
		} else {
			decoded, err = png.Decode(bytes.NewReader(outBytes))
		}
		if err != nil {
			t.Fatalf("decode out %s: %v", s.file, err)
		}
		wantW := max(1, int(math.RoundToEven(float64(srcCfg.Width)*0.5)))
		wantH := max(1, int(math.RoundToEven(float64(srcCfg.Height)*0.5)))
		if decoded.Bounds().Dx() != wantW || decoded.Bounds().Dy() != wantH {
			t.Fatalf("%s out dims = %dx%d, want %dx%d",
				s.file, decoded.Bounds().Dx(), decoded.Bounds().Dy(), wantW, wantH)
		}
	}

	// Mixed batch: unsupported WebP + empty file + corrupt JPEG + valid sample.
	badDir := t.TempDir()
	emptyFile := filepath.Join(badDir, "empty.jpg")
	corruptFile := filepath.Join(badDir, "corrupt.jpg")
	if err := os.WriteFile(emptyFile, nil, 0o644); err != nil {
		t.Fatalf("write empty: %v", err)
	}
	if err := os.WriteFile(corruptFile, []byte{0xFF, 0xD8, 0xFF, 0x00, 0x01}, 0o644); err != nil {
		t.Fatalf("write corrupt: %v", err)
	}

	mixedOut := t.TempDir()
	stdout.Reset()
	stderr.Reset()
	code := run([]string{
		"-max", "256",
		"-o", mixedOut,
		filepath.Join(samplesDir, "sample_lossy.webp"),
		emptyFile,
		corruptFile,
		filepath.Join(samplesDir, "sample_q85.jpg"),
	}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("mixed batch exit = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "WebP") {
		t.Fatalf("stderr should mention WebP skip, got: %s", stderr.String())
	}
	validOut, err := os.ReadFile(filepath.Join(mixedOut, "sample_q85.jpg"))
	if err != nil {
		t.Fatalf("valid file in mixed batch should still succeed: %v", err)
	}
	decoded, err := jpeg.Decode(bytes.NewReader(validOut))
	if err != nil {
		t.Fatalf("decode mixed valid output: %v", err)
	}
	if max(decoded.Bounds().Dx(), decoded.Bounds().Dy()) != 256 {
		t.Fatalf("-max 256 longest side = %d, want 256", max(decoded.Bounds().Dx(), decoded.Bounds().Dy()))
	}
}

func injectExifOrientation(jpegBytes []byte, orientation uint16) []byte {
	var tiff bytes.Buffer
	tiff.WriteString("MM")
	var u16 [2]byte
	var u32 [4]byte
	binary.BigEndian.PutUint16(u16[:], 42)
	tiff.Write(u16[:])
	binary.BigEndian.PutUint32(u32[:], 8)
	tiff.Write(u32[:])
	binary.BigEndian.PutUint16(u16[:], 1)
	tiff.Write(u16[:])
	binary.BigEndian.PutUint16(u16[:], 0x0112)
	tiff.Write(u16[:])
	binary.BigEndian.PutUint16(u16[:], 3)
	tiff.Write(u16[:])
	binary.BigEndian.PutUint32(u32[:], 1)
	tiff.Write(u32[:])
	binary.BigEndian.PutUint16(u16[:], orientation)
	tiff.Write(u16[:])
	tiff.Write([]byte{0, 0})
	binary.BigEndian.PutUint32(u32[:], 0)
	tiff.Write(u32[:])

	payload := append([]byte("Exif\x00\x00"), tiff.Bytes()...)
	var out bytes.Buffer
	out.Write(jpegBytes[:2])
	out.Write([]byte{0xFF, 0xE1})
	binary.BigEndian.PutUint16(u16[:], uint16(len(payload)+2))
	out.Write(u16[:])
	out.Write(payload)
	out.Write(jpegBytes[2:])
	return out.Bytes()
}
