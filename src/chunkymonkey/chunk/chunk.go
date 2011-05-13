// Map chunks

package chunk

import (
	"bytes"
	"flag"
	"log"
	"rand"
	"sync"
	"time"

	"chunkymonkey/block"
	"chunkymonkey/chunkstore"
	"chunkymonkey/entity"
	. "chunkymonkey/interfaces"
	"chunkymonkey/item"
	"chunkymonkey/itemtype"
	"chunkymonkey/mob"
	"chunkymonkey/proto"
	"chunkymonkey/recipe"
	"chunkymonkey/slot"
	. "chunkymonkey/types"
)

const (
	// Assumed values for size of player axis-aligned bounding box (AAB).
	playerAabH = AbsCoord(0.75) // Each side of player.
	playerAabY = AbsCoord(2.00) // From player's feet position upwards.
)

var enableMobs = flag.Bool(
	"enableMobs", false, "EXPERIMENTAL: spawn mobs.")


// A chunk is slice of the world map.
type Chunk struct {
	mainQueue  chan func(IChunk)
	mgr        *ChunkManager
	loc        ChunkXz
	blocks     []byte
	blockData  []byte
	blockLight []byte
	skyLight   []byte
	heightMap  []byte
	// spawners are typically mobs or items.
	// TODO: (discuss) Maybe split this back into mobs and items?
	// There are many more users of "spawners" than of only mobs or items. So
	// I'm inclined to leave it as is.
	spawners      map[EntityId]entity.ISpawn
	blockExtra    map[BlockIndex]interface{} // Used by IBlockAspect to store private specific data.
	rand          *rand.Rand
	neighbours    neighboursCache
	cachedPacket  []byte                       // Cached packet data for this block.
	subscribers   map[IChunkSubscriber]bool    // Subscribers getting updates from the chunk.
	subscriberPos map[IChunkSubscriber]*AbsXyz // Player positions that are near or in the chunk.
}

func newChunkFromReader(reader chunkstore.ChunkReader, mgr *ChunkManager) (chunk *Chunk) {
	chunk = &Chunk{
		mainQueue:     make(chan func(IChunk), 256),
		mgr:           mgr,
		loc:           *reader.ChunkLoc(),
		blocks:        reader.Blocks(),
		blockData:     reader.BlockData(),
		skyLight:      reader.SkyLight(),
		blockLight:    reader.BlockLight(),
		heightMap:     reader.HeightMap(),
		spawners:      make(map[EntityId]entity.ISpawn),
		blockExtra:    make(map[BlockIndex]interface{}),
		rand:          rand.New(rand.NewSource(time.UTC().Seconds())),
		subscribers:   make(map[IChunkSubscriber]bool),
		subscriberPos: make(map[IChunkSubscriber]*AbsXyz),
	}
	chunk.neighbours.init()
	go chunk.mainLoop()
	return
}

// Sets a block and its data. Returns true if the block was not changed.
func (chunk *Chunk) setBlock(blockLoc *BlockXyz, subLoc *SubChunkXyz, index BlockIndex, blockType BlockId, blockMetadata byte) {

	// Invalidate cached packet.
	chunk.cachedPacket = nil

	index.SetBlockId(chunk.blocks, blockType)
	index.SetBlockData(chunk.blockData, blockMetadata)

	chunk.blockExtra[index] = nil, false

	// Tell players that the block changed.
	packet := &bytes.Buffer{}
	proto.WriteBlockChange(packet, blockLoc, blockType, blockMetadata)
	chunk.multicastSubscribers(packet.Bytes())

	// Update neighbour caches of this change.
	chunk.neighbours.setBlock(subLoc, blockType)

	return
}

func (chunk *Chunk) GetLoc() *ChunkXz {
	return &chunk.loc
}

func (chunk *Chunk) Enqueue(f func(IChunk)) {
	chunk.mainQueue <- f
}

