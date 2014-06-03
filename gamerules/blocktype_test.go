package gamerules

import (
	"testing"
)

func TestMergeBlockItems(t *testing.T) {
	// We hypothetically define "stone" *item* type to have different values
	// than those that would be derived from the block by default.
	itemTypes := ItemTypeMap{
		1: &ItemType{
			Id:       1,
			Name:     "stone item",
			MaxStack: 32,
			ToolType: 0,
			ToolUses: 0,
		},
		5: &ItemType{
			Id: 5,
		},
	}

	blockTypes := BlockTypeList{
		BlockType{
			BlockAttrs: BlockAttrs{
				Name:    "air",
				defined: true,
			},
		},
		BlockType{
			BlockAttrs: BlockAttrs{
				Name:    "stone block",
				defined: true,
			},
		},
		// These block types should not create an item, as they are not defined.
		BlockType{
			BlockAttrs: BlockAttrs{
				defined: false,
			},
		},
		BlockType{
			BlockAttrs: BlockAttrs{
				defined: false,
			},
		},
		BlockType{
			BlockAttrs: BlockAttrs{
				defined: false,
			},
		},
		BlockType{
			BlockAttrs: BlockAttrs{
				Name:    "wood plank",
				defined: true,
			},
		},
	}

	blockTypes.CreateBlockItemTypes(itemTypes)

	for id, itemType := range itemTypes {
		t.Logf("[%d] = %#v", id, itemType)
	}

	if 3 != len(itemTypes) {
		t.Fatalf("Expected 3 item types to exist, but found %d", len(itemTypes))
	}

	assertItemTypeEq(t,
		&ItemType{
			Id:       0,
			Name:     "air",
			MaxStack: MaxStackDefault,
			ToolType: 0,
			ToolUses: 0,
		},
		itemTypes[0],
	)

	// Stone should contain the custom overrides.
	assertItemTypeEq(t,
		&ItemType{
			Id:       1,
			Name:     "stone item",
			MaxStack: 32,
			ToolType: 0,
			ToolUses: 0,
		},
		itemTypes[1],
	)

	// Plank should get its Name and MaxStack filled in.
	assertItemTypeEq(t,
		&ItemType{
			Id:       5,
			Name:     "wood plank",
			MaxStack: 64,
			ToolType: 0,
			ToolUses: 0,
		},
		itemTypes[5],
	)
}
