package shardserver

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"rand"
	"time"

	"chunkymonkey/block"
	"chunkymonkey/chunkstore"
	"chunkymonkey/item"
	"chunkymonkey/itemtype"
	"chunkymonkey/mob"
	"chunkymonkey/object"
	"chunkymonkey/proto"
	"chunkymonkey/recipe"
	"chunkymonkey/slot"
	"chunkymonkey/stub"
	. "chunkymonkey/types"
)

var enableMobs = flag.Bool(
	"enableMobs", false, "EXPERIMENTAL: spawn mobs.")

// A chunk is slice of the world map.
type Chunk struct {
	mgr        *LocalShardManager
	shard      *ChunkShard
	loc        ChunkXz
	blocks     []byte
	blockData  []byte
	blockLight []byte
	skyLight   []byte
	heightMap  []byte
	// spawn are typically mobs or items.
	// TODO: (discuss) Maybe split this back into mobs and items?
	// There are many more users of "spawn" than of only mobs or items. So
	// I'm inclined to leave it as is.
	spawn        map[EntityId]object.INonPlayerSpawn
	blockExtra   map[BlockIndex]interface{} // Used by IBlockAspect to store private specific data.
	rand         *rand.Rand
	neighbours   neighboursCache
	cachedPacket []byte                               // Cached packet data for this chunk.
	subscribers  map[EntityId]stub.IShardPlayerClient // Players getting updates from the chunk.
	playersData  map[EntityId]*playerData             // Some player data for player(s) in the chunk.
	onUnsub      map[EntityId][]block.IUnsubscribed   // Functions to be called when unsubscribed.

	activeBlocks    map[BlockIndex]bool // Blocks that need to "tick".
	newActiveBlocks map[BlockIndex]bool // Blocks added as active for next "tick".
}

func newChunkFromReader(reader chunkstore.IChunkReader, mgr *LocalShardManager, shard *ChunkShard) (chunk *Chunk) {
	chunk = &Chunk{
		mgr:         mgr,
		shard:       shard,
		loc:         *reader.ChunkLoc(),
		blocks:      reader.Blocks(),
		blockData:   reader.BlockData(),
		skyLight:    reader.SkyLight(),
		blockLight:  reader.BlockLight(),
		heightMap:   reader.HeightMap(),
		spawn:       make(map[EntityId]object.INonPlayerSpawn),
		blockExtra:  make(map[BlockIndex]interface{}),
		rand:        rand.New(rand.NewSource(time.UTC().Seconds())),
		subscribers: make(map[EntityId]stub.IShardPlayerClient),
		playersData: make(map[EntityId]*playerData),
		onUnsub:     make(map[EntityId][]block.IUnsubscribed),

		activeBlocks:    make(map[BlockIndex]bool),
		newActiveBlocks: make(map[BlockIndex]bool),
	}
	chunk.neighbours.init()
	return
}

func (chunk *Chunk) String() string {
	return fmt.Sprintf("Chunk[%d,%d]", chunk.loc.X, chunk.loc.Z)
}

// Sets a block and its data. Returns true if the block was not changed.
func (chunk *Chunk) setBlock(blockLoc *BlockXyz, subLoc *SubChunkXyz, index BlockIndex, blockType BlockId, blockData byte) {

	// Invalidate cached packet.
	chunk.cachedPacket = nil

	index.SetBlockId(chunk.blocks, blockType)
	index.SetBlockData(chunk.blockData, blockData)

	chunk.blockExtra[index] = nil, false

	// Tell players that the block changed.
	packet := new(bytes.Buffer)
	proto.WriteBlockChange(packet, blockLoc, blockType, blockData)
	chunk.reqMulticastPlayers(-1, packet.Bytes())

	// Update neighbour caches of this change.
	chunk.neighbours.setBlock(subLoc, blockType)

	return
}

func (chunk *Chunk) SetBlockByIndex(blockIndex BlockIndex, blockId BlockId, blockData byte) {
	subLoc := blockIndex.ToSubChunkXyz()
	blockLoc := chunk.loc.ToBlockXyz(&subLoc)

	chunk.setBlock(
		blockLoc,
		&subLoc,
		blockIndex,
		blockId,
		blockData)
}

