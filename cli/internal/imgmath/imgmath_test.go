package imgmath

import "testing"

func TestTargetSize(t *testing.T) {
	tests := []struct {
		original int
		percent  int
		want     int
	}{
		{5, 50, 2},
		{7, 50, 4},
		{1000, 33, 330},
		{1000, 10, 100},
		{1, 1, 1},
		{99, 100, 99},
	}
	for _, tc := range tests {
		if got := TargetSize(tc.original, tc.percent); got != tc.want {
			t.Errorf("TargetSize(%d, %d) = %d, want %d", tc.original, tc.percent, got, tc.want)
		}
	}
}

func TestTargetSizeByMaxDim(t *testing.T) {
	tests := []struct {
		name         string
		w, h, maxDim int
		wantW, wantH int
	}{
		{"no_upscale", 800, 600, 1000, 800, 600},
		{"exact_match", 1000, 600, 1000, 1000, 600},
		{"landscape", 4000, 2000, 1000, 1000, 500},
		{"portrait", 2000, 4000, 1000, 500, 1000},
		{"extreme_aspect", 10000, 10, 100, 100, 1},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotW, gotH := TargetSizeByMaxDim(tc.w, tc.h, tc.maxDim)
			if gotW != tc.wantW || gotH != tc.wantH {
				t.Fatalf("TargetSizeByMaxDim(%d, %d, %d) = (%d, %d), want (%d, %d)",
					tc.w, tc.h, tc.maxDim, gotW, gotH, tc.wantW, tc.wantH)
			}
		})
	}
}

func TestTargetBytes(t *testing.T) {
	if got := TargetBytes(200_000, 1000*1000, 500*500); got != 50_000 {
		t.Fatalf("50%% scale target = %d, want 50000", got)
	}
	if got := TargetBytes(1000, 100*100, 50*50); got != MinTargetBytes {
		t.Fatalf("tiny image target = %d, want %d", got, MinTargetBytes)
	}
}

func TestSolveQuality(t *testing.T) {
	encode := func(q int) int64 { return int64(q*10 + 50) }

	if got := SolveQuality(encode, 2000, 40, 97); got != 97 {
		t.Fatalf("all pass = %d, want 97", got)
	}
	if got := SolveQuality(encode, 100, 40, 97); got != 40 {
		t.Fatalf("all fail = %d, want 40", got)
	}
	if got := SolveQuality(encode, 600, 40, 97); got != 55 {
		t.Fatalf("exact boundary = %d, want 55", got)
	}
	if got := SolveQuality(encode, 600, 55, 55); got != 55 {
		t.Fatalf("single point = %d, want 55", got)
	}
}
