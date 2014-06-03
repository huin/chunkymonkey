package shardserver

import (
	"chunkymonkey/gamerules"
	. "chunkymonkey/types"
)

// localPlayerShardClient implements IPlayerShardClient for LocalShardManager.
type localPlayerShardClient struct {
	entityId EntityId
	player   gamerules.IPlayerClient
	shard    *ChunkShard
}

func newLocalPlayerShardClient(entityId EntityId, player gamerules.IPlayerClient, shard *ChunkShard) *localPlayerShardClient {
	return &localPlayerShardClient{
		entityId: entityId,
		player:   player,
		shard:    shard,
	}
}

func (conn *localPlayerShardClient) Disconnect() {
	// TODO This inefficiently unsubscribes from all chunks, even if not
	// subscribed to.
	conn.shard.enqueueAllChunks(func(chunk *Chunk) {
		chunk.reqUnsubscribeChunk(conn.entityId, false)
	})
}

func (conn *localPlayerShardClient) ReqSubscribeChunk(chunkLoc ChunkXz, notify bool) {
	conn.shard.enqueueOnChunk(chunkLoc, func(chunk *Chunk) {
		chunk.reqSubscribeChunk(conn.entityId, conn.player, notify)
	})
}

func (conn *localPlayerShardClient) ReqUnsubscribeChunk(chunkLoc ChunkXz) {
	conn.shard.enqueueOnChunk(chunkLoc, func(chunk *Chunk) {
		chunk.reqUnsubscribeChunk(conn.entityId, true)
	})
}

func (conn *localPlayerShardClient) ReqMulticastPlayers(chunkLoc ChunkXz, exclude EntityId, packet []byte) {
	conn.shard.enqueueOnChunk(chunkLoc, func(chunk *Chunk) {
		chunk.reqMulticastPlayers(exclude, packet)
	})
}

func (conn *localPlayerShardClient) ReqAddPlayerData(chunkLoc ChunkXz, name string, position AbsXyz, look LookBytes, held ItemTypeId) {
	conn.shard.enqueueOnChunk(chunkLoc, func(chunk *Chunk) {
		chunk.reqAddPlayerData(conn.entityId, name, position, look, held)
	})
}

func (conn *localPlayerShardClient) ReqRemovePlayerData(chunkLoc ChunkXz, isDisconnect bool) {
	conn.shard.enqueueOnChunk(chunkLoc, func(chunk *Chunk) {
		chunk.reqRemovePlayerData(conn.entityId, isDisconnect)
	})
}

func (conn *localPlayerShardClient) ReqSetPlayerPosition(chunkLoc ChunkXz, position AbsXyz) {
	conn.shard.enqueueOnChunk(chunkLoc, func(chunk *Chunk) {
		chunk.reqSetPlayerPosition(conn.entityId, position)
	})
}

func (conn *localPlayerShardClient) ReqSetPlayerLook(chunkLoc ChunkXz, look LookBytes) {
	conn.shard.enqueueOnChunk(chunkLoc, func(chunk *Chunk) {
		chunk.reqSetPlayerLook(conn.entityId, look)
	})
}

func (conn *localPlayerShardClient) ReqHitBlock(held gamerules.Slot, target BlockXyz, digStatus DigStatus, face Face) {
	chunkLoc := target.ToChunkXz()

	conn.shard.enqueueOnChunk(*chunkLoc, func(chunk *Chunk) {
		chunk.reqHitBlock(conn.player, held, digStatus, &target, face)
	})
}

func (conn *localPlayerShardClient) ReqInteractBlock(held gamerules.Slot, target BlockXyz, face Face) {
	chunkLoc := target.ToChunkXz()

	conn.shard.enqueueOnChunk(*chunkLoc, func(chunk *Chunk) {
		chunk.reqInteractBlock(conn.player, held, &target, face)
	})
}

func (conn *localPlayerShardClient) ReqPlaceItem(target BlockXyz, slot gamerules.Slot) {
	chunkLoc, _ := target.ToChunkLocal()

	conn.shard.enqueueOnChunk(*chunkLoc, func(chunk *Chunk) {
		chunk.reqPlaceItem(conn.player, &target, &slot)
	})
}

func (conn *localPlayerShardClient) ReqTakeItem(chunkLoc ChunkXz, entityId EntityId) {
	conn.shard.enqueueOnChunk(chunkLoc, func(chunk *Chunk) {
		chunk.reqTakeItem(conn.player, entityId)
	})
}

func (conn *localPlayerShardClient) ReqDropItem(content gamerules.Slot, position AbsXyz, velocity AbsVelocity, pickupImmunity Ticks) {
	chunkLoc := position.ToChunkXz()
	conn.shard.enqueueOnChunk(chunkLoc, func(chunk *Chunk) {
		chunk.reqDropItem(conn.player, &content, &position, &velocity, pickupImmunity)
	})
}

func (conn *localPlayerShardClient) ReqInventoryClick(block BlockXyz, click gamerules.Click) {
	chunkLoc := block.ToChunkXz()
	conn.shard.enqueueOnChunk(*chunkLoc, func(chunk *Chunk) {
		chunk.reqInventoryClick(conn.player, &block, &click)
	})
}

func (conn *localPlayerShardClient) ReqInventoryUnsubscribed(block BlockXyz) {
	chunkLoc := block.ToChunkXz()
	conn.shard.enqueueOnChunk(*chunkLoc, func(chunk *Chunk) {
		chunk.reqInventoryUnsubscribed(conn.player, &block)
	})
}
