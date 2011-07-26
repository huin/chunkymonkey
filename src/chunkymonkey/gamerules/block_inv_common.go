package gamerules

// InventoryAspect is the common behaviour for blocks that have inventory.
type InventoryAspect struct {
	StandardAspect
	name                 string
	createBlockInventory func(instance *BlockInstance) *blockInventory
}

func (aspect *InventoryAspect) Name() string {
	return aspect.name
}

func (aspect *InventoryAspect) Interact(instance *BlockInstance, player IPlayerClient) {
	blkInv := aspect.blockInv(instance, true)
	if blkInv != nil {
		blkInv.AddSubscriber(player)
	}
}

func (aspect *InventoryAspect) InventoryClick(instance *BlockInstance, player IPlayerClient, click *Click) {
	blkInv := aspect.blockInv(instance, false)
	if blkInv != nil {
		blkInv.Click(player, click)
	} else {
		// No inventory to act on (shouldn't happen, normally).
		player.InventoryTxState(blkInv.instance.BlockLoc, click.TxId, false)
		player.InventoryCursorUpdate(instance.BlockLoc, click.Cursor)
		return
	}
}

func (aspect *InventoryAspect) InventoryUnsubscribed(instance *BlockInstance, player IPlayerClient) {
	blkInv := aspect.blockInv(instance, false)
	if blkInv != nil {
		blkInv.RemoveSubscriber(player.GetEntityId())
	}
}

func (aspect *InventoryAspect) Destroy(instance *BlockInstance) {
	blkInv := aspect.blockInv(instance, false)
	if blkInv != nil {
		blkInv.EjectItems()
		blkInv.Destroyed()
	}

	aspect.StandardAspect.Destroy(instance)
}

func (aspect *InventoryAspect) blockInv(instance *BlockInstance, create bool) *blockInventory {
	blkInv, ok := instance.Chunk.BlockExtra(instance.Index).(*blockInventory)
	if !ok && create {
		blkInv = aspect.createBlockInventory(instance)
		instance.Chunk.SetBlockExtra(instance.Index, blkInv)
	}

	return blkInv
}
