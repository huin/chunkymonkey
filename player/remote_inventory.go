package player

import (
	"chunkymonkey/gamerules"
	"chunkymonkey/proto"
	. "chunkymonkey/types"
)

type RemoteInventory struct {
	blockLoc   BlockXyz
	chunkSubs  *chunkSubscriptions
	slots      []proto.WindowSlot
	subscriber gamerules.IInventorySubscriber
}

func NewRemoteInventory(block *BlockXyz, chunkSubs *chunkSubscriptions, slots []proto.WindowSlot) *RemoteInventory {
	return &RemoteInventory{
		blockLoc:   *block,
		chunkSubs:  chunkSubs,
		slots:      slots,
		subscriber: nil,
	}
}

func (inv *RemoteInventory) IsForBlock(block *BlockXyz) bool {
	return inv.blockLoc.X == block.X && inv.blockLoc.Y == block.Y && inv.blockLoc.Z == block.Z
}

func (inv *RemoteInventory) slotUpdate(slot *gamerules.Slot, slotId SlotId) {
	if inv.subscriber != nil {
		inv.subscriber.SlotUpdate(slot, slotId)
	}
}

func (inv *RemoteInventory) progressUpdate(prgBarId PrgBarId, value PrgBarValue) {
	if inv.subscriber != nil {
		inv.subscriber.ProgressUpdate(prgBarId, value)
	}
}

func (inv *RemoteInventory) Close() {
	shard, _, ok := inv.chunkSubs.ShardClientForBlockXyz(&inv.blockLoc)

	if ok {
		shard.ReqInventoryUnsubscribed(inv.blockLoc)
	}
}

// The following methods are to implement window.IInventory.

func (inv *RemoteInventory) NumSlots() SlotId {
	return SlotId(len(inv.slots))
}

func (inv *RemoteInventory) Click(click *gamerules.Click) (txState TxState) {
	shard, _, ok := inv.chunkSubs.ShardClientForBlockXyz(&inv.blockLoc)

	if ok {
		shard.ReqInventoryClick(inv.blockLoc, *click)
	}

	return TxStateDeferred
}

func (inv *RemoteInventory) SetSubscriber(subscriber gamerules.IInventorySubscriber) {
	inv.subscriber = subscriber
}

func (inv *RemoteInventory) WriteProtoSlots(slots []proto.WindowSlot) {
	// Note that this only produces accurate results before any slot updates come
	// through. inv.slots contains only a snapshot of the state when the window
	// opened.
	copy(slots, inv.slots)
}
