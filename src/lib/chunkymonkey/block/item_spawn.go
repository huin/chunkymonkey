package block

import (
	"log"

	"chunkymonkey/item"
	"chunkymonkey/itemtype"
	. "chunkymonkey/types"
)

// spawnItemInBlock creates an item in a block. It must be run within
// instance.Chunk's goroutine.
func spawnItemInBlock(instance *BlockInstance, itemType *itemtype.ItemType, count ItemCount, data ItemData) {
	rand := instance.Chunk.GetRand()
	position := instance.BlockLoc.ToAbsXyz()
	position.X += AbsCoord(blockItemSpawnFromEdge + rand.Float64()*(1-2*blockItemSpawnFromEdge))
	position.Y += AbsCoord(blockItemSpawnFromEdge)
	position.Z += AbsCoord(blockItemSpawnFromEdge + rand.Float64()*(1-2*blockItemSpawnFromEdge))
	instance.Chunk.AddSpawner(
		item.NewItem(
			itemType, count, data,
			position,
			&AbsVelocity{0, 0, 0},
		),
	)
}

type blockDropItem struct {
	DroppedItem ItemTypeId
	Probability byte // Probabilities specified as a percentage
	Count       ItemCount
	CopyData    bool
}

func (bdi *blockDropItem) drop(instance *BlockInstance) {
	itemType, ok := instance.Chunk.GetItemType(bdi.DroppedItem)

	if !ok {
		log.Printf(
			"Warning: tried to create item with type ID #%d - "+
				"but no such item type is defined. block and item "+
				"definitions out of sync?",bdi.DroppedItem)

		return
	}

	var itemData ItemData
	if bdi.CopyData {
		itemData = ItemData(instance.Data)
	} else {
		itemData = 0
	}

	for i := bdi.Count; i > 0; i-- {
		spawnItemInBlock(instance, itemType, 1, itemData)
	}
}