func (chunk *Chunk) Rand() *rand.Rand {
	return chunk.rand
}

func (chunk *Chunk) ItemType(itemTypeId ItemTypeId) (itemType *itemtype.ItemType, ok bool) {
	itemType, ok = chunk.mgr.gameRules.ItemTypes[itemTypeId]
	return
}

// Tells the chunk to take posession of the item/mob from another chunk.
func (chunk *Chunk) transferSpawn(s object.INonPlayerSpawn) {
	chunk.spawn[s.GetEntityId()] = s
}

// AddSpawn creates a mob or item in this chunk and notifies the new spawn to
// all chunk subscribers.
func (chunk *Chunk) AddSpawn(s object.INonPlayerSpawn) {
	newEntityId := chunk.mgr.entityMgr.NewEntity()
	s.SetEntityId(newEntityId)
	chunk.spawn[newEntityId] = s

	// Spawn new item/mob for players.
	buf := &bytes.Buffer{}
	s.SendSpawn(buf)
	chunk.reqMulticastPlayers(-1, buf.Bytes())
}

func (chunk *Chunk) removeSpawn(s object.INonPlayerSpawn) {
	e := s.GetEntityId()
	chunk.mgr.entityMgr.RemoveEntityById(e)
	chunk.spawn[e] = nil, false
	// Tell all subscribers that the spawn's entity is destroyed.
	buf := new(bytes.Buffer)
	proto.WriteEntityDestroy(buf, e)
	chunk.reqMulticastPlayers(-1, buf.Bytes())
}

func (chunk *Chunk) BlockExtra(index BlockIndex) interface{} {
	if extra, ok := chunk.blockExtra[index]; ok {
		return extra
	}
	return nil
}

func (chunk *Chunk) SetBlockExtra(index BlockIndex, extra interface{}) {
	chunk.blockExtra[index] = extra, extra != nil
}

func (chunk *Chunk) getBlockIndexByBlockXyz(blockLoc *BlockXyz) (index BlockIndex, subLoc *SubChunkXyz, ok bool) {
	chunkLoc, subLoc := blockLoc.ToChunkLocal()

	if chunkLoc.X != chunk.loc.X || chunkLoc.Z != chunk.loc.Z {
		log.Printf(
			"%v.getBlockIndexByBlockXyz: position (%T%#v) is not within chunk",
			chunk, blockLoc, blockLoc)
		return 0, nil, false
	}

	index, ok = subLoc.BlockIndex()
	if !ok {
		log.Printf(
			"%v.getBlockIndexByBlockXyz: invalid position (%T%#v) within chunk",
			chunk, blockLoc, blockLoc)
	}

	return
}

func (chunk *Chunk) blockTypeAndData(index BlockIndex) (blockType *block.BlockType, blockData byte, ok bool) {
	blockTypeId := index.GetBlockId(chunk.blocks)

	blockType, ok = chunk.mgr.gameRules.BlockTypes.Get(blockTypeId)
	if !ok {
		log.Printf(
			"%v.blockTypeAndData: unknown block type %d at index %d",
			chunk, blockTypeId, index,
		)
		return nil, 0, false
	}

	blockData = index.GetBlockData(chunk.blockData)
	return
}

func (chunk *Chunk) blockInstanceAndType(blockLoc *BlockXyz) (blockInstance *block.BlockInstance, blockType *block.BlockType, ok bool) {
	index, subLoc, ok := chunk.getBlockIndexByBlockXyz(blockLoc)
	if !ok {
		return
	}

	blockType, blockData, ok := chunk.blockTypeAndData(index)
	if !ok {
		return
	}

	blockInstance = &block.BlockInstance{
		Chunk:     chunk,
		BlockLoc:  *blockLoc,
		SubLoc:    *subLoc,
		Index:     index,
		BlockType: blockType,
		Data:      blockData,
	}

	return
}

func (chunk *Chunk) RecipeSet() *recipe.RecipeSet {
	return chunk.mgr.gameRules.Recipes
}

