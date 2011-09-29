package gamerules

import (
	. "chunkymonkey/types"
)

type BlockAttrs struct {
	id              BlockId
	Name            string
	Opacity         int8
	defined         bool
	Destructable    bool
	Solid           bool
	Replaceable     bool
	Attachable      bool
	BlastResistance float32
	Luminance       int8
}

// The core information about any block type.
type BlockType struct {
	BlockAttrs
	Aspect IBlockAspect
}

// Lookup table of blocks.
type BlockTypeList []BlockType

// Get returns the requested BlockType by ID. ok = false if the block type does
// not exist.
func (btl *BlockTypeList) Get(id BlockId) (block *BlockType, ok bool) {
	if id < 0 || int(id) > len(*btl) {
		ok = false
		return
	}
	block = &(*btl)[id]
	ok = block.defined
	return
}

// MergeBlockItems creates default item types from a defined list of block
// types. It does not override any pre-existing items types.
func (btl *BlockTypeList) CreateBlockItemTypes(itemTypes ItemTypeMap) {
	for id := range *btl {
		blockType := &(*btl)[id]
		if !blockType.defined {
			continue
		}
		if itemType, exists := itemTypes[ItemTypeId(id)]; exists {
			if len(itemType.Name) == 0 {
				itemType.Name = blockType.Name
			}
			if itemType.MaxStack == 0 {
				itemType.MaxStack = MaxStackDefault
			}
			continue
		}

		itemTypes[ItemTypeId(id)] = &ItemType{
			Id:       ItemTypeId(id),
			Name:     blockType.Name,
			MaxStack: MaxStackDefault,
		}
	}
}
