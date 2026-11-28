package format

import "testing"

func TestSniff(t *testing.T) {
	tests := []struct {
		name   string
		header []byte
		want   Kind
	}{
		{"jpeg", []byte{0xFF, 0xD8, 0xFF, 0xE0}, JPEG},
		{"png", []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0x00}, PNG},
		{"webp", []byte{'R', 'I', 'F', 'F', 0, 0, 0, 0, 'W', 'E', 'B', 'P'}, WebP},
		{"heif_heic", []byte("    ftypheic...."), HEIF},
		{"heif_heix", []byte("    ftypheix...."), HEIF},
		{"heif_hevc", []byte("    ftyphevc...."), HEIF},
		{"heif_hevx", []byte("    ftyphevx...."), HEIF},
		{"heif_heim", []byte("    ftypheim...."), HEIF},
		{"heif_heis", []byte("    ftypheis...."), HEIF},
		{"heif_hevm", []byte("    ftyphevm...."), HEIF},
		{"heif_hevs", []byte("    ftyphevs...."), HEIF},
		{"heif_mif1", []byte("    ftypmif1...."), HEIF},
		{"heif_msf1", []byte("    ftypmsf1...."), HEIF},
		{"non_heif_ftyp", []byte("    ftypmp42...."), Unknown},
		{"short", []byte{0xFF, 0xD8}, Unknown},
		{"empty", nil, Unknown},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := Sniff(tc.header); got != tc.want {
				t.Fatalf("Sniff() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestDetect(t *testing.T) {
	tests := []struct {
		name   string
		path   string
		header []byte
		want   Kind
	}{
		{"magic_overrides_ext", "photo.png", []byte{0xFF, 0xD8, 0xFF, 0xE1}, JPEG},
		{"fallback_jpg", "photo.JPG", nil, JPEG},
		{"fallback_jpeg", "photo.jpeg", nil, JPEG},
		{"fallback_png", "photo.png", []byte{1, 2, 3}, PNG},
		{"fallback_webp", "photo.webp", nil, WebP},
		{"fallback_heic", "photo.heic", nil, HEIF},
		{"fallback_heif", "photo.HEIF", nil, HEIF},
		{"unknown", "photo.bmp", nil, Unknown},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := Detect(tc.path, tc.header); got != tc.want {
				t.Fatalf("Detect(%q) = %v, want %v", tc.path, got, tc.want)
			}
		})
	}
}