func (chunk *Chunk) FurnaceData() *recipe.FurnaceData {
	return &chunk.mgr.gameRules.FurnaceData
}

func (chunk *Chunk) ItemTypes() itemtype.ItemTypeMap {
	return chunk.mgr.gameRules.ItemTypes
}

func (chunk *Chunk) reqHitBlock(player stub.IShardPlayerClient, held slot.Slot, digStatus DigStatus, target *BlockXyz, face Face) {

	blockInstance, blockType, ok := chunk.blockInstanceAndType(target)
	if !ok {
		return
	}

	if blockType.Destructable && blockType.Aspect.Hit(blockInstance, player, digStatus) {
		blockType.Aspect.Destroy(blockInstance)
		chunk.setBlock(target, &blockInstance.SubLoc, blockInstance.Index, BlockIdAir, 0)
	}

	return
}

func (chunk *Chunk) reqInteractBlock(player stub.IShardPlayerClient, held slot.Slot, target *BlockXyz, againstFace Face) {
	// TODO use held item to better check of if the player is trying to place a
	// block vs. perform some other interaction (e.g hoeing dirt). This is
	// perhaps best solved by sending held item type and the face to
	// blockType.Aspect.Interact()

	blockInstance, blockType, ok := chunk.blockInstanceAndType(target)
	if !ok {
		return
	}

	if _, isBlockHeld := held.GetItemTypeId().ToBlockId(); isBlockHeld && blockType.Attachable {
		// The player is interacting with a block that can be attached to.

		// Work out the position to put the block at.
		dx, dy, dz := againstFace.GetDxyz()
		destLoc := target.AddXyz(dx, dy, dz)
		if destLoc == nil {
			// there is overflow with the translation, so do nothing
			return
		}

		player.ReqPlaceHeldItem(*destLoc, held)
	} else {
		// Player is otherwise interacting with the block.
		blockType.Aspect.Interact(blockInstance, player)
	}

	return
}

// placeBlock attempts to place a block. This is called by PlayerBlockInteract
// in the situation where the player interacts with an attachable block
// (potentially in a different chunk to the one where the block gets placed).
func (chunk *Chunk) reqPlaceItem(player stub.IShardPlayerClient, target *BlockXyz, slot *slot.Slot) {
	// TODO defer a check for remaining items in slot, and do something with them
	// (send to player or drop on the ground).

	// TODO more flexible item checking for block placement (e.g placing seed
	// items on farmland doesn't fit this current simplistic model). The block
	// type for the block being placed against should probably contain this logic
	// (i.e farmland block should know about the seed item).
	heldBlockType, ok := slot.GetItemTypeId().ToBlockId()
	if !ok || slot.Count < 1 {
		// Not a placeable item.
		return
	}

	index, subLoc, ok := chunk.getBlockIndexByBlockXyz(target)
	if !ok {
		return
	}

	// Blocks can only replace certain blocks.
	blockTypeId := index.GetBlockId(chunk.blocks)
	blockType, ok := chunk.mgr.gameRules.BlockTypes.Get(blockTypeId)
	if !ok || !blockType.Replaceable {
		return
	}

	// Safe to replace block.
	chunk.setBlock(target, subLoc, index, heldBlockType, byte(slot.Data))

	slot.Decrement()
}

func (chunk *Chunk) reqTakeItem(player stub.IShardPlayerClient, entityId EntityId) {
	if spawn, ok := chunk.spawn[entityId]; ok {
		if item, ok := spawn.(*item.Item); ok {
			player.ReqGiveItem(*item.Position(), *item.GetSlot())

			// Tell all subscribers to animate the item flying at the
			// player.
			buf := new(bytes.Buffer)
			proto.WriteItemCollect(buf, entityId, player.GetEntityId())
			chunk.reqMulticastPlayers(-1, buf.Bytes())
			chunk.removeSpawn(item)
		}
	}
}

func (chunk *Chunk) reqDropItem(player stub.IShardPlayerClient, content *slot.Slot, position *AbsXyz, velocity *AbsVelocity) {
	spawnedItem := item.NewItem(
		content.ItemType,
		content.Count,
		content.Data,
		position,
		velocity,
	)

	chunk.AddSpawn(spawnedItem)
}

