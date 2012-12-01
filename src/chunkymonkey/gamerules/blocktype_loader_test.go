package gamerules

import (
	"bytes"
	"reflect"
	"strings"
	"testing"

	. "chunkymonkey/types"
)

const twoBlocks = ("{\n" +
	"  \"0\": {\n" +
	"    \"Aspect\": \"Void\",\n" +
	"    \"AspectArgs\": {},\n" +
	"    \"Name\": \"air\",\n" +
	"    \"Opacity\": 0,\n" +
	"    \"Destructable\": true,\n" +
	"    \"Solid\": false,\n" +
	"    \"Replaceable\": true,\n" +
	"    \"Attachable\": false\n" +
	"  },\n" +
	"  \"1\": {\n" +
	"    \"Aspect\": \"Standard\",\n" +
	"    \"AspectArgs\": {\n" +
	"      \"DroppedItems\": [\n" +
	"        {\n" +
	"          \"DroppedItem\": 4,\n" +
	"          \"Probability\": 100,\n" +
	"          \"Count\": 1\n" +
	"        }\n" +
	"      ],\n" +
	"      \"BreakOn\": 2\n" +
	"    },\n" +
	"    \"Name\": \"stone\",\n" +
	"    \"Opacity\": 15,\n" +
	"    \"Destructable\": true,\n" +
	"    \"Solid\": true,\n" +
	"    \"Replaceable\": false,\n" +
	"    \"Attachable\": true\n" +
	"  }\n" +
	"}")

const badAspect = ("{\n" +
	"  \"0\": {\n" +
	"    \"Aspect\": \"Standard\",\n" +
	"    \"AspectArgs\": {\n" +
	"      \"DroppedItems\": 5,\n" +
	"      \"BreakOn\": \"foo\"\n" +
	"    },\n" +
	"    \"Name\": \"air\",\n" +
	"    \"Opacity\": 0,\n" +
	"    \"Destructable\": true,\n" +
	"    \"Solid\": false,\n" +
	"    \"Replaceable\": true,\n" +
	"    \"Attachable\": false\n" +
	"  }\n" +
	"}")

func assertBlockTypeEq(t *testing.T, expected, result *BlockType) {
	if !reflect.DeepEqual(expected, result) {
		t.Error("BlockTypes differed")
		t.Logf("    expected %#v", expected)
		t.Logf("    result   %#v", result)
		t.Logf("    expected aspect %#v", expected.Aspect)
		t.Logf("    result aspect   %#v", result.Aspect)
	}
}

func displayBlocks(t *testing.T, blocks BlockTypeList) {
	t.Logf("%d blocks:", len(blocks))
	for id := range blocks {
		block := &blocks[id]
		t.Logf("    [%d] attrs=%#v aspect=%#v", id, block.BlockAttrs, block.Aspect)
	}
}

func makeBlockType(attrs BlockAttrs, aspect IBlockAspect) *BlockType {
	blockType := &BlockType{
		BlockAttrs: attrs,
		Aspect:     aspect,
	}
	aspect.setAttrs(&blockType.BlockAttrs)
	return blockType
}

func TestLoadBlockDefs(t *testing.T) {
	reader := strings.NewReader(twoBlocks)
	t.Log(twoBlocks)

	blocks, err := LoadBlockDefs(reader)
	if err != nil {
		t.Fatalf("Expected no error but got %v", err)
	}
	if len(blocks) != 2 {
		t.Fatalf("Expected 2 block defs, but got %d", len(blocks))
	}

	assertBlockTypeEq(
		t,
		makeBlockType(
			BlockAttrs{
				id:           0,
				Name:         "air",
				Opacity:      0,
				defined:      true,
				Destructable: true,
				Solid:        false,
				Replaceable:  true,
				Attachable:   false,
			},
			&VoidAspect{},
		),
		&blocks[0],
	)

	assertBlockTypeEq(
		t,
		makeBlockType(
			BlockAttrs{
				id:           1,
				Name:         "stone",
				Opacity:      15,
				defined:      true,
				Destructable: true,
				Solid:        true,
				Replaceable:  false,
				Attachable:   true,
			},
			&StandardAspect{
				DroppedItems: []blockDropItem{
					blockDropItem{
						DroppedItem: 4,
						Probability: 100,
						Count:       1,
					},
				},
				BreakOn: DigBlockBroke,
			},
		),
		&blocks[1],
	)
}

// Test if loading blocks, saving them and loading them gives an equal result
// to the original.
func TestLoadSaveAndLoadBlockDefs(t *testing.T) {
	var err error

	var originalBlocks BlockTypeList
	{
		reader := strings.NewReader(twoBlocks)

		originalBlocks, err = LoadBlockDefs(reader)
		if err != nil {
			t.Fatalf("Expected no error on first load but got %v", err)
		}
	}

	var savedData []byte
	{
		writer := &bytes.Buffer{}
		err = SaveBlockDefs(writer, originalBlocks)
		if err != nil {
			t.Fatalf("Expected no error on write but got %v", err)
		}
		savedData = writer.Bytes()
	}
	t.Logf("Saved data: %s", savedData)

	var reloadedBlocks BlockTypeList
	{
		reader := bytes.NewBuffer(savedData)

		reloadedBlocks, err = LoadBlockDefs(reader)
		if err != nil {
			t.Fatalf("Expected no error on second load but got %v", err)
		}
	}

	if !reflect.DeepEqual(originalBlocks, reloadedBlocks) {
		t.Errorf("Expected reloaded blocks to be the same as the original")
		t.Logf("Original:")
		displayBlocks(t, originalBlocks)
		t.Logf("Reloaded:")
		displayBlocks(t, reloadedBlocks)
	}
}

func TestBadAspectArgFails(t *testing.T) {
	reader := strings.NewReader(badAspect)
	t.Log(badAspect)

	blocks, err := LoadBlockDefs(reader)
	if err == nil {
		t.Error("Expected an error")
	}
	displayBlocks(t, blocks)
}
