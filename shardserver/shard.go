package shardserver

import (
	"fmt"
	"log"
	"time"

	"chunkymonkey/chunkstore"
	"chunkymonkey/entity"
	"chunkymonkey/gamerules"
	. "chunkymonkey/types"
)

const chunksPerShard = ShardSize * ShardSize

// TODO Allow configuration of this.
const ticksBetweenSaves = TicksPerSecond * 60

// chunkXzToChunkIndex assumes that locDelta is offset relative to the shard
// origin.
func chunkXzToChunkIndex(locDelta *ChunkXz) int {
	return int(locDelta.X)*ShardSize + int(locDelta.Z)
}

// ChunkShard represents a square shard of chunks that share a master
// goroutine.
type ChunkShard struct {
	shardConnecter   gamerules.IShardConnecter
	chunkStore       chunkstore.IChunkStore
	entityMgr        *entity.EntityManager
	loc              ShardXz
	originChunkLoc   ChunkXz // The lowest X and Z located chunk in the shard.
	chunks           [chunksPerShard]*Chunk
	requests         chan iShardRequest
	ticksSinceUpdate Ticks
	ticksSinceSave   Ticks
	saveChunks       bool

	newActiveBlocks []BlockXyz
	newActiveShards map[uint64]*destActiveShard

	shardClients map[uint64]gamerules.IShardShardClient
	selfClient   shardSelfClient
}

func NewChunkShard(shardConnecter gamerules.IShardConnecter, chunkStore chunkstore.IChunkStore, entityMgr *entity.EntityManager, loc ShardXz) (shard *ChunkShard) {
	shard = &ChunkShard{
		shardConnecter:   shardConnecter,
		chunkStore:       chunkStore,
		entityMgr:        entityMgr,
		loc:              loc,
		originChunkLoc:   loc.ToChunkXz(),
		requests:         make(chan iShardRequest, 256),
		ticksSinceUpdate: 0,
		saveChunks:       chunkStore.SupportsWrite(),

		// Offset shard saves.
		ticksSinceSave: (31 * Ticks(loc.Key())) % ticksBetweenSaves,

		newActiveShards: make(map[uint64]*destActiveShard),

		shardClients: make(map[uint64]gamerules.IShardShardClient),
	}

	shard.selfClient.shard = shard

	return
}

// serve services shard requests in the foreground.
func (shard *ChunkShard) serve() {
	ticker := time.NewTicker(NanosecondsInSecond / TicksPerSecond)

	for {
		select {
		case <-ticker.C:
			shard.tick()

		case request := <-shard.requests:
			request.perform(shard)
		}
	}
}

// tick runs the shard for a single tick.
func (shard *ChunkShard) tick() {
	shard.ticksSinceUpdate++

	for _, chunk := range shard.chunks {
		if chunk != nil {
			chunk.tick()
		}
	}

	if shard.ticksSinceUpdate >= TicksPerSecond {
		for _, chunk := range shard.chunks {
			if chunk != nil {
				chunk.sendUpdate()
			}
		}
		shard.ticksSinceUpdate = 0
	}

	if shard.saveChunks && shard.chunkStore.SupportsWrite() {
		shard.ticksSinceSave++
		if shard.ticksSinceSave > ticksBetweenSaves {
			log.Printf("%s: Writing chunks.", shard)
			// TODO Stagger the per-chunk saves over multiple ticks.
			for _, chunk := range shard.chunks {
				if chunk != nil {
					chunk.save(shard.chunkStore)
				}
			}
			shard.ticksSinceSave = 0
		}
	}

	shard.transferActiveBlocks()
}

// clientForShard is used to get a IShardShardClient for a given shard, reusing
// IShardShardClient connections for use within the shard. Returns nil if the
// shard does not exist.
func (shard *ChunkShard) clientForShard(shardLoc ShardXz) (client gamerules.IShardShardClient) {
	var ok bool

	if shard.loc.Equals(&shardLoc) {
		return &shard.selfClient
	}

	shardKey := shardLoc.Key()

	if client, ok = shard.shardClients[shardKey]; !ok {
		client = shard.shardConnecter.ShardShardConnect(shardLoc)
		if client != nil {
			shard.shardClients[shardKey] = client
		}
	}

	return
}

// blockQuery performs a relatively fast query of the BlockId at the given
// location. known=true if the returned blockTypeId is valid.
func (shard *ChunkShard) blockQuery(chunkLoc ChunkXz, subLoc *SubChunkXyz) (blockTypeId BlockId, known bool) {

	chunkIndex, _, _, ok := shard.chunkIndexAndRelLoc(chunkLoc)

	if !ok {
		// blockLoc is in another shard.
		// TODO Have a good fast case to deal with this.
		return
	}

	chunk := shard.chunks[chunkIndex]

	if chunk == nil {
		// Chunk not loaded. Don't bother to load it just for a block query.
		return
	}

	blockIndex, _ := subLoc.BlockIndex()
	blockTypeId = chunk.blockId(blockIndex)
	known = true

	return
}

