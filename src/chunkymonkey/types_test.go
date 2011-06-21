package types

import (
	"testing"
)

func TestChunkSizeConsts(t *testing.T) {
	type Test struct {
		desc     string
		input    int
		expected int
	}

	tests := []Test{
		{"ChunkSizeH", ChunkSizeH, 16},
		{"ChunkSizeY", ChunkSizeY, 128},
	}

	for _, r := range tests {
		if r.expected != r.input {
			t.Errorf("Expected %s == %d but it was %d", r.desc, r.expected, r.input)
		}
	}
}

func TestLookDegrees_ToLookBytes(t *testing.T) {
	type Test struct {
		input    LookDegrees
		expected LookBytes
	}

	var tests = []Test{
		{LookDegrees{0, 0}, LookBytes{0, 0}},
		{LookDegrees{0, 90}, LookBytes{0, 64}},
		{LookDegrees{0, 180}, LookBytes{0, 128}},
		{LookDegrees{0, -90}, LookBytes{0, 192}},
		{LookDegrees{0, 270}, LookBytes{0, 192}},
		{LookDegrees{90, 0}, LookBytes{64, 0}},
		{LookDegrees{180, 0}, LookBytes{128, 0}},
		{LookDegrees{-90, 0}, LookBytes{192, 0}},
		{LookDegrees{270, 0}, LookBytes{192, 0}},
	}

	for _, r := range tests {
		result := r.input.ToLookBytes()
		if r.expected.Yaw != result.Yaw || r.expected.Pitch != result.Pitch {
			t.Errorf("LookDegrees%v expected LookBytes%v got LookBytes%v",
				r.input, r.expected, result)
		}
	}
}

func TestChunkSideDir_GetOpposite(t *testing.T) {
	type Test struct {
		input, expected ChunkSideDir
	}

	tests := []Test{
		{ChunkSideEast, ChunkSideWest},
		{ChunkSideSouth, ChunkSideNorth},
		{ChunkSideWest, ChunkSideEast},
		{ChunkSideNorth, ChunkSideSouth},
	}

	for i := range tests {
		input, expected := tests[i].input, tests[i].expected
		result := input.GetOpposite()
		if expected != result {
			t.Errorf(
				"%T(%v).GetOpposite()=>%d but expected %d",
				input, input, result, expected)
		}
	}
}

func TestAbsXyz_ToChunkXz(t *testing.T) {
	type Test struct {
		input    AbsXyz
		expected ChunkXz
	}
	var tests = []Test{
		{AbsXyz{0, 0, 0}, ChunkXz{0, 0}},
		{AbsXyz{0, 0, 16}, ChunkXz{0, 1}},
		{AbsXyz{16, 0, 0}, ChunkXz{1, 0}},
		{AbsXyz{0, 0, -16}, ChunkXz{0, -1}},
		{AbsXyz{-16, 0, 0}, ChunkXz{-1, 0}},
		{AbsXyz{-1, 0, -1}, ChunkXz{-1, -1}},
	}

	for _, test := range tests {
		input, expected := test.input, test.expected
		result := input.ToChunkXz()
		if expected.X != result.X || expected.Z != result.Z {
			t.Errorf("AbsXyz%+v.UpdateChunkXz() expected ChunkXz%+v got ChunkXz%+v",
				input, expected, result)
		}
	}
}

func TestAbsXyz_ToBlockXyz(t *testing.T) {
	type Test struct {
		pos AbsXyz
		exp BlockXyz
	}

	var tests = []Test{
		// Simple positive tests
		{AbsXyz{0.0, 0.0, 0.0}, BlockXyz{0, 0, 0}},
		{AbsXyz{0.1, 0.2, 0.3}, BlockXyz{0, 0, 0}},
		{AbsXyz{1.0, 2.0, 3.0}, BlockXyz{1, 2, 3}},

		// Negative tests
		{AbsXyz{-0.1, -0.2, -0.3}, BlockXyz{-1, -1, -1}},
		{AbsXyz{-1.0, -2.0, -3.0}, BlockXyz{-1, -2, -3}},
		{AbsXyz{-1.5, -2.5, -3.5}, BlockXyz{-2, -3, -4}},
	}

	for _, r := range tests {
		result := r.pos.ToBlockXyz()
		if r.exp.X != result.X || r.exp.Y != result.Y || r.exp.Z != result.Z {
			t.Errorf("AbsXyz%v.ToBlockXyz() expected BlockXyz%v got BlockXyz%v",
				r.pos, r.exp, result)
		}
	}
}

