package gamerules

import (
	"math/rand"

	. "chunkymonkey/types"
)

// The distance from the edge of a block that items spawn at in fractional
// blocks.
const blockItemSpawnFromEdge = 4.0 / PixelsPerBlock

// The interface required of a chunk by block behaviour.
type IChunkBlock interface {
	Rand() *rand.Rand
	ItemType(itemTypeId ItemTypeId) (itemType *ItemType, ok bool)
	AddEntity(s INonPlayerEntity)
	SetBlockByIndex(blockIndex BlockIndex, blockId BlockId, blockData byte)
	TileEntity(blockIndex BlockIndex) ITileEntity
	SetTileEntity(blockIndex BlockIndex, extra ITileEntity)
	AddOnUnsubscribe(entityId EntityId, observer IUnsubscribed)
	RemoveOnUnsubscribe(entityId EntityId, observer IUnsubscribed)

	// AddActiveBlock flags a block in any chunk as active.
	AddActiveBlock(blockXyz *BlockXyz)

	// AddActiveBlockIndex flags a block in the chunk itself as active by index.
	AddActiveBlockIndex(blockIndex BlockIndex)
}

// IUnsubscribed is the interface by which blocks (and potentially other
// things) can register themselves to be called when a player unsubscribes from
// a chunk.
type IUnsubscribed interface {
	Unsubscribed(entityId EntityId)
}

// BlockInstance represents the instance of a block within a chunk. It is used
// to pass context to a IBlockAspect method call. A BlockInstance belongs to
// the chunk that creates it - a copy must be made if a block aspect needs to
// persist the value of one.
type BlockInstance struct {
	Chunk     IChunkBlock
	BlockLoc  BlockXyz
	SubLoc    SubChunkXyz
	Index     BlockIndex
	BlockType *BlockType
	// Note that only the lower nibble of data is stored.
	Data byte
}

// Defines the behaviour of a block.
type IBlockAspect interface {
	setAttrs(blockAttrs *BlockAttrs)

	// Name is currently used purely for the serialization of aspect
	// configuration data.
	Name() string

	// Check tests that the block aspect has been configured correctly,
	// returning nil if it is correct.
	Check() error

	// Hit is called when the player hits a block.
	Hit(instance *BlockInstance, player IPlayerClient, digStatus DigStatus) (destroyed bool)

	// Interact is called when a player right-clicks a block.
	Interact(instance *BlockInstance, player IPlayerClient)

	// InventoryClick is called when the player clicked on a slot inside the
	// inventory for the block (assuming it still has one).
	InventoryClick(instance *BlockInstance, player IPlayerClient, click *Click)

	// InventoryUnsubscribed is called when the player closes the window for the
	// inventory for the block (assuming it still has one).
	InventoryUnsubscribed(instance *BlockInstance, player IPlayerClient)

	// Destroy is called when the block is destroyed by a player hitting it.
	// TODO And in other situations, maybe?
	Destroy(instance *BlockInstance)

	// Tick tells the aspect to run the block for a tick. It should return false
	// if the block should not tick again.
	Tick(instance *BlockInstance) bool
}
