package player

import (
	"log"

	"chunkymonkey/gamerules"
	. "chunkymonkey/types"
)

// shardRef holds a reference to a shard connection and context for the number
// of subscribed chunks inside the shard.
type shardRef struct {
	shard gamerules.IPlayerShardClient
	count int
}

// chunkSubscriptions is part of the player frontend code, and maintains:
// * the shards to be connected to,
// * the chunks that to be subscribed to (via their shardClients),
// * moving the player from one shard to another.
type chunkSubscriptions struct {
	player         *Player
	shardConnecter gamerules.IShardConnecter
	playerClient   gamerules.IPlayerClient
	entityId       EntityId
	curShardLoc    ShardXz                      // Shard the player is currently in.
	curChunkLoc    ChunkXz                      // Chunk the player is currently in.
	curShard       gamerules.IPlayerShardClient // Shard the player is hosted on.
	shardClients   map[uint64]*shardRef         // Connections to shards.
}

func (sub *chunkSubscriptions) Init(player *Player) {
	sub.player = player
	sub.playerClient = &player.playerClient
	sub.shardConnecter = player.shardConnecter
	sub.entityId = player.EntityId
	sub.curShardLoc = player.position.ToShardXz()
	sub.curChunkLoc = player.position.ToChunkXz()
	sub.shardClients = make(map[uint64]*shardRef)

	initialChunkLocs := orderedChunkSquare(sub.curChunkLoc, ChunkRadius)
	sub.subscribeToChunks(sub.curChunkLoc, initialChunkLocs)

	sub.curShard = sub.shardClients[sub.curShardLoc.Key()].shard
	sub.curShard.ReqAddPlayerData(
		sub.curChunkLoc,
		player.name,
		player.position,
		*player.look.ToLookBytes(),
		player.getHeldItemTypeId(),
	)
}

// Move should be called as the player moves around the world. It replicates
// the player's position to the chunk they are in, and adjusts chunk
// subscriptions as necessary. Returns true if the new location is not yet
// subscribed to by a chunk, indicating that the player will receive a
// notifyChunkLoad when that chunk has been sent to the client.
func (sub *chunkSubscriptions) Move(newLoc *AbsXyz) (notify bool) {
	newChunkLoc := newLoc.ToChunkXz()
	if newChunkLoc.X != sub.curChunkLoc.X || newChunkLoc.Z != sub.curChunkLoc.Z {
		notify = sub.moveToChunk(newChunkLoc, newLoc)

		newShardLoc := newLoc.ToShardXz()
		if newShardLoc.X != sub.curShardLoc.X || newShardLoc.Z != sub.curShardLoc.Z {
			sub.moveToShard(newShardLoc)
		}
	} else {
		sub.curShard.ReqSetPlayerPosition(sub.curChunkLoc, *newLoc)
	}

	return
}

// Close closes down all shard connections. Use when the player is
// disconnected.
func (sub *chunkSubscriptions) Close() {
	curShardLoc := sub.curChunkLoc.ToShardXz()
	if ref, ok := sub.shardClients[curShardLoc.Key()]; ok {
		ref.shard.ReqRemovePlayerData(sub.curChunkLoc, true)
	}

	for key, ref := range sub.shardClients {
		ref.shard.Disconnect()
		delete(sub.shardClients, key)
	}
}

// CurrentShardClient is a convenience function to get a client shard
// connection for the player's current shard. ok=false if no such connection
// exists.
func (sub *chunkSubscriptions) CurrentShardClient() (conn gamerules.IPlayerShardClient, ok bool) {
	curShardLoc := sub.curChunkLoc.ToShardXz()
	shardRef, ok := sub.shardClients[curShardLoc.Key()]
	return shardRef.shard, ok
}

// ShardClientForBlockXyz is a convenience function to get the correct shard
// connection and the ChunkXz within that chunk for a given BlockXyz position.
// Returns ok = false if there is no open connection for that shard. Note that
// this doesn't check if the chunk actually exists.
func (sub *chunkSubscriptions) ShardClientForBlockXyz(blockLoc *BlockXyz) (conn gamerules.IPlayerShardClient, chunkLoc *ChunkXz, ok bool) {

	chunkLoc = blockLoc.ToChunkXz()

	shardLoc := chunkLoc.ToShardXz()
	ref, ok := sub.shardClients[shardLoc.Key()]
	if !ok {
		return
	}

	conn = ref.shard
	ok = true

	return
}

// ShardClientForChunkXz is a convenience function to get the correct shard
// connection for a given ChunkXz position. Returns ok = false if there is no
// open connection for that shard. Note that this doesn't check if the chunk
// actually exists.
func (sub *chunkSubscriptions) ShardClientForChunkXz(chunkLoc *ChunkXz) (conn gamerules.IPlayerShardClient, ok bool) {

	shardLoc := chunkLoc.ToShardXz()
	ref, ok := sub.shardClients[shardLoc.Key()]
	if !ok {
		return
	}

	conn = ref.shard
	ok = true

	return
}

// subscribeToChunks connects to shards and subscribes to chunks for the chunk
// locations given.
func (sub *chunkSubscriptions) subscribeToChunks(destLoc ChunkXz, chunkLocs []ChunkXz) (notify bool) {
	for _, chunkLoc := range chunkLocs {
		shardLoc := chunkLoc.ToShardXz()
		shardKey := shardLoc.Key()
		ref, ok := sub.shardClients[shardKey]
		if !ok {
			ref = &shardRef{
				shard: sub.shardConnecter.PlayerShardConnect(sub.entityId, sub.playerClient, shardLoc),
				count: 0,
			}
			sub.shardClients[shardKey] = ref
		}

		isDestChunk := chunkLoc.X == destLoc.X && chunkLoc.Z == destLoc.Z
		notify = notify || isDestChunk
		ref.shard.ReqSubscribeChunk(chunkLoc, isDestChunk)
		ref.count++
	}

	return
}