func TestAbsIntXyz_ToChunkXz(t *testing.T) {
	type Test struct {
		input    AbsIntXyz
		expected ChunkXz
	}

	var tests = []Test{
		{AbsIntXyz{0, 0, 0}, ChunkXz{0, 0}},
		{AbsIntXyz{8 * 32, 0, 8 * 32}, ChunkXz{0, 0}},
		{AbsIntXyz{15 * 32, 0, 15 * 32}, ChunkXz{0, 0}},
		{AbsIntXyz{16 * 32, 0, 16 * 32}, ChunkXz{1, 1}},
		{AbsIntXyz{31*32 + 31, 0, 31*32 + 31}, ChunkXz{1, 1}},
		{AbsIntXyz{32 * 32, 0, 32 * 32}, ChunkXz{2, 2}},
		{AbsIntXyz{0, 0, 32 * 32}, ChunkXz{0, 2}},
		{AbsIntXyz{0, 0, -16 * 32}, ChunkXz{0, -1}},
		{AbsIntXyz{0, 0, -1 * 32}, ChunkXz{0, -1}},
		{AbsIntXyz{0, 0, -1}, ChunkXz{0, -1}},
	}

	for _, r := range tests {
		result := r.input.ToChunkXz()
		if r.expected.X != result.X || r.expected.Z != result.Z {
			t.Errorf("AbsIntXyz%v expected ChunkXz%v got ChunkXz%v",
				r.input, r.expected, result)
		}
	}
}

func TestCoordDivMod(t *testing.T) {
	type CoordDivModTest struct {
		expected_div, expected_mod int32
		num, denom                 int32
	}

	var CoordDivModTests = []CoordDivModTest{
		// Simple +ve numerator cases
		CoordDivModTest{0, 0, 0, 16},
		CoordDivModTest{0, 1, 1, 16},
		CoordDivModTest{0, 15, 15, 16},
		CoordDivModTest{1, 0, 16, 16},
		CoordDivModTest{1, 15, 31, 16},

		// -ve numerator cases
		CoordDivModTest{-1, 15, -1, 16},
		CoordDivModTest{-1, 0, -16, 16},
		CoordDivModTest{-2, 15, -17, 16},
		CoordDivModTest{-2, 0, -32, 16},
	}

	for _, r := range CoordDivModTests {
		div, mod := coordDivMod(r.num, r.denom)
		if r.expected_div != div || r.expected_mod != mod {
			t.Errorf("coordDivMod(%d, %d) expected (%d, %d) got (%d, %d)",
				r.num, r.denom, r.expected_div, r.expected_mod, div, mod)
		}
	}
}

func TestChunkCoord_ToShardCoord(t *testing.T) {
	type Test struct {
		input    ChunkCoord
		expected ShardCoord
	}

	tests := []Test{
		{-2*ShardSize - 1, -3},
		{-2 * ShardSize, -2},
		{-ShardSize - 1, -2},
		{-ShardSize, -1},
		{-1, -1},
		{0, 0},
		{ShardSize - 1, 0},
		{ShardSize, 1},
		{2*ShardSize - 1, 1},
		{2 * ShardSize, 2},
	}

	for _, test := range tests {
		result := test.input.ToShardCoord()
		if test.expected != result {
			t.Errorf(
				"ChunkCoord(%d) expected %d, but got %d",
				test.input, test.expected, result,
			)
		}
	}
}

func TestChunkXz_GetChunkCornerBlockXY(t *testing.T) {
	type Test struct {
		input    ChunkXz
		expected BlockXyz
	}

	var tests = []Test{
		{ChunkXz{0, 0}, BlockXyz{0, 0, 0}},
		{ChunkXz{0, 1}, BlockXyz{0, 0, 16}},
		{ChunkXz{1, 0}, BlockXyz{16, 0, 0}},
		{ChunkXz{0, -1}, BlockXyz{0, 0, -16}},
		{ChunkXz{-1, 0}, BlockXyz{-16, 0, 0}},
	}

	for _, r := range tests {
		result := r.input.GetChunkCornerBlockXY()
		if r.expected.X != result.X || r.expected.Y != result.Y || r.expected.Z != result.Z {
			t.Errorf("ChunkXz%v expected BlockXyz%v got BlockXyz%v",
				r.input, r.expected, result)
		}
	}
}

