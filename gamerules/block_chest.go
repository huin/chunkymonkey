package gamerules

import (
	. "github.com/huin/chunkymonkey/types"
)

func makeChestAspect() (aspect IBlockAspect) {
	return &InventoryAspect{
		name:                 "Chest",
		createBlockInventory: createChestInventory,
	}
}

// Creates a new tile entity for a chest. UnmarshalNbt and SetChunk must be
// called before any other methods.
func NewChestTileEntity() ITileEntity {
	return createChestInventory(nil)
}

func createChestInventory(instance *BlockInstance) *blockInventory {
	return newBlockInventory(
		instance,
		NewChestInventory(),
		false,
		InvTypeIdChest,
	)
}
