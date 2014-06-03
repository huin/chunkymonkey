package perlin

import (
	"testing"
)

func Benchmark_Perlin_At2d(b *testing.B) {
	n := NewPerlinNoise(0)
	b.ResetTimer()
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		n.At2d(0, 0)
	}
}