func TestChunkXz_ChunkKey(t *testing.T) {
	type Test struct {
		input    ChunkXz
		expected uint64
	}

	var tests = []Test{
		{ChunkXz{0, 0}, 0},
		{ChunkXz{0, 1}, 0x0000000000000001},
		{ChunkXz{1, 0}, 0x0000000100000000},
		{ChunkXz{0, -1}, 0x00000000ffffffff},
		{ChunkXz{-1, 0}, 0xffffffff00000000},
		{ChunkXz{0, 10}, 0x000000000000000a},
		{ChunkXz{10, 0}, 0x0000000a00000000},
		{ChunkXz{10, 11}, 0x0000000a0000000b},
	}

	for _, r := range tests {
		result := r.input.ChunkKey()
		if r.expected != result {
			t.Errorf("ChunkXz%+v.ChunkKey() expected %d got %d",
				r.input, r.expected, result)
		}
	}
}

func BenchmarkSubChunkXyz_BlockIndex(b *testing.B) {
	loc := SubChunkXyz{1, 2, 3}
	for i := 0; i < b.N; i++ {
		loc.BlockIndex()
	}
}

func BenchmarkBlockIndex_ToSubChunkXyz(b *testing.B) {
	index := BlockIndex(123)
	for i := 0; i < b.N; i++ {
		index.ToSubChunkXyz()
	}
}

func TestSubChunkXyz_BlockIndex(t *testing.T) {
	type Test struct {
		input    SubChunkXyz
		expIndex BlockIndex
		expOk    bool
	}

	tests := []Test{
		Test{SubChunkXyz{0, 0, 0}, 0, true},
		Test{SubChunkXyz{0, 1, 0}, 1, true},
		Test{SubChunkXyz{0, 2, 0}, 2, true},
		Test{SubChunkXyz{0, 3, 0}, 3, true},

		Test{SubChunkXyz{0, 127, 0}, 127, true},
		Test{SubChunkXyz{0, 0, 1}, 128, true},

		Test{SubChunkXyz{0, 127, 1}, 255, true},
		Test{SubChunkXyz{0, 0, 2}, 256, true},

		Test{SubChunkXyz{1, 0, 0}, 16 * 128, true},

		// Invalid locations
		Test{SubChunkXyz{16, 0, 0}, 0, false},
		Test{SubChunkXyz{0, 128, 0}, 0, false},
		Test{SubChunkXyz{0, 0, 16}, 0, false},
	}

	for _, r := range tests {
		t.Logf("%#v", r.input)
		t.Logf("  expecting: index=%d, ok=%t", r.expIndex, r.expOk)
		index, ok := r.input.BlockIndex()
		if r.expOk != ok {
			t.Errorf("  ok=%t", ok)
		}
		if !ok {
			continue
		}
		if r.expIndex != index {
			t.Errorf("  index=%d", index)
			continue
		}
		// Test reverse conversion.
		subLoc := index.ToSubChunkXyz()
		if subLoc.X != r.input.X || subLoc.Y != r.input.Y || subLoc.Z != r.input.Z {
			t.Errorf("  reverse conversion to SubChunkXyz resulted in %#v", subLoc)
		}
	}
}

// {{{ BlockIndex tests

type blockIndexTest struct {
	index    BlockIndex
	input    byte
	before   []byte
	expAfter []byte
}

func (test *blockIndexTest) test(t *testing.T, desc string, fn func(index BlockIndex, input byte, data []byte)) {
	t.Logf("%T(%v) %s", test.index, test.index, desc)
	t.Logf("  before   = %v", test.before)
	t.Logf("  expAfter = %v", test.expAfter)
	if len(test.before) != len(test.expAfter) {
		t.Errorf("  Error in test: data lengths not equal")
		return
	}

	data := make([]byte, len(test.before))
	copy(data, test.before)

	fn(test.index, test.input, data)

	t.Logf("  result   = %v", data)

	for i := range data {
		if test.expAfter[i] != data[i] {
			t.Errorf("  Fail: output differs at index %d", i)
			break
		}
	}
}

type blockIndexTests []blockIndexTest

func (tests blockIndexTests) runTests(t *testing.T, desc string, fn func(index BlockIndex, input byte, data []byte)) {
	for i := range tests {
		tests[i].test(t, desc, fn)
	}
}

func TestBlockIndex_SetBlockId(t *testing.T) {
	tests := blockIndexTests{
		{0, 1, []byte{2, 2}, []byte{1, 2}},
		{1, 1, []byte{2, 2}, []byte{2, 1}},
	}
	tests.runTests(t, "SetBlockId", func(index BlockIndex, input byte, data []byte) {
		index.SetBlockId(data, BlockId(input))
	})
}

