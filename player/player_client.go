package player

import (
	"bytes"
	"chunkymonkey/gamerules"
	"chunkymonkey/proto"
	. "chunkymonkey/types"
)

// playerClient presents a thread-safe interface for interacting with a Player
// object.
type playerClient struct {
	player *Player
}

func (p *playerClient) Init(player *Player) {
	p.player = player
}

func (p *playerClient) GetEntityId() EntityId {
	return p.player.EntityId
}

func (p *playerClient) TransmitPacket(packet []byte) {
	p.player.TransmitPacket(packet)
}

func (p *playerClient) NotifyChunkLoad() {
	p.player.Enqueue(func(_ *Player) {
		p.player.notifyChunkLoad()
	})
}

func (p *playerClient) InventorySubscribed(block BlockXyz, invTypeId InvTypeId, slots []proto.WindowSlot) {
	p.player.Enqueue(func(_ *Player) {
		p.player.inventorySubscribed(&block, invTypeId, slots)
	})
}

func (p *playerClient) InventorySlotUpdate(block BlockXyz, slot gamerules.Slot, slotId SlotId) {
	p.player.Enqueue(func(_ *Player) {
		p.player.inventorySlotUpdate(&block, &slot, slotId)
	})
}

func (p *playerClient) InventoryProgressUpdate(block BlockXyz, prgBarId PrgBarId, value PrgBarValue) {
	p.player.Enqueue(func(_ *Player) {
		p.player.inventoryProgressUpdate(&block, prgBarId, value)
	})
}

func (p *playerClient) InventoryCursorUpdate(block BlockXyz, cursor gamerules.Slot) {
	p.player.Enqueue(func(_ *Player) {
		p.player.inventoryCursorUpdate(&block, &cursor)
	})
}

func (p *playerClient) InventoryTxState(block BlockXyz, txId TxId, accepted bool) {
	p.player.Enqueue(func(_ *Player) {
		p.player.inventoryTxState(&block, txId, accepted)
	})
}

func (p *playerClient) InventoryUnsubscribed(block BlockXyz) {
	p.player.Enqueue(func(_ *Player) {
		p.player.inventoryUnsubscribed(&block)
	})
}

func (p *playerClient) PlaceHeldItem(target BlockXyz, wasHeld gamerules.Slot) {
	p.player.Enqueue(func(_ *Player) {
		p.player.placeHeldItem(&target, &wasHeld)
	})
}

func (p *playerClient) OfferItem(fromChunk ChunkXz, entityId EntityId, item gamerules.Slot) {
	p.player.Enqueue(func(_ *Player) {
		p.player.offerItem(&fromChunk, entityId, &item)
	})
}

func (p *playerClient) GiveItemAtPosition(atPosition AbsXyz, item gamerules.Slot) {
	p.player.Enqueue(func(_ *Player) {
		p.player.giveItem(&atPosition, &item)
	})
}

func (p *playerClient) GiveItem(item gamerules.Slot) {
	p.player.Enqueue(func(_ *Player) {
		p.player.giveItem(&p.player.position, &item)
	})
}

func (p *playerClient) EchoMessage(msg string) {
	p.player.Enqueue(func(_ *Player) {
		buf := new(bytes.Buffer)
		proto.WriteChatMessage(buf, msg)
		p.TransmitPacket(buf.Bytes())
	})
}

func (p *playerClient) PositionLook() (AbsXyz, LookDegrees) {
	posChan := make(chan AbsXyz)
	lookChan := make(chan LookDegrees)

	p.player.Enqueue(func(player *Player) {
		posChan <- player.position
		lookChan <- player.look
		close(posChan)
		close(lookChan)
	})

	pos := <-posChan
	look := <-lookChan

	return pos, look
}

func (p *playerClient) SetPositionLook(pos AbsXyz, look LookDegrees) {
	p.player.Enqueue(func(player *Player) {
		player.setPositionLook(pos, look)
	})
}