func (chunk *Chunk) reqInventoryClick(player stub.IShardPlayerClient, blockLoc *BlockXyz, slotId SlotId, cursor *slot.Slot, rightClick bool, shiftClick bool, txId TxId, expectedSlot *slot.Slot) {
	blockInstance, blockType, ok := chunk.blockInstanceAndType(blockLoc)
	if !ok {
		return
	}

	blockType.Aspect.InventoryClick(
		blockInstance, player, slotId, cursor,
		rightClick, shiftClick,
		txId, expectedSlot)
}

func (chunk *Chunk) reqInventoryUnsubscribed(player stub.IShardPlayerClient, blockLoc *BlockXyz) {
	blockInstance, blockType, ok := chunk.blockInstanceAndType(blockLoc)
	if !ok {
		return
	}

	blockType.Aspect.InventoryUnsubscribed(blockInstance, player)
}

// Used to read the BlockId of a block that's either in the chunk, or
// immediately adjoining it in a neighbouring chunk via the side caches.
func (chunk *Chunk) blockQuery(blockLoc *BlockXyz) (blockType *block.BlockType, isWithinChunk bool, blockUnknownId bool) {
	chunkLoc, subLoc := blockLoc.ToChunkLocal()

	var blockTypeId BlockId
	var ok bool

	if chunkLoc.X == chunk.loc.X && chunkLoc.Z == chunk.loc.Z {
		// The item is asking about this chunk.
		index, ok := subLoc.BlockIndex()
		if !ok {
			return
		}

		blockTypeId = index.GetBlockId(chunk.blocks)
		isWithinChunk = true
	} else {
		// The item is asking about a separate chunk.
		isWithinChunk = false

		ok, blockTypeId = chunk.neighbours.CachedBlock(
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
			"%v.blockQuery found unknown block type Id %d at %+v",
			chunk, blockTypeId, blockLoc)
		blockUnknownId = true
	}

	return
}

func (chunk *Chunk) tick() {
	// Update neighbouring chunks of block changes in this chunk.
	chunk.neighbours.flush()

	chunk.spawnTick()

	chunk.blockTick()
}

// spawnTick runs all spawns for a tick.
func (chunk *Chunk) spawnTick() {
	if len(chunk.spawn) == 0 {
		// Nothing to do, bail out early.
		return
	}

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
	outgoingSpawns := []object.INonPlayerSpawn{}

	for _, e := range chunk.spawn {
		if e.Tick(blockQuery) {
			if e.Position().Y <= 0 {
				// Item or mob fell out of the world.
				chunk.removeSpawn(e)
			} else {
				outgoingSpawns = append(outgoingSpawns, e)
			}
		}
	}

	if len(outgoingSpawns) > 0 {
		// Transfer spawns to new chunk.
		for _, e := range outgoingSpawns {
			// Remove mob/items from this chunk.
			chunk.spawn[e.GetEntityId()] = nil, false

			// Transfer to other chunk.
			chunkLoc := e.Position().ToChunkXz()

			// TODO Batch spawns up into a request per shard if there are efficiency
			// concerns in sending them individually.
			chunk.mgr.EnqueueOnChunk(chunkLoc, func(blockChunk *Chunk) {
				blockChunk.transferSpawn(e)
			})
		}
	}

	// XXX: Testing hack. If player is in a chunk with no mobs, spawn a pig.
	if *enableMobs {
		for _, playerData := range chunk.playersData {
			loc := playerData.position.ToChunkXz()
			if chunk.isSameChunk(&loc) {
				ms := chunk.mobs()
				if len(ms) == 0 {
					log.Printf("%v.Tick: spawning a mob at %v", chunk, playerData.position)
					m := mob.NewPig(&playerData.position, &AbsVelocity{5, 5, 5})
					chunk.AddSpawn(&m.Mob)
				}
				break
			}
		}
	}
}