func (chunk *Chunk) EnqueueGeneric(f func(interface{})) {
	chunk.mainQueue <- func(chunk IChunk) {
		f(chunk)
	}
}

func (chunk *Chunk) mainLoop() {
	for {
		f := <-chunk.mainQueue
		f(chunk)
	}
}

func (chunk *Chunk) GetRand() *rand.Rand {
	return chunk.rand
}

func (chunk *Chunk) GetItemType(itemTypeId ItemTypeId) (itemType *itemtype.ItemType, ok bool) {
	itemType, ok = chunk.mgr.gameRules.ItemTypes[itemTypeId]
	return
}

func (chunk *Chunk) TransferSpawner(s entity.ISpawn) {
	chunk.spawners[s.GetEntity().EntityId] = s
}

// AddSpawner creates a mob or item in this chunk and notifies the new spawn to
// all chunk subscribers.
func (chunk *Chunk) AddSpawner(s entity.ISpawn) {
	wg := &sync.WaitGroup{}
	wg.Add(1)
	chunk.mgr.game.Enqueue(func(game IGame) {
		e := s.GetEntity()
		game.AddEntity(e)
		chunk.spawners[e.EntityId] = s
		wg.Done()
	})
	wg.Wait()

	// Spawn new item/mob for players.
	buf := &bytes.Buffer{}
	s.SendSpawn(buf)
	chunk.multicastSubscribers(buf.Bytes())
}

func (chunk *Chunk) removeSpawner(s entity.ISpawn) {
	e := s.GetEntity()
	chunk.mgr.game.Enqueue(func(game IGame) {
		game.RemoveEntity(e)
	})
	chunk.spawners[e.EntityId] = nil, false
	// Tell all subscribers that the spawner's entity is
	// destroyed.
	buf := &bytes.Buffer{}
	proto.WriteEntityDestroy(buf, e.EntityId)
	chunk.multicastSubscribers(buf.Bytes())
}

func (chunk *Chunk) GetBlockExtra(subLoc *SubChunkXyz) interface{} {
	if index, ok := subLoc.BlockIndex(); ok {
		if extra, ok := chunk.blockExtra[index]; ok {
			return extra
		}
	}
	return nil
}

func (chunk *Chunk) SetBlockExtra(subLoc *SubChunkXyz, extra interface{}) {
	if index, ok := subLoc.BlockIndex(); ok {
		chunk.blockExtra[index] = extra, extra != nil
	}
}

func (chunk *Chunk) GetBlock(subLoc *SubChunkXyz) (blockType BlockId, ok bool) {
	index, ok := subLoc.BlockIndex()
	if !ok {
		return
	}

	blockType = index.GetBlockId(chunk.blocks)

	return
}

func (chunk *Chunk) GetRecipeSet() *recipe.RecipeSet {
	return chunk.mgr.gameRules.Recipes
}

func (chunk *Chunk) PlayerBlockHit(player IPlayer, subLoc *SubChunkXyz, digStatus DigStatus) (ok bool) {
	index, ok := subLoc.BlockIndex()
	if !ok {
		return
	}

	blockTypeId := index.GetBlockId(chunk.blocks)

	if blockType, ok := chunk.mgr.gameRules.BlockTypes.Get(blockTypeId); ok && blockType.Destructable {
		blockData := index.GetBlockData(chunk.blockData)
		blockLoc := chunk.loc.ToBlockXyz(subLoc)

		blockInstance := &block.BlockInstance{
			Chunk:    chunk,
			BlockLoc: *blockLoc,
			SubLoc:   *subLoc,
			Data:     blockData,
		}
		if blockType.Aspect.Hit(blockInstance, player, digStatus) {
			chunk.setBlock(blockLoc, subLoc, index, BlockIdAir, 0)
		}
	} else {
		log.Printf("Chunk/PlayerBlockHit: Attempted to destroy unknown block Id %d", blockTypeId)
		ok = false
	}

	return
}