// transferActiveBlocks takes blocks marked as newly active by addActiveBlock,
// and informs the chunk in the destination shards.
func (shard *ChunkShard) transferActiveBlocks() {
	if len(shard.newActiveBlocks) == 0 {
		return
	}

	thisShardKey := shard.loc.Key()
	for shardKey, activeShard := range shard.newActiveShards {
		if shardKey == thisShardKey {
			shard.reqSetBlocksActive(activeShard.blocks)
		} else {
			if client := shard.clientForShard(activeShard.loc); client != nil {
				client.ReqSetActiveBlocks(activeShard.blocks)
			}
		}
	}
}

// reqSetBlocksActive sets each block in the given slice to be active within
// the chunk. Note: if a block is within a different shard, it is discarded.
func (shard *ChunkShard) reqSetBlocksActive(blocks []BlockXyz) {
	for _, block := range blocks {
		chunkXz := block.ToChunkXz()
		chunkIndex, _, _, isThisShard := shard.chunkIndexAndRelLoc(*chunkXz)
		if isThisShard {
			chunk := shard.chunks[chunkIndex]
			if chunk == nil {
				continue
			}
			chunk.AddActiveBlock(&block)
		}
	}
}

// addActiveBlock sets the given block to be active on the next tick. This
// works even if the block is not within the shard - it will be made active
// provided that the chunk that the block is within is loaded.
func (shard *ChunkShard) addActiveBlock(block *BlockXyz) {
	chunkXz := block.ToChunkXz()
	shardXz := chunkXz.ToShardXz()
	shardKey := shardXz.Key()
	activeShard, ok := shard.newActiveShards[shardKey]
	if ok {
		activeShard = &destActiveShard{
			loc:    shardXz,
			blocks: []BlockXyz{*block},
		}
		shard.newActiveShards[shardKey] = activeShard
	} else {
		activeShard.blocks = append(activeShard.blocks, *block)
	}
}

func (shard *ChunkShard) String() string {
	return fmt.Sprintf("ChunkShard[%#v/%#v]", shard.loc, shard.originChunkLoc)
}

func (shard *ChunkShard) chunkIndexAndRelLoc(loc ChunkXz) (index int, x, z ChunkCoord, ok bool) {
	x = loc.X - shard.originChunkLoc.X
	z = loc.Z - shard.originChunkLoc.Z

	if x < 0 || z < 0 || x >= ShardSize || z >= ShardSize {
		return 0, 0, 0, false
	}

	ok = true

	index = int(x*ShardSize + z)

	return
}

// Get returns the Chunk at at given coordinates, loading it if it is not
// already loaded.
func (shard *ChunkShard) chunkAt(loc ChunkXz) *Chunk {
	chunkIndex, dx, dz, ok := shard.chunkIndexAndRelLoc(loc)
	if !ok {
		log.Printf("%v.Get(%#v): ChunkXz outside of shard", shard, loc)
		return nil
	}

	chunk := shard.chunks[chunkIndex]

	// Chunk already loaded.
	if chunk != nil {
		return chunk
	}

	chunk = shard.loadChunk(loc, ChunkXz{dx, dz})

	if chunk == nil {
		// No chunk available at that location. (Return nil explicitly - interfaces
		// containing nil pointer don't equal nil).
		return nil
	}

	shard.chunks[chunkIndex] = chunk

	return chunk
}

// loadChunk loads the specified chunk from store, and returns it.
// loc - The absolute world position of the chunk.
// locDelta - The relative position of the chunk within the shard.
func (shard *ChunkShard) loadChunk(loc ChunkXz, locDelta ChunkXz) *Chunk {
	chunkResult := <-shard.chunkStore.ReadChunk(loc)
	chunkReader, err := chunkResult.Reader, chunkResult.Err
	if err != nil {
		if _, ok := err.(chunkstore.NoSuchChunkError); !ok {
			log.Printf("%v.load(%#v): chunk loading error: %v", shard, loc, err)
			return nil
		} else {
			// Chunk doesn't exist in store.
			return nil
		}
	}

	chunk := newChunkFromReader(chunkReader, shard)

	return chunk
}

// enqueueAllChunks runs a given function on all loaded chunks in the shard.
func (shard *ChunkShard) enqueueAllChunks(fn func(chunk *Chunk)) {
	shard.requests <- &runOnAllChunks{fn}
}

// enqueueOnChunk runs a function on the chunk at the given location. If the
// chunk does not exist, it does nothing.
func (shard *ChunkShard) enqueueOnChunk(loc ChunkXz, fn func(chunk *Chunk)) {
	shard.requests <- &runOnChunk{loc, fn}
}

func (shard *ChunkShard) enqueue(fn func()) {
	shard.requests <- &runGeneric{fn}
}

func (shard *ChunkShard) enqueueRequest(req iShardRequest) {
	shard.requests <- req
}

type destActiveShard struct {
	loc    ShardXz
	blocks []BlockXyz
}

// shardSelfClient implements IShardShardClient for a shard to efficiently talk
// to itself.
type shardSelfClient struct {
	shard *ChunkShard
}

func (client *shardSelfClient) Disconnect() {
}

func (client *shardSelfClient) ReqSetActiveBlocks(blocks []BlockXyz) {
	client.shard.reqSetBlocksActive(blocks)
}

func (client *shardSelfClient) ReqTransferEntity(loc ChunkXz, entity gamerules.INonPlayerEntity) {
	chunk := client.shard.chunkAt(loc)
	if chunk != nil {
		chunk.transferEntity(entity)
	}
}
