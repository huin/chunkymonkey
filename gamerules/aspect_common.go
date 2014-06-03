package gamerules

import (
	"fmt"

	. "chunkymonkey/types"
)

// spawnItemInBlock creates an item in a block. It must be run within
// instance.Chunk's goroutine.
func spawnItemInBlock(chunk IChunkBlock, blockLoc BlockXyz, itemTypeId ItemTypeId, count ItemCount, data ItemData) {
	rand := chunk.Rand()
	position := blockLoc.ToAbsXyz()
	position.X += AbsCoord(blockItemSpawnFromEdge + rand.Float64()*(1-2*blockItemSpawnFromEdge))
	position.Y += AbsCoord(blockItemSpawnFromEdge)
	position.Z += AbsCoord(blockItemSpawnFromEdge + rand.Float64()*(1-2*blockItemSpawnFromEdge))
	chunk.AddEntity(
		NewItem(
			itemTypeId, count, data,
			position,
			&AbsVelocity{0, 0, 0},
			0,
		),
	)
}

type blockDropItem struct {
	DroppedItem ItemTypeId
	Probability byte // Probabilities specified as a percentage
	Count       ItemCount
	CopyData    bool
}

func (bdi *blockDropItem) drop(chunk IChunkBlock, blockLoc BlockXyz, blockData byte) {
	var itemData ItemData
	if !bdi.CopyData {
		itemData = 0
	} else {
		itemData = ItemData(blockData)
	}

	spawnItemInBlock(chunk, blockLoc, bdi.DroppedItem, bdi.Count, itemData)
}

func (bdi *blockDropItem) check() error {
	if _, ok := Items[bdi.DroppedItem]; !ok {
		return fmt.Errorf("dropped item type %d does not exist", bdi.DroppedItem)
	}

	if bdi.Count <= 0 {
		return fmt.Errorf("dropped item has Count %d", bdi.Count)
	}

	return nil
}