func (chunk *Chunk) PlayerBlockInteract(player IPlayer, target *BlockXyz, againstFace Face) {
	// TODO pass in currently held item to allow better checking of if the player
	// is trying to place a block vs. perform some other interaction (e.g hoeing
	// dirt). placeBlock() will need to check again for the purposes of *taking*
	// a block for placement, of course.

	chunkLoc, subLoc := target.ToChunkLocal()
	if chunkLoc.X != chunk.loc.X || chunkLoc.Z != chunk.loc.Z {
		log.Printf(
			"Chunk/PlayerBlockInteract: target position (%#v) is not within chunk (%#v)",
			target, chunk.loc)
		return
	}
	index, ok := subLoc.BlockIndex()
	if !ok {
		log.Printf(
			"Chunk/PlayerBlockInteract: invalid target position (%#v) within chunk (%#v)",
			target, chunk.loc)
		return
	}

	blockTypeId := BlockId(chunk.blocks[index])
	blockType, ok := chunk.mgr.gameRules.BlockTypes.Get(blockTypeId)
	if !ok {
		log.Printf(
			"Chunk/PlayerBlockInteract: unknown target block type %d at target position (%#v)",
			blockTypeId, target)
		return
	}

	if blockType.Attachable {
		// The player is interacting with a block that can be attached to.

		// Work out the position to put the block at.
		// TODO check for overflow, especially in Y.
		dx, dy, dz := againstFace.GetDxyz()
		destLoc := &BlockXyz{
			target.X + dx,
			target.Y + dy,
			target.Z + dz,
		}
		destChunkLoc, destSubLoc := destLoc.ToChunkLocal()

		// Place the block.
		if chunk.IsSameChunk(destChunkLoc) {
			chunk.placeBlock(player, destLoc, destSubLoc, againstFace)
		} else {
			chunk.mgr.game.Enqueue(func(game IGame) {
				destChunk := chunk.mgr.Get(destChunkLoc).(*Chunk)
				if destChunk != nil {
					destChunk.placeBlock(player, destLoc, destSubLoc, againstFace)
				}
			})
		}

	} else {
		// Player is otherwise interacting with the block.
		blockInstance := &block.BlockInstance{
			Chunk:    chunk,
			BlockLoc: *target,
			SubLoc:   *subLoc,
			Data:     index.GetBlockData(chunk.blockData),
		}
		blockType.Aspect.Interact(blockInstance, player)
	}

	return
}

// placeBlock attempts to place a block. This is called by PlayerBlockInteract
// in the situation where the player interacts with an attachable block
// (potentially in a different chunk to the one where the block gets placed).
func (chunk *Chunk) placeBlock(player IPlayer, destLoc *BlockXyz, destSubLoc *SubChunkXyz, face Face) {
	index, ok := destSubLoc.BlockIndex()
	if !ok {
		return
	}

	// Blocks can only replace certain blocks.
	blockTypeId := index.GetBlockId(chunk.blocks)
	blockType, ok := chunk.mgr.gameRules.BlockTypes.Get(blockTypeId)
	if !ok || !blockType.Replaceable {
		return
	}

	var takenItem slot.Slot
	takenItem.Init()

	// Final check that the player is holding a placeable block item. We use
	// WithLock so that things happen synchronously for a transactional item take
	// and placement.
	player.WithLock(func(player IPlayer) {
		heldItemType := player.GetHeldItemType()
		// TODO more flexible item checking for block placement (e.g placing seed
		// items on farmland doesn't fit this current simplistic model). The block
		// type for the block being placed against should probably contain this
		// logic (i.e farmland block should know about the seed item).
		if heldItemType == nil || heldItemType.Id < BlockIdMin || heldItemType.Id > BlockIdMax {
			// Not a placeable item.
			return
		}

		player.TakeOneHeldItem(&takenItem)
	})

	if takenItem.Count < 1 || takenItem.ItemType == nil {
		return
	}

	// TODO block metadata
	chunk.setBlock(destLoc, destSubLoc, index, BlockId(takenItem.ItemType.Id), 0)
}

