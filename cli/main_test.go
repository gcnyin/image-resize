package main

import (
	"testing"

	"image-resize/cli/internal/format"
)

func TestOutputPath(t *testing.T) {
	cases := []struct {
		path, outDir, want string
	}{
		{"/a/b/photo.jpg", "", "/a/b/photo_min.jpg"},
		{"/a/b/photo.jpg", "/out", "/out/photo.jpg"},
		{"photo.jpeg", "", "photo_min.jpeg"},
		{"photo", "", "photo_min"},
		// -o 指向源文件所在目录：退回 _min 后缀，防止覆盖源文件。
		{"/a/b/photo.jpg", "/a/b", "/a/b/photo_min.jpg"},
		{"/a/b/photo.jpg", "/a/b/", "/a/b/photo_min.jpg"},
	}
	for _, c := range cases {
		if got := outputPath(c.path, c.outDir); got != c.want {
			t.Errorf("outputPath(%q, %q) = %q, want %q", c.path, c.outDir, got, c.want)
		}
	}
}

func TestKindName(t *testing.T) {
	cases := []struct {
		in   format.Kind
		want string
	}{
		{format.Unknown, "未知格式"},
		{format.JPEG, "JPEG"},
		{format.PNG, "PNG"},
		{format.WebP, "WebP"},
		{format.HEIF, "HEIF"},
	}
	for _, c := range cases {
		if got := kindName(c.in); got != c.want {
			t.Errorf("kindName(%v) = %q, want %q", c.in, got, c.want)
		}
	}
}
