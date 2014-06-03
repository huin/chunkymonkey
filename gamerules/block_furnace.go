package gamerules

import (
	. "chunkymonkey/types"
)

func makeFurnaceAspect() IBlockAspect {
	return &FurnaceAspect{
		InventoryAspect: InventoryAspect{
			name:                 "Furnace",
			createBlockInventory: createFurnaceInventory,
		},
	}
}

type FurnaceAspect struct {
	InventoryAspect
	Inactive BlockId
	Active   BlockId
}

// Creates a new tile entity for a furnace. UnmarshalNbt and SetChunk must be
// called before any other methods.
func NewFurnaceTileEntity() ITileEntity {
	return createFurnaceInventory(nil)
}

func createFurnaceInventory(instance *BlockInstance) *blockInventory {
	return newBlockInventory(
		instance,
		NewFurnaceInventory(),
		false,
		InvTypeIdFurnace,
	)
}

func (aspect *FurnaceAspect) InventoryClick(instance *BlockInstance, player IPlayerClient, click *Click) {

	aspect.InventoryAspect.InventoryClick(instance, player, click)

	blockInv, furnaceInv := aspect.furnaceInventory(instance)
	if furnaceInv == nil {
		// Invalid or missing inventory.
		return
	}

	aspect.updateBlock(instance, blockInv, furnaceInv.IsLit())
}

func (aspect *FurnaceAspect) Tick(instance *BlockInstance) bool {
	blockInv, furnaceInv := aspect.furnaceInventory(instance)
	if furnaceInv == nil {
		// Invalid or missing inventory.
		return false
	}

	furnaceInv.Tick()

	currentState := furnaceInv.IsLit()

	aspect.updateBlock(instance, blockInv, currentState)

	return furnaceInv.IsLit()
}

func (aspect *FurnaceAspect) furnaceInventory(instance *BlockInstance) (blockInv *blockInventory, furnaceInv *FurnaceInventory) {

	blockInv = aspect.InventoryAspect.blockInv(instance, false)
	if blockInv == nil {
		return nil, nil
	}

	furnaceInv, ok := blockInv.inv.(*FurnaceInventory)
	if !ok {
		return nil, nil
	}

	return
}

func (aspect *FurnaceAspect) updateBlock(instance *BlockInstance, blockInv *blockInventory, currentState bool) {
	// The prior state is determined by the ID of the aspect called to handle the
	// current state.
	priorState := aspect.blockAttrs.id == aspect.Active

	if priorState != currentState {
		var newBlockId BlockId
		if currentState {
			newBlockId = aspect.Active
			instance.Chunk.AddActiveBlockIndex(instance.Index)
		} else {
			newBlockId = aspect.Inactive
		}
		instance.Chunk.SetBlockByIndex(instance.Index, newBlockId, instance.Data)
		instance.Chunk.SetTileEntity(instance.Index, blockInv)
	}
}
