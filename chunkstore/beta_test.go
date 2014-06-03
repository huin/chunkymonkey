package chunkstore

import (
	"testing"

	. "github.com/huin/chunkymonkey/types"
)

func TestRegionLoc_regionFilePath(t *testing.T) {
	type Test struct {
		loc      ChunkXz
		expected string
	}

	tests := []Test{
		{ChunkXz{0, 0}, "/foo/r.0.0.mcr"},
		{ChunkXz{31, 0}, "/foo/r.0.0.mcr"},
		{ChunkXz{0, 31}, "/foo/r.0.0.mcr"},
		{ChunkXz{32, 0}, "/foo/r.1.0.mcr"},
		{ChunkXz{0, 32}, "/foo/r.0.1.mcr"},
		{ChunkXz{-1, 0}, "/foo/r.-1.0.mcr"},
		{ChunkXz{0, -1}, "/foo/r.0.-1.mcr"},
		{ChunkXz{-32, 0}, "/foo/r.-1.0.mcr"},
		{ChunkXz{0, -32}, "/foo/r.0.-1.mcr"},
		{ChunkXz{-33, 0}, "/foo/r.-2.0.mcr"},
		{ChunkXz{0, -33}, "/foo/r.0.-2.mcr"},
	}

	for _, test := range tests {
		regionLoc := regionLocForChunkXz(test.loc)
		result := regionLoc.regionFilePath("/foo")
		if test.expected != result {
			t.Errorf(
				"regionFilePath(\"/foo\", %+v) expected %#v but got %#v",
				test.loc, test.expected, result)
		}
	}
}

func TestChunkOffset(t *testing.T) {
	if chunkOffset(0).IsPresent() {
		t.Errorf("chunkOffset(0).IsPresent() should return false")
	}

	type Test struct {
		offset         chunkOffset
		expSectorIndex uint32
		expSectorCount uint32
	}

	tests := []Test{
		{0x00000101, 1, 1},
		{0x00000aff, 10, 255},
	}

	for _, test := range tests {
		sectorCount, sectorIndex := test.offset.Get()
		if test.expSectorCount != sectorCount || test.expSectorIndex != sectorIndex {
			t.Errorf(
				"chunkOffset(0x%x) expected (%d, %d) but got (%d, %d)",
				test.offset,
				test.expSectorCount, test.expSectorIndex,
				sectorCount, sectorIndex)
		}
	}
}
