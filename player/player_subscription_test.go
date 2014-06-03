package player

import (
	"fmt"
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
func checkChunkOrder(t *testing.T, locs []ChunkXz, center ChunkXz) {
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

	testChunkOrder := func(radius ChunkCoord, center ChunkXz) {
		locs = orderedChunkSquare(center, radius)
		t.Logf("Testing chunkOrder(%d, %+v)", radius, center)
		checkChunksPresent(
			t, locs,
			center.X-radius, center.X+radius, center.Z-radius, center.Z+radius)
		checkChunkOrder(t, locs, center)
	}

	// At the origin
	testChunkOrder(2, ChunkXz{0, 0})

	// Away from the origin
	testChunkOrder(5, ChunkXz{3, 1})
}

// Slow, dumb and simple implementation of squareDifference() that should be
// easy to check by eye. We use this to generate the expected results for tests
// on squareDifference().
func slowSimpleSquareDifference(centerA, centerB ChunkXz, radius ChunkCoord) []ChunkXz {
	result := make([]ChunkXz, 0, radius*radius)
	for x := centerA.X - radius; x <= centerA.X+radius; x++ {
		for z := centerA.Z - radius; z <= centerA.Z+radius; z++ {
			if x >= centerB.X-radius && x <= centerB.X+radius && z >= centerB.Z-radius && z <= centerB.Z+radius {
				// {x, z} is within square B. Don't include this.
				continue
			}
			result = append(result, ChunkXz{x, z})
		}
	}
	return result
}

type diffTest struct {
	centerA ChunkXz
	centerB ChunkXz
	radius  ChunkCoord
	exp     []ChunkXz
}

func newDiffTest(centerA, centerB ChunkXz, radius ChunkCoord) *diffTest {
	return &diffTest{
		centerA: centerA,
		centerB: centerB,
		radius:  radius,
		exp:     slowSimpleSquareDifference(centerA, centerB, radius),
	}
}

func (dt *diffTest) String() string {
	return fmt.Sprintf(
		"centerA=%v centerB=%v radius=%d", dt.centerA, dt.centerB, dt.radius,
	)
}

func (dt *diffTest) run(t *testing.T) {
	t.Logf("difference test: %v", dt)

	// Run the fn under test:
	result := squareDifference(dt.centerA, dt.centerB, dt.radius)

	// Check results:
	if len(dt.exp) != len(result) {
		t.Errorf(
			"  expected len(result) == %d, but got %d",
			len(dt.exp), len(result),
		)
	}
	for _, expXz := range dt.exp {
		found := false
	SEARCH:
		for _, resXz := range result {
			if expXz.X == resXz.X && expXz.Z == resXz.Z {
				found = true
				break SEARCH
			}
		}
		if !found {
			t.Errorf("  missing %#v", expXz)
		}
	}
}

func Test_squareDifference(t *testing.T) {
	radius := ChunkCoord(2)
	start := radius*-2 - 1
	end := radius*2 + 1
	centerA := ChunkXz{0, 0}
	for x := start; x <= end; x++ {
		for z := start; z <= end; z++ {
			centerB := ChunkXz{x, z}
			dt := newDiffTest(centerA, centerB, radius)
			dt.run(t)
		}
	}
}