// Used to read the BlockId of a block that's either in the chunk, or
// immediately adjoining it in a neighbouring chunk via the side caches.
func (chunk *Chunk) blockQuery(blockLoc *BlockXyz) (blockType *block.BlockType, isWithinChunk bool, blockUnknownId bool) {
	chunkLoc, subLoc := blockLoc.ToChunkLocal()

	var blockTypeId BlockId
	var ok bool

	if chunkLoc.X == chunk.loc.X && chunkLoc.Z == chunk.loc.Z {
		// The item is asking about this chunk.
		blockTypeId, _ = chunk.GetBlock(subLoc)
		isWithinChunk = true
	} else {
		// The item is asking about a separate chunk.
		isWithinChunk = false

		ok, blockTypeId = chunk.neighbours.GetCachedBlock(
			chunk.loc.X-chunkLoc.X,
			chunk.loc.Z-chunkLoc.Z,
			subLoc,
		)

		if !ok {
			// The chunk side isn't cached or isn't a neighbouring block.
			blockUnknownId = true
			return
		}
	}

	blockType, ok = chunk.mgr.gameRules.BlockTypes.Get(blockTypeId)
	if !ok {
		log.Printf(
			"Chunk/blockQuery found unknown block type Id %d at %+v",
			blockTypeId, blockLoc)
		blockUnknownId = true
	}

	return
}

func (chunk *Chunk) Tick() {
	// Update neighbouring chunks of block changes in this chunk
	chunk.neighbours.flush()

	blockQuery := func(blockLoc *BlockXyz) (isSolid bool, isWithinChunk bool) {
		blockType, isWithinChunk, blockUnknownId := chunk.blockQuery(blockLoc)
		if blockUnknownId {
			// If we are in doubt, we assume that the block asked about is
			// solid (this way objects don't fly off the side of the map
			// needlessly).
			isSolid = true
		} else {
			isSolid = blockType.Solid
		}
		return
	}
	outgoingSpawners := []entity.ISpawn{}

	for _, e := range chunk.spawners {
		if e.Tick(blockQuery) {
			if e.GetPosition().Y <= 0 {
				// Item or mob fell out of the world.
				chunk.removeSpawner(e)
			} else {
				outgoingSpawners = append(outgoingSpawners, e)
			}
		}
	}

	if len(outgoingSpawners) > 0 {
		// Remove mob/items from this chunk.
		for _, e := range outgoingSpawners {
			chunk.spawners[e.GetEntity().EntityId] = nil, false
		}

		// Send items to new chunk.
		chunk.mgr.game.Enqueue(func(game IGame) {
			mgr := game.GetChunkManager()
			for _, e := range outgoingSpawners {
				chunkLoc := e.GetPosition().ToChunkXz()
				blockChunk := mgr.Get(chunkLoc)
				blockChunk.Enqueue(func(blockChunk IChunk) {
					blockChunk.AddSpawner(e)
				})
			}
		})
	}
	// XXX: Testing hack. If player is in a chunk with no mobs, spawn a pig.
	if *enableMobs {
		for _, playerPos := range chunk.subscriberPos {
			loc, _ := playerPos.ToBlockXyz().ToChunkLocal()
			if chunk.IsSameChunk(loc) {
				ms := chunk.mobs()
				if len(ms) == 0 {
					log.Println("spawning a mob", chunk.loc, "at", playerPos)
					m := mob.NewPig(playerPos, &AbsVelocity{5, 5, 5})
					chunk.AddSpawner(&m.Mob)
				}
				break
			}
		}
	}
}

func (chunk *Chunk) mobs() (s []*mob.Mob) {
	s = make([]*mob.Mob, 0, 3)
	for _, e := range chunk.spawners {
		switch e.(type) {
		case *mob.Mob:
			s = append(s, e.(*mob.Mob))
		}
	}
	return
}

