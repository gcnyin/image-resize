package main

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"path/filepath"
	"testing"
)

// runTest 集成测试：临时目录生成真实 JPEG，验证退出码与源文件不被覆盖。
func TestRunExitCodes(t *testing.T) {
	dir := t.TempDir()

	// 合法 JPEG 源。
	img := image.NewRGBA(image.Rect(0, 0, 128, 128))
	for y := 0; y < 128; y++ {
		for x := 0; x < 128; x++ {
			img.Set(x, y, color.RGBA{uint8(x), uint8(y), 128, 255})
		}
	}
	jpgPath := filepath.Join(dir, "a.jpg")
	f, err := os.Create(jpgPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := jpeg.Encode(f, img, &jpeg.Options{Quality: 95}); err != nil {
		t.Fatal(err)
	}
	f.Close()
	srcBytes, _ := os.ReadFile(jpgPath)

	// 损坏 JPEG：合法魔数 + 截断数据。
	badPath := filepath.Join(dir, "bad.jpg")
	if err := os.WriteFile(badPath, append([]byte{0xFF, 0xD8, 0xFF, 0xE0}, make([]byte, 32)...), 0o644); err != nil {
		t.Fatal(err)
	}
	// 非图片。
	txtPath := filepath.Join(dir, "x.txt")
	if err := os.WriteFile(txtPath, []byte("nope"), 0o644); err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name string
		args []string
		want int
	}{
		{"无参数", nil, 2},
		{"-o缺参数", []string{"-o"}, 2},
		{"单个成功", []string{jpgPath}, 0},
		{"损坏JPEG", []string{badPath}, 1},
		{"非图片", []string{txtPath}, 1},
		{"成功加失败", []string{jpgPath, txtPath}, 1},
		{"-o输出目录", []string{"-o", filepath.Join(dir, "out"), jpgPath}, 0},
		// -o 指向源目录：不得覆盖源文件。
		{"-o源目录不覆盖源", []string{"-o", dir, jpgPath}, 0},
	}
	for _, c := range cases {
		var stdout, stderr bytes.Buffer
		oldOut, oldErr := os.Stdout, os.Stderr
		r, w, _ := os.Pipe()
		os.Stdout = w
		os.Stderr = w
		ch := make(chan string)
		go func() {
			var buf bytes.Buffer
			_, _ = buf.ReadFrom(r)
			ch <- buf.String()
		}()
		got := run(c.args)
		_ = w.Close()
		os.Stdout, os.Stderr = oldOut, oldErr
		<-ch
		stdout.Reset()
		stderr.Reset()
		if got != c.want {
			t.Errorf("%s: exit=%d want %d", c.name, got, c.want)
		}
	}

	// 源文件必须原样存在（未被任何一次运行覆盖）。
	after, err := os.ReadFile(jpgPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(srcBytes, after) {
		t.Errorf("源文件被修改: %d -> %d 字节", len(srcBytes), len(after))
	}
	// -o 源目录那次运行应产出 a_min.jpg 而不是覆盖 a.jpg。
	if _, err := os.Stat(filepath.Join(dir, "a_min.jpg")); err != nil {
		t.Errorf("-o 源目录未产出 _min 文件: %v", err)
	}
}
