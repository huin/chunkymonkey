package itemtype

import (
	"reflect"
	"strings"
	"testing"

	. "chunkymonkey/types"
)

const threeItems = ("{\n" +
	"  \"256\": {\n" +
	"    \"Name\": \"iron shovel\",\n" +
	"    \"MaxStack\": 1,\n" +
	"    \"ToolType\": 1,\n" +
	"    \"ToolUses\": 251\n" +
	"  },\n" +
	"  \"264\": {\n" +
	"    \"Name\": \"diamond\",\n" +
	"    \"MaxStack\": 64\n" +
	"  },\n" +
	"  \"261\": {\n" +
	"    \"Name\": \"bow\",\n" +
	"    \"MaxStack\": 1,\n" +
	"    \"ToolType\": 11\n" +
	"  }\n" +
	"}")

func assertItemTypeEq(t *testing.T, expected *ItemType, result *ItemType) {
	if !reflect.DeepEqual(expected, result) {
		t.Error("ItemTypes differed")
		t.Logf("    expected %#v", expected)
		t.Logf("    result   %#v", result)
	}
}

func displayItems(t *testing.T, items ItemTypeMap) {
	t.Logf("%d items:", len(items))
	for id, item := range items {
		t.Logf("    [%d] %#v", id, item)
	}
}

func TestLoadItemDefs(t *testing.T) {
	reader := strings.NewReader(threeItems)
	t.Log(threeItems)

	items, err := LoadItemDefs(reader)
	if err != nil {
		t.Fatalf("Expected no error but got %v", err)
	}
	// All 3 items should be defined, plus the "null" item.
	if len(items) != 4 {
		t.Fatalf("Expected 4 item types, but got %d", len(items))
	}

	assertItemTypeEq(
		t,
		&ItemType{
			Id:       ItemTypeIdNull,
			Name:     "null item",
			MaxStack: 0,
			ToolType: 0,
			ToolUses: 0,
		},
		items[ItemTypeIdNull],
	)

	assertItemTypeEq(
		t,
		&ItemType{
			Id:       256,
			Name:     "iron shovel",
			MaxStack: 1,
			ToolType: 1,
			ToolUses: 251,
		},
		items[256],
	)

	assertItemTypeEq(
		t,
		&ItemType{
			Id:       264,
			Name:     "diamond",
			MaxStack: 64,
			ToolType: 0,
			ToolUses: 0,
		},
		items[264],
	)

	assertItemTypeEq(
		t,
		&ItemType{
			Id:       261,
			Name:     "bow",
			MaxStack: 1,
			ToolType: 11,
			ToolUses: 0,
		},
		items[261],
	)
}