func TestBlockIndex_SetBlockData(t *testing.T) {
	tests := blockIndexTests{
		// Tests indexing of the nibble, and correct bit setting in filled bytes.
		{0, 1, []byte{0xff, 0xff}, []byte{0xf1, 0xff}},
		{1, 1, []byte{0xff, 0xff}, []byte{0x1f, 0xff}},
		{2, 1, []byte{0xff, 0xff}, []byte{0xff, 0xf1}},
		{3, 1, []byte{0xff, 0xff}, []byte{0xff, 0x1f}},

		// Tests correct bit setting in zero bytes.
		{0, 1, []byte{0x00, 0x00}, []byte{0x01, 0x00}},
		{1, 1, []byte{0x00, 0x00}, []byte{0x10, 0x00}},

		// Tests correct bit setting in half-filled bytes.
		{0, 1, []byte{0x0f, 0x0f}, []byte{0x01, 0x0f}},
		{1, 1, []byte{0x0f, 0x0f}, []byte{0x1f, 0x0f}},
	}
	tests.runTests(t, "SetBlockData", func(index BlockIndex, input byte, data []byte) {
		index.SetBlockData(data, input)
	})
}

// }}} BlockIndex tests

func TestAddXyz_Overflow(t *testing.T) {
	type Test struct {
		input *BlockXyz
		dx BlockCoord
		dy BlockYCoord
		dz BlockCoord
		expected *BlockXyz
	}
	maxblock := &BlockXyz{MaxXCoord, MaxYCoord, MaxZCoord}
	minblock := &BlockXyz{MinXCoord, MinYCoord, MinZCoord}
	var tests = []Test{
		{&BlockXyz{0, 0, 0}, 5, 5, 5, &BlockXyz{5, 5, 5}},
		{maxblock, 0, 0, 0, maxblock},
		{minblock, 0, 0, 0, minblock},
		{maxblock, 1, 0, 0, nil},
		{maxblock, 0, 1, 0, nil},
		{maxblock, 0, 0, 1, nil},
		{minblock, -1, 0, 0, nil},
		{minblock, 0, -1, 0, nil},
		{minblock, 0, 0, -1, nil},
		{&BlockXyz{MaxXCoord, 0, 0}, 0, 5, -5, &BlockXyz{MaxXCoord, 5, -5}},
		{&BlockXyz{MinXCoord, 0, 0}, 0, 5, -5, &BlockXyz{MinXCoord, 5, -5}},
	}

	for _, r := range tests {
		result := r.input.AddXyz(r.dx, r.dy, r.dz)
		if r.expected == nil {
			if result != nil {
				t.Errorf("BlockXyz%v expected nil got BlockXyz%v", r.input, result)
			}
		} else if r.expected.X != result.X || r.expected.Y != result.Y || r.expected.Z != result.Z {
			t.Errorf("BlockXyz%v expected BlockXyz%v got BlockXyz%v",
				r.input, r.expected, result)
		}
	}
}

func TestBlockXyz_ToAbsIntXyz(t *testing.T) {
	type Test struct {
		input    BlockXyz
		expected AbsIntXyz
	}

	var tests = []Test{
		{BlockXyz{0, 0, 0}, AbsIntXyz{0, 0, 0}},
		{BlockXyz{0, 0, 1}, AbsIntXyz{0, 0, 32}},
		{BlockXyz{0, 0, -1}, AbsIntXyz{0, 0, -32}},
		{BlockXyz{1, 0, 0}, AbsIntXyz{32, 0, 0}},
		{BlockXyz{-1, 0, 0}, AbsIntXyz{-32, 0, 0}},
		{BlockXyz{0, 1, 0}, AbsIntXyz{0, 32, 0}},
		{BlockXyz{0, 10, 0}, AbsIntXyz{0, 320, 0}},
		{BlockXyz{0, 63, 0}, AbsIntXyz{0, 2016, 0}},
		{BlockXyz{0, 64, 0}, AbsIntXyz{0, 2048, 0}},
	}

	for _, r := range tests {
		result := r.input.ToAbsIntXyz()
		if r.expected.X != result.X || r.expected.Y != result.Y || r.expected.Z != result.Z {
			t.Errorf("BlockXyz%v expected AbsIntXyz%v got AbsIntXyz%v",
				r.input, r.expected, result)
		}
	}
}