func (chunk *Chunk) items() (s []*item.Item) {
	s = make([]*item.Item, 0, 10)
	for _, e := range chunk.spawners {
		switch e.(type) {
		case *item.Item:
			s = append(s, e.(*item.Item))
		}
	}
	return
}

func (chunk *Chunk) AddSubscriber(subscriber IChunkSubscriber) {
	chunk.subscribers[subscriber] = true
	subscriber.TransmitPacket(chunk.chunkPacket())

	// Send spawns of all mobs/items in the chunk.
	if len(chunk.spawners) > 0 {
		buf := &bytes.Buffer{}
		for _, e := range chunk.spawners {
			e.SendSpawn(buf)
		}
		subscriber.TransmitPacket(buf.Bytes())
	}
}

func (chunk *Chunk) RemoveSubscriber(subscriber IChunkSubscriber, sendPacket bool) {
	chunk.subscribers[subscriber] = false, false
	if sendPacket {
		buf := &bytes.Buffer{}
		proto.WritePreChunk(buf, &chunk.loc, ChunkUnload)
		subscriber.TransmitPacket(buf.Bytes())
	}
}

func (chunk *Chunk) multicastSubscribers(packet []byte) {
	for subscriber, _ := range chunk.subscribers {
		subscriber.TransmitPacket(packet)
	}
}

func (chunk *Chunk) SetSubscriberPosition(subscriber IChunkSubscriber, pos *AbsXyz) {
	chunk.subscriberPos[subscriber] = pos, pos != nil
	if pos != nil {
		// Does the subscriber overlap with any items?
		minX := pos.X - playerAabH
		maxX := pos.X + playerAabH
		minZ := pos.Z - playerAabH
		maxZ := pos.Z + playerAabH
		minY := pos.Y
		maxY := pos.Y + playerAabY

		for entityId, item := range chunk.items() {
			// TODO This check should be performed when items move as well.
			pos := item.GetPosition()
			if pos.X >= minX && pos.X <= maxX && pos.Y >= minY && pos.Y <= maxY && pos.Z >= minZ && pos.Z <= maxZ {
				slot := item.GetSlot()
				subscriber.OfferItem(slot)
				if slot.Count == 0 {
					// The item has been accepted and completely consumed.

					buf := &bytes.Buffer{}

					// Tell all subscribers to animate the item flying at the
					// subscriber.
					proto.WriteItemCollect(buf, EntityId(entityId), subscriber.GetEntityId())
					chunk.multicastSubscribers(buf.Bytes())
					chunk.removeSpawner(item)
				}

				// TODO Check for how to properly handle partially consumed
				// items? Probably not high priority since all dropped items
				// have a count of 1 at the moment. Might need to respawn the
				// item with a new count. Do the clients even care what the
				// count is or if it changes? Or if an item is "collected" but
				// still exists?
			}
		}
	}
}

func (chunk *Chunk) chunkPacket() []byte {
	if chunk.cachedPacket == nil {
		buf := &bytes.Buffer{}
		proto.WriteMapChunk(buf, &chunk.loc, chunk.blocks, chunk.blockData, chunk.blockLight, chunk.skyLight)
		chunk.cachedPacket = buf.Bytes()
	}

	return chunk.cachedPacket
}

func (chunk *Chunk) SendUpdate() {
	buf := &bytes.Buffer{}
	for _, e := range chunk.spawners {
		e.SendUpdate(buf)
	}
	chunk.multicastSubscribers(buf.Bytes())
}

func (chunk *Chunk) sideCacheSetNeighbour(side ChunkSideDir, neighbour *Chunk) {
	chunk.neighbours.sideCacheSetNeighbour(side, neighbour, chunk.blocks)
}

func (chunk *Chunk) IsSameChunk(otherChunkLoc *ChunkXz) bool {
	return otherChunkLoc.X == chunk.loc.X && otherChunkLoc.Z == chunk.loc.Z
}
