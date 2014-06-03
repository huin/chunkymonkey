package gamerules

import (
	. "github.com/huin/chunkymonkey/types"
)

func makeDispenserAspect() (aspect IBlockAspect) {
	return &InventoryAspect{
		name:                 "Dispenser",
		createBlockInventory: createDispenserInventory,
	}
}

func NewDispenserTileEntity() ITileEntity {
	return createDispenserInventory(nil)
}

func createDispenserInventory(instance *BlockInstance) *blockInventory {
	return newBlockInventory(
		instance,
		NewDispenserInventory(),
		false,
		InvTypeIdDispenser,
	)
}

// TODO behaviours for dispensers.
