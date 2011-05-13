package recipe

import (
	"reflect"
	"strings"
	"testing"

	"chunkymonkey/slot"
	. "chunkymonkey/types"
)

// Borrows some test code from loader_test.go

// Helper for defining multiple input slots with less syntactic boilerplate.
func sl(slots ...slot.Slot) []slot.Slot {
	return slots
}

func assertSlotEq(t *testing.T, expected, result *slot.Slot) {
	if !reflect.DeepEqual(expected, result) {
		t.Error("Slots did not match:")
		t.Logf("Expected: %#v", expected)
		t.Logf("Result:   %#v", result)
	}
}

func TestRecipeSet_Match(t *testing.T) {
	itemTypes := createItemTypes()

	// Helper for defining slots with less syntactic boilerplate.
	s := func(itemTypeId ItemTypeId, count ItemCount, data ItemData) (out *slot.Slot) {
		out = &slot.Slot{
			ItemType: nil,
			Count:    count,
			Data:     data,
		}
		if itemTypeId != ItemTypeIdNull {
			out.ItemType = itemTypes[itemTypeId]
		}
		return
	}

	reader := strings.NewReader(threeRecipes)
	recipes, err := LoadRecipes(reader, itemTypes)
	if err != nil {
		t.Fatal("Failed to load recipes for match test")
	}

	empty := *s(ItemTypeIdNull, 0, 0)
	plank := *s(5, 1, 0)
	log := *s(17, 1, 0)
	flintAndSteel := *s(259, 1, 0)
	iron := *s(265, 1, 0)
	flint := *s(318, 1, 0)

	tests := []struct {
		comment string
		width   int
		height  int
		input   []slot.Slot
		expect  *slot.Slot
	}{
		// A plank in any of 4 slots should produce nothing.
		{
			"P.\n..",
			2, 2,
			sl(plank, empty, empty, empty),
			&empty,
		},
		{
			".P\n..",
			2, 2,
			sl(empty, plank, empty, empty),
			&empty,
		},
		{
			"..\nP.",
			2, 2,
			sl(empty, empty, plank, empty),
			&empty,
		},
		{
			"..\n.P",
			2, 2,
			sl(empty, empty, empty, plank),
			&empty,
		},
		// A log in any of 4 slots should produce 4 planks.
		{
			"L.\n..",
			2, 2,
			sl(log, empty, empty, empty),
			s(5, 4, 0),
		},
		{
			".L\n..",
			2, 2,
			sl(empty, log, empty, empty),
			s(5, 4, 0),
		},
		{
			"..\nL.",
			2, 2,
			sl(empty, empty, log, empty),
			s(5, 4, 0),
		},
		{
			"..\n.L",
			2, 2,
			sl(empty, empty, empty, log),
			s(5, 4, 0),
		},
		// Flint and steel
		{
			"F.\n.I",
			2, 2,
			sl(flint, empty, empty, iron),
			&flintAndSteel,
		},
		{
			"F..\n.I.\n...",
			3, 3,
			sl(flint, empty, empty, empty, iron, empty, empty, empty, empty),
			&flintAndSteel,
		},
		{
			"...\n.F.\n..I",
			3, 3,
			sl(empty, empty, empty, empty, flint, empty, empty, empty, iron),
			&flintAndSteel,
		},
	}

	for i := range tests {
		test := &tests[i]
		t.Logf("Test #%d:\n%s", i, test.comment)
		output := recipes.Match(test.width, test.height, test.input)
		assertSlotEq(t, test.expect, &output)
	}

	// TODO test things other than square or 1x1 recipes
	// TODO test recipes with gaps in
}
