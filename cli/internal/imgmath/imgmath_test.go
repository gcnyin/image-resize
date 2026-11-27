package imgmath

import "testing"

func TestSolveQuality(t *testing.T) {
	// 单调递增：字节 = 质量 × 1000。
	encode := func(q int) int64 { return int64(q) * 1000 }

	cases := []struct {
		name     string
		target   int64
		min, max int
		want     int
	}{
		{"区间中段", 75_000, 40, 97, 75},
		{"全部达标取上限", 100_000, 40, 97, 97},
		{"全部超目标取下限", 30_000, 40, 97, 40},
		{"恰好命中", 40_000, 40, 97, 40},
	}
	for _, c := range cases {
		if got := SolveQuality(encode, c.target, c.min, c.max); got != c.want {
			t.Errorf("%s: got %d want %d", c.name, got, c.want)
		}
	}
}

func TestSolveQualityMonotonicNonStrict(t *testing.T) {
	// 非严格单调（质量升高字节不变）：任何满足目标的最高质量都合法。
	encode := func(q int) int64 { return 42 }
	got := SolveQuality(encode, 100, 40, 97)
	if got != 97 {
		t.Errorf("恒定字节: got %d want 97", got)
	}
}
