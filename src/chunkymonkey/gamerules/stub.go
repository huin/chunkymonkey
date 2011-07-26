package gamerules

import (
	"chunkymonkey/object"
	"chunkymonkey/proto"
	. "chunkymonkey/types"
)

// IShardConnecter is used to look up shards and connect to them.
type IShardConnecter interface {
	// PlayerShardConnect makes a connection from a player to a shard.
	PlayerShardConnect(entityId EntityId, player IPlayerClient, shardLoc ShardXz) IPlayerShardClient

	// ShardShardConnect makes a connection from one shard to another.
	// TODO Consider making this package-private to shardserver.
	ShardShardConnect(shardLoc ShardXz) IShardShardClient
}

// IPlayerShardClient is the interface by which shards can be communicated to by
// player frontend code.
type IPlayerShardClient interface {
	// Removes connection to shard, and removes all subscriptions to chunks in
	// the shard. Note that this does *not* send packets to tell the client to
	// unload the subscribed chunks.
	Disconnect()

	// The following methods are requests upon chunks.

	ReqSubscribeChunk(chunkLoc ChunkXz, notify bool)

	ReqUnsubscribeChunk(chunkLoc ChunkXz)

	ReqMulticastPlayers(chunkLoc ChunkXz, exclude EntityId, packet []byte)

	ReqAddPlayerData(chunkLoc ChunkXz, name string, position AbsXyz, look LookBytes, held ItemTypeId)

	ReqRemovePlayerData(chunkLoc ChunkXz, isDisconnect bool)

	ReqSetPlayerPosition(chunkLoc ChunkXz, position AbsXyz)

	ReqSetPlayerLook(chunkLoc ChunkXz, look LookBytes)

	// ReqHitBlock requests that the targetted block be hit.
	ReqHitBlock(held Slot, target BlockXyz, digStatus DigStatus, face Face)

	// ReqHitBlock requests that the targetted block be interacted with.
	ReqInteractBlock(held Slot, target BlockXyz, face Face)

	// ReqPlaceItem requests that the item passed be placed at the given target
	// location. The shard *may* choose not to do this, but if it cannot, then it
	// *must* account for the item in some way (maybe hand it back to the player
	// or just drop it on the ground).
	ReqPlaceItem(target BlockXyz, slot Slot)

	// ReqTakeItem requests that the item with the specified entityId is given to
	// the player. The chunk doesn't have to respect this (particularly if the
	// item no longer exists).
	ReqTakeItem(chunkLoc ChunkXz, entityId EntityId)

	// ReqDropItem requests that an item be created.
	ReqDropItem(content Slot, position AbsXyz, velocity AbsVelocity, pickupImmunity Ticks)

	// ReqInventoryClick requests that the given cursor be "clicked" onto the
	// inventory. The chunk should send a replying ReqInventoryCursorUpdate to
	// reflect the new state of the cursor afterwards - in addition to any
	// ReqInventorySlotUpdate to all subscribers to the inventory.
	ReqInventoryClick(block BlockXyz, click Click)

	// ReqInventoryUnsubscribed requests that the inventory for the block be
	// unsubscribed to.
	ReqInventoryUnsubscribed(block BlockXyz)
}

// IShardShardClient provides an interface for shards to make requests against
// another shard.
// TODO Consider making this package-private to shardserver.
type IShardShardClient interface {
	Disconnect()

	ReqSetActiveBlocks(blocks []BlockXyz)

	ReqTransferEntity(loc ChunkXz, entity object.INonPlayerEntity)
}

// IGame provide an interface for interacting with and taking action on the
// game, including getting information about the game state, etc.
type IGame interface {
	// Broadcast a message to all players on the server
	BroadcastMessage(msg string)

	// Return a player from their name
	PlayerByName(name string) IPlayerClient

	// Return a player from an EntityId
	PlayerByEntityId(id EntityId) IPlayerClient

	// Return an ItemType from a numeric item. The boolean flag indicates
	// whether or not 'id' was a valid item type.
	ItemTypeById(id int) (ItemType, bool)
}

// IShardClient is the interface by which shards communicate to players on
// the frontend.
type IPlayerClient interface {
	GetEntityId() EntityId

	TransmitPacket(packet []byte)

	// NotifyChunkLoad informs Player that a chunk subscription request with
	// notify=true has completed.
	NotifyChunkLoad()

	// InventorySubscribed informs the player that an inventory has been
	// opened.
	InventorySubscribed(block BlockXyz, invTypeId InvTypeId, slots []proto.WindowSlot)

	// InventorySlotUpdate informs the player of a change to a slot in the
	// open inventory.
	InventorySlotUpdate(block BlockXyz, slot Slot, slotId SlotId)

	// InventoryProgressUpdate informs the player of a change of a progress
	// bar in a window.
	InventoryProgressUpdate(block BlockXyz, prgBarId PrgBarId, value PrgBarValue)

	// InventoryCursorUpdate informs the player of their new cursor contents.
	InventoryCursorUpdate(block BlockXyz, cursor Slot)

	// InventoryTxState requests that the player report the transaction state
	// as accepted or not. This is used by remote inventories when
	// TxStateDeferred is returned from Click.
	InventoryTxState(block BlockXyz, txId TxId, accepted bool)

	// InventorySubscribed informs the player that an inventory has been
	// closed.
	InventoryUnsubscribed(block BlockXyz)

	// PlaceHeldItem requests that the player frontend take one item from the
	// held item stack and send it in a ReqPlaceItem to the target block.  The
	// player code may *not* honour this request (e.g there might be no suitable
	// held item).
	PlaceHeldItem(target BlockXyz, wasHeld Slot)

	// OfferItem requests that the player check if it can take the item.  If
	// it can then it should ReqTakeItem from the chunk.
	OfferItem(fromChunk ChunkXz, entityId EntityId, item Slot)

	// GiveItemAtPosition requests that the player takes the item contents
	// into their inventory. If they cannot, then the player should drop the
	// item at the given position.
	GiveItemAtPosition(atPosition AbsXyz, item Slot)

	// GiveItem is a wrapper for GiveItemAtPosition that uses the player's
	// current position as the 'atPosition'.
	GiveItem(item Slot)

	// PositionLook returns the player's current position and look
	PositionLook() (AbsXyz, LookDegrees)

	// SetPositionLook changes the player's position and look
	SetPositionLook(AbsXyz, LookDegrees)

	// EchoMessage displays a message to the player
	EchoMessage(string)
}

type ICommandFramework interface {
	Prefix() string
	Process(player IPlayerClient, cmd string, game IGame)
}
