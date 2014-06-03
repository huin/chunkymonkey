package generation

import (
	"testing"

	. "github.com/huin/chunkymonkey/types"
)

func Benchmark_TestGenerator_generate(b *testing.B) {
	gen := NewTestGenerator(0)
	var loc ChunkXz

	b.ResetTimer()
	b.StartTimer()

	for i := 0; i < b.N; i++ {
		// Generate a good sweep of different chunks, but don't go off forever.
		loc.X = ChunkCoord(i & 0xffff)
		gen.ReadChunk(loc)
	}
}
