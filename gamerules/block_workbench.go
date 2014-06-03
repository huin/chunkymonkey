package gamerules

import (
	. "github.com/huin/chunkymonkey/types"
)

func makeWorkbenchAspect() (aspect IBlockAspect) {
	return &InventoryAspect{
		name:                 "Workbench",
		createBlockInventory: createWorkbenchInventory,
	}
}

func createWorkbenchInventory(instance *BlockInstance) *blockInventory {
	inv := NewWorkbenchInventory()
	return newBlockInventory(
		instance,
		inv,
		true,
		InvTypeIdWorkbench,
	)
}
