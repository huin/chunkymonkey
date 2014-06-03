package gamerules

import (
	"fmt"

	. "chunkymonkey/types"
)

func makeStandardAspect() (aspect IBlockAspect) {
	return &StandardAspect{}
}

// Behaviour of a "standard" block. A StandardAspect block is one that is
// diggable, and drops items in a simple manner. StandardAspect blocks do not
// use block metadata.
type StandardAspect struct {
	blockAttrs *BlockAttrs
	// Items, up to one of which will potentially spawn when block destroyed.
	DroppedItems []blockDropItem
	BreakOn      DigStatus
}

func (aspect *StandardAspect) setAttrs(blockAttrs *BlockAttrs) {
	aspect.blockAttrs = blockAttrs
}

func (aspect *StandardAspect) Name() string {
	return "Standard"
}

func (aspect *StandardAspect) Check() error {
	for i := range aspect.DroppedItems {
		if err := aspect.DroppedItems[i].check(); err != nil {
			return fmt.Errorf("block %q: %v", aspect.blockAttrs.Name, err)
		}
	}
	return nil
}

func (aspect *StandardAspect) Hit(instance *BlockInstance, player IPlayerClient, digStatus DigStatus) (destroyed bool) {
	if aspect.BreakOn != digStatus {
		return
	}

	destroyed = true

	return
}

func (aspect *StandardAspect) Interact(instance *BlockInstance, player IPlayerClient) {
}

func (aspect *StandardAspect) InventoryClick(instance *BlockInstance, player IPlayerClient, click *Click) {
}

func (aspect *StandardAspect) InventoryUnsubscribed(instance *BlockInstance, player IPlayerClient) {
}

func (aspect *StandardAspect) Destroy(instance *BlockInstance) {
	if len(aspect.DroppedItems) > 0 {
		rand := instance.Chunk.Rand()
		// Possibly drop item(s)
		r := byte(rand.Intn(100))
		for _, dropItem := range aspect.DroppedItems {
			if dropItem.Probability > r {
				dropItem.drop(instance.Chunk, instance.BlockLoc, instance.Data)
				break
			}
			r -= dropItem.Probability
		}
	}
}

func (aspect *StandardAspect) Tick(instance *BlockInstance) bool {
	return false
}