// blockTick runs any blocks that need to do something each tick.
func (chunk *Chunk) blockTick() {
	if len(chunk.activeBlocks) == 0 && len(chunk.newActiveBlocks) == 0 {
		return
	}

	for blockIndex := range chunk.newActiveBlocks {
		chunk.activeBlocks[blockIndex] = true
		chunk.newActiveBlocks[blockIndex] = false, false
	}

	var ok bool
	var blockInstance block.BlockInstance
	blockInstance.Chunk = chunk

	for blockIndex := range chunk.activeBlocks {
		blockInstance.BlockType, blockInstance.Data, ok = chunk.blockTypeAndData(blockIndex)
		if !ok {
			// Invalid block.
			chunk.activeBlocks[blockIndex] = false, false
		}

		blockInstance.SubLoc = blockIndex.ToSubChunkXyz()
		blockInstance.Index = blockIndex
		blockInstance.BlockLoc = *chunk.loc.ToBlockXyz(&blockInstance.SubLoc)

		if !blockInstance.BlockType.Aspect.Tick(&blockInstance) {
			// Block now inactive. Remove this block from the active list.
			chunk.activeBlocks[blockIndex] = false, false
		}
	}
}

func (chunk *Chunk) AddActiveBlock(blockXyz *BlockXyz) {
	chunkXz, subLoc := blockXyz.ToChunkLocal()
	if chunk.isSameChunk(chunkXz) {
		if index, ok := subLoc.BlockIndex(); ok {
			chunk.newActiveBlocks[index] = true
		}
	}
}

func (chunk *Chunk) AddActiveBlockIndex(blockIndex BlockIndex) {
	chunk.newActiveBlocks[blockIndex] = true
}

func (chunk *Chunk) mobs() (s []*mob.Mob) {
	s = make([]*mob.Mob, 0, 3)
	for _, e := range chunk.spawn {
		switch e.(type) {
		case *mob.Mob:
			s = append(s, e.(*mob.Mob))
		}
	}
	return
}

func (chunk *Chunk) items() (s []*item.Item) {
	s = make([]*item.Item, 0, 10)
	for _, e := range chunk.spawn {
		switch e.(type) {
		case *item.Item:
			s = append(s, e.(*item.Item))
		}
	}
	return
}

func (chunk *Chunk) reqSubscribeChunk(entityId EntityId, player stub.IShardPlayerClient) {
	if _, ok := chunk.subscribers[entityId]; ok {
		// Already subscribed.
		return
	}

	chunk.subscribers[entityId] = player

	buf := new(bytes.Buffer)
	proto.WritePreChunk(buf, &chunk.loc, ChunkInit)
	player.TransmitPacket(buf.Bytes())

	player.TransmitPacket(chunk.chunkPacket())

	// Send spawns of all mobs/items in the chunk.
	if len(chunk.spawn) > 0 {
		buf := new(bytes.Buffer)
		for _, e := range chunk.spawn {
			e.SendSpawn(buf)
		}
		player.TransmitPacket(buf.Bytes())
	}

	// Spawn existing players for new player.
	if len(chunk.playersData) > 0 {
		playersPacket := new(bytes.Buffer)
		for _, existing := range chunk.playersData {
			if existing.entityId != entityId {
				existing.sendSpawn(playersPacket)
			}
		}
		player.TransmitPacket(playersPacket.Bytes())
	}
}

func (chunk *Chunk) reqUnsubscribeChunk(entityId EntityId, sendPacket bool) {
	if player, ok := chunk.subscribers[entityId]; ok {
		chunk.subscribers[entityId] = nil, false

		// Call any observers registered with AddOnUnsubscribe.
		if observers, ok := chunk.onUnsub[entityId]; ok {
			chunk.onUnsub[entityId] = nil, false
			for _, observer := range observers {
				observer.Unsubscribed(entityId)
			}
		}

		if sendPacket {
			buf := new(bytes.Buffer)
			proto.WritePreChunk(buf, &chunk.loc, ChunkUnload)
			// TODO send PacketEntityDestroy packets for spawns in this chunk.
			player.TransmitPacket(buf.Bytes())
		}
	}
}

// AddOnUnsubscribe registers a function to be called when the given subscriber
// unsubscribes.
func (chunk *Chunk) AddOnUnsubscribe(entityId EntityId, observer block.IUnsubscribed) {
	observers := chunk.onUnsub[entityId]
	observers = append(observers, observer)
	chunk.onUnsub[entityId] = observers
}