// unsubscribeFromChunks unsubscribes from chunks for the chunk locations
// given, and disconnects from shards where there are no subscribed chunks.
func (sub *chunkSubscriptions) unsubscribeFromChunks(chunkLocs []ChunkXz) {
	for _, chunkLoc := range chunkLocs {
		shardLoc := chunkLoc.ToShardXz()
		shardKey := shardLoc.Key()
		if ref, ok := sub.shardClients[shardKey]; ok {
			ref.shard.ReqUnsubscribeChunk(chunkLoc)
			ref.count--
			if ref.count <= 0 {
				ref.shard.Disconnect()
				delete(sub.shardClients, shardKey)
			}
		} else {
			// Odd - we don't have a shard connection for that chunk.
			log.Printf("chunkSubscriptions.unsubscribeFromChunks() attempted to "+
				"unsubscribe from chunk @%v in unconnected shard @%v.", chunkLoc, shardLoc)
		}
	}
}

// moveToChunk subscribes to chunks that are newly in range, and unsubscribes
// to those that have just left.
func (sub *chunkSubscriptions) moveToChunk(newChunkLoc ChunkXz, newLoc *AbsXyz) (notify bool) {
	addChunkLocs := squareDifference(newChunkLoc, sub.curChunkLoc, ChunkRadius)
	notify = sub.subscribeToChunks(newChunkLoc, addChunkLocs)

	newShardLoc := newChunkLoc.ToShardXz()
	if ref, ok := sub.shardClients[newShardLoc.Key()]; ok {
		ref.shard.ReqAddPlayerData(
			newChunkLoc,
			sub.player.name,
			sub.player.position,
			*sub.player.look.ToLookBytes(),
			sub.player.getHeldItemTypeId(),
		)
	}

	curShardLoc := sub.curChunkLoc.ToShardXz()
	if ref, ok := sub.shardClients[curShardLoc.Key()]; ok {
		ref.shard.ReqRemovePlayerData(sub.curChunkLoc, false)
	}

	delChunkLocs := squareDifference(sub.curChunkLoc, newChunkLoc, ChunkRadius)
	sub.unsubscribeFromChunks(delChunkLocs)

	sub.curChunkLoc = newChunkLoc

	return
}

func (sub *chunkSubscriptions) moveToShard(newShardLoc ShardXz) {
	// The new current shard is assumed to be present in sub.shardClients already.
	sub.curShard = sub.shardClients[newShardLoc.Key()].shard
	sub.curShardLoc = newShardLoc
}

// squareDifference computes the ChunkXz values that are in "square radius" of
// centerA, but not in "square radius" of centerB.
//
// E.g, input squares:
// centerA={1,1}, centerB={2,2}, radius = 1
// A = in square A, B = in square B, C = in both
//
// AAA
// ACCB
// ACCB
//  BBB
//
// Results will be:
//
// AAA
// A
// A
func squareDifference(centerA, centerB ChunkXz, radius ChunkCoord) []ChunkXz {
	// TODO Currently this is the exact same slow simple O(n²) algorithm used in
	// the test to produce the expected result. This could be optimized somewhat.
	// Even if a general optimization is too much effort, the trivial cases of
	// moving ±1X and/or ±1Z are the very common cases, and the simple dumb
	// algorithm can be used otherwise.
	areaEdgeSize := radius*2 + 1
	result := make([]ChunkXz, 0, areaEdgeSize)
	for x := centerA.X - radius; x <= centerA.X+radius; x++ {
		for z := centerA.Z - radius; z <= centerA.Z+radius; z++ {
			if x >= centerB.X-radius && x <= centerB.X+radius && z >= centerB.Z-radius && z <= centerB.Z+radius {
				// {x, z} is within square B. Don't include this.
				continue
			}
			result = append(result, ChunkXz{x, z})
		}
	}
	return result
}

// orderedChunkSquare creates a slice of chunk locations in a square centered
// on `center`, with sides `radius` chunks away from the center. The chunk
// locations are output in approx this order for radius=2 (where lower numbered
// chunks are earlier):
//
// 33333
// 32223
// 32123
// 32223
// 33333
func orderedChunkSquare(center ChunkXz, radius ChunkCoord) (locs []ChunkXz) {
	areaEdgeSize := radius*2 + 1
	locs = make([]ChunkXz, areaEdgeSize*areaEdgeSize)
	locs[0] = center
	index := 1
	for curRadius := ChunkCoord(1); curRadius <= radius; curRadius++ {
		xMin := ChunkCoord(-curRadius + center.X)
		xMax := ChunkCoord(curRadius + center.X)
		zMin := ChunkCoord(-curRadius + center.Z)
		zMax := ChunkCoord(curRadius + center.Z)

		// Northern and southern rows of chunks.
		for x := xMin; x <= xMax; x++ {
			locs[index] = ChunkXz{x, zMin}
			index++
			locs[index] = ChunkXz{x, zMax}
			index++
		}

		// Eastern and western columns (except for where they intersect the
		// north and south rows).
		for z := zMin + 1; z < zMax; z++ {
			locs[index] = ChunkXz{xMin, z}
			index++
			locs[index] = ChunkXz{xMax, z}
			index++
		}
	}
	return
}
