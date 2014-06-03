package shardserver

import (
	"sync"

	"github.com/huin/chunkymonkey/chunkstore"
	"github.com/huin/chunkymonkey/entity"
	"github.com/huin/chunkymonkey/gamerules"
	. "github.com/huin/chunkymonkey/types"
)

// LocalShardManager contains all chunk shards and can look them up. It
// implements IShardConnecter and is for use in hosting all shards in the local
// process.
type LocalShardManager struct {
	entityMgr  *entity.EntityManager
	chunkStore chunkstore.IChunkStore
	shards     map[uint64]*ChunkShard
	lock       sync.Mutex
}

func NewLocalShardManager(chunkStore chunkstore.IChunkStore, entityMgr *entity.EntityManager) *LocalShardManager {
	return &LocalShardManager{
		entityMgr:  entityMgr,
		chunkStore: chunkStore,
		shards:     make(map[uint64]*ChunkShard),
	}
}

func (mgr *LocalShardManager) getShard(loc ShardXz, create bool) *ChunkShard {
	shardKey := loc.Key()
	if shard, ok := mgr.shards[shardKey]; ok {
		// Shard already exists.
		return shard
	}

	if !create {
		return nil
	}

	// Create shard.
	shard := NewChunkShard(mgr, mgr.chunkStore, mgr.entityMgr, loc)
	mgr.shards[shardKey] = shard
	go shard.serve()

	return shard
}

func (mgr *LocalShardManager) PlayerShardConnect(entityId EntityId, player gamerules.IPlayerClient, shardLoc ShardXz) gamerules.IPlayerShardClient {
	mgr.lock.Lock()
	defer mgr.lock.Unlock()

	shard := mgr.getShard(shardLoc, true)
	return newLocalPlayerShardClient(entityId, player, shard)
}

func (mgr *LocalShardManager) ShardShardConnect(shardLoc ShardXz) gamerules.IShardShardClient {
	mgr.lock.Lock()
	defer mgr.lock.Unlock()

	shard := mgr.getShard(shardLoc, false)

	if shard == nil {
		return nil
	}

	return newLocalShardShardClient(shard)
}

// TODO remove Enqueue* methods

// EnqueueAllChunks runs a given function on all loaded chunks.
func (mgr *LocalShardManager) EnqueueAllChunks(fn func(chunk *Chunk)) {
	mgr.lock.Lock()
	defer mgr.lock.Unlock()

	for _, shard := range mgr.shards {
		shard.enqueueAllChunks(fn)
	}
}

// EnqueueOnChunk runs a function on the chunk at the given location. If the
// chunk does not exist, it does nothing.
func (mgr *LocalShardManager) EnqueueOnChunk(loc ChunkXz, fn func(chunk *Chunk)) {
	mgr.lock.Lock()
	defer mgr.lock.Unlock()

	shard := mgr.getShard(loc.ToShardXz(), true)
	shard.enqueueOnChunk(loc, fn)
}