// RemoveOnUnsubscribe removes a function previously registered
func (chunk *Chunk) RemoveOnUnsubscribe(entityId EntityId, observer block.IUnsubscribed) {
	observers, ok := chunk.onUnsub[entityId]
	if !ok {
		return
	}

	for i := range observers {
		if observers[i] == observer {
			// Remove!
			if i < len(observers)-1 {
				observers[i] = observers[len(observers)-1]
			}
			observers = observers[:len(observers)-1]

			// Replace slice in map, or remove if empty.
			chunk.onUnsub[entityId] = observers, (len(observers) > 0)

			return
		}
	}
}

func (chunk *Chunk) reqMulticastPlayers(exclude EntityId, packet []byte) {
	for entityId, player := range chunk.subscribers {
		if entityId != exclude {
			player.TransmitPacket(packet)
		}
	}
}

func (chunk *Chunk) reqAddPlayerData(entityId EntityId, name string, pos AbsXyz, look LookBytes, held ItemTypeId) {
	// TODO add other initial data in here.
	newPlayerData := &playerData{
		entityId:   entityId,
		name:       name,
		position:   pos,
		look:       look,
		heldItemId: held,
	}
	chunk.playersData[entityId] = newPlayerData

	// Spawn new player for existing players.
	newPlayerPacket := new(bytes.Buffer)
	newPlayerData.sendSpawn(newPlayerPacket)
	chunk.reqMulticastPlayers(entityId, newPlayerPacket.Bytes())
}

func (chunk *Chunk) reqRemovePlayerData(entityId EntityId, isDisconnect bool) {
	chunk.playersData[entityId] = nil, false

	if isDisconnect {
		buf := new(bytes.Buffer)
		proto.WriteEntityDestroy(buf, entityId)
		chunk.reqMulticastPlayers(entityId, buf.Bytes())
	}
}

func (chunk *Chunk) reqSetPlayerPositionLook(entityId EntityId, pos AbsXyz, look LookBytes, moved bool) {
	data, ok := chunk.playersData[entityId]

	if !ok {
		log.Printf(
			"%v.setPlayerPosition: called for EntityId (%d) not present as playerData.",
			chunk, entityId,
		)
		return
	}

	data.position = pos
	data.look = look

	// Update subscribers.
	buf := new(bytes.Buffer)
	data.sendPositionLook(buf)
	chunk.reqMulticastPlayers(entityId, buf.Bytes())

	if moved {
		player, ok := chunk.subscribers[entityId]

		if ok {
			// Does the player overlap with any items?
			for _, item := range chunk.items() {
				// TODO This check should be performed when items move as well.
				if data.OverlapsItem(item) {
					slot := item.GetSlot()
					player.ReqOfferItem(chunk.loc, item.EntityId, *slot)
				}
			}
		}
	}
}

func (chunk *Chunk) chunkPacket() []byte {
	if chunk.cachedPacket == nil {
		buf := new(bytes.Buffer)
		proto.WriteMapChunk(buf, &chunk.loc, chunk.blocks, chunk.blockData, chunk.blockLight, chunk.skyLight)
		chunk.cachedPacket = buf.Bytes()
	}

	return chunk.cachedPacket
}

func (chunk *Chunk) sendUpdate() {
	buf := &bytes.Buffer{}
	for _, e := range chunk.spawn {
		e.SendUpdate(buf)
	}
	chunk.reqMulticastPlayers(-1, buf.Bytes())
}

func (chunk *Chunk) sideCacheSetNeighbour(side ChunkSideDir, neighbour *Chunk) {
	chunk.neighbours.sideCacheSetNeighbour(side, neighbour, chunk.blocks)
}

func (chunk *Chunk) isSameChunk(otherChunkLoc *ChunkXz) bool {
	return otherChunkLoc.X == chunk.loc.X && otherChunkLoc.Z == chunk.loc.Z
}

func (chunk *Chunk) EnqueueGeneric(fn func()) {
	chunk.shard.enqueueRequest(&runGeneric{fn})
}
