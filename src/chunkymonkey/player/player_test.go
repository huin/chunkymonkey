package player

import (
	"testing"

	. "chunkymonkey/types"
)

// Tests that the chunk locations are all within expected bounds, and that all
// chunks within min and max bounds are appear exactly once.
func checkChunksPresent(t *testing.T, locs []ChunkXz, minX, maxX, minZ, maxZ ChunkCoord) {
	for _, loc := range locs {
		if loc.X < minX || loc.X > maxX || loc.Z < minZ || loc.Z > maxZ {
			t.Errorf("Found out-of range location %+v", loc)
		}
	}
	for x := ChunkCoord(minX); x <= maxX; x++ {
		for z := ChunkCoord(minZ); z <= maxZ; z++ {
			numFound := 0
			for _, loc := range locs {
				if x == loc.X && z == loc.Z {
					numFound++
				}
			}
			if numFound != 1 {
				t.Errorf(
					"Expected one instance of (%d, %d), but found %d",
					x, z, numFound)
			}
		}
	}
}

// Tests that the chunk locations are in increasing order of max(dx,dz)
// distance from the given center.
func checkChunkOrder(t *testing.T, locs []ChunkXz, center *ChunkXz) {
	var previous *ChunkXz
	var curDistance ChunkCoord
	prevDistance := ChunkCoord(-1)

	for _, loc := range locs {
		t.Logf("%+v", loc)
		dx := (center.X - loc.X).Abs()
		dz := (center.Z - loc.Z).Abs()
		if dx > dz {
			curDistance = dx
		} else {
			curDistance = dz
		}
		if curDistance < prevDistance {
			t.Errorf("got location %+v after closer location %+v", loc, previous)
		}
		prevDistance = curDistance
		previous = &loc
	}
}

func TestChunkOrder(t *testing.T) {
	var locs []ChunkXz

	testChunkOrder := func(radius ChunkCoord, center *ChunkXz) {
		locs = chunkOrder(radius, center)
		t.Logf("Testing chunkOrder(%d, %+v)", radius, center)
		checkChunksPresent(
			t, locs,
			center.X-radius, center.X+radius, center.Z-radius, center.Z+radius)
		checkChunkOrder(t, locs, center)
	}

	// At the origin
	testChunkOrder(2, &ChunkXz{0, 0})

	// Away from the origin
	testChunkOrder(5, &ChunkXz{3, 1})
}
