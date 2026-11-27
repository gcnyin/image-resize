package format

import "testing"

func TestSniff(t *testing.T) {
	cases := []struct {
		name   string
		header []byte
		want   Kind
	}{
		{"jpeg", []byte{0xFF, 0xD8, 0xFF, 0xE0}, JPEG},
		{"png", []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A}, PNG},
		{"webp", append([]byte("RIFF"), append(make([]byte, 4), 'W', 'E', 'B', 'P')...), WebP},
		{"empty", nil, Unknown},
		{"text", []byte("hello world, definitely not an image"), Unknown},
	}
	for _, c := range cases {
		if got := Sniff(c.header); got != c.want {
			t.Errorf("%s: got %v want %v", c.name, got, c.want)
		}
	}
}

func TestSniffHeifBrands(t *testing.T) {
	for _, brand := range []string{"heic", "heix", "hevc", "hevx", "heim", "heis", "hevm", "hevs", "mif1", "msf1"} {
		header := append([]byte{0, 0, 0, 24, 'f', 't', 'y', 'p'}, brand...)
		if got := Sniff(header); got != HEIF {
			t.Errorf("brand %s: got %v want HEIF", brand, got)
		}
	}
	// 非 HEIF 的 ftyp 品牌（如 isom/mp4）不算 HEIF。
	if got := Sniff([]byte{0, 0, 0, 24, 'f', 't', 'y', 'p', 'i', 's', 'o', 'm'}); got != Unknown {
		t.Errorf("ftyp isom: got %v want Unknown", got)
	}
}

func TestDetectExtensionFallback(t *testing.T) {
	cases := []struct {
		path   string
		header []byte
		want   Kind
	}{
		{"a.jpg", []byte{0x00}, JPEG},
		{"a.JPEG", []byte{0x00}, JPEG},
		{"b.png", []byte{0x00}, PNG},
		{"c.webp", []byte{0x00}, WebP},
		{"d.heic", []byte{0x00}, HEIF},
		{"e.txt", []byte{0x00}, Unknown},
		{"noext", []byte{0x00}, Unknown},
		{"magic.wins", []byte{0xFF, 0xD8, 0xFF, 0xE0}, JPEG},
	}
	for _, c := range cases {
		if got := Detect(c.path, c.header); got != c.want {
			t.Errorf("%s: got %v want %v", c.path, got, c.want)
		}
	}
}
