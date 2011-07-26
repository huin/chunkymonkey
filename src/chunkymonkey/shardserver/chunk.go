package shardserver

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"rand"
	"time"

	"chunkymonkey/chunkstore"
	"chunkymonkey/gamerules"
	"chunkymonkey/mob"
	"chunkymonkey/nbtutil"
	"chunkymonkey/object"
	"chunkymonkey/proto"
	. "chunkymonkey/types"
	"nbt"
)

var enableMobs = flag.Bool(
	"enableMobs", false, "EXPERIMENTAL: spawn mobs.")

// A chunk is slice of the world map.
type Chunk struct {
	shard        *ChunkShard
	loc          ChunkXz
	blocks       []byte
	blockData    []byte
	blockLight   []byte
	skyLight     []byte
	heightMap    []byte
	entities     map[EntityId]object.INonPlayerEntity // Entities (mobs, items, etc)
	blockExtra   map[BlockIndex]interface{}           // Used by IBlockAspect to store private specific data.
	rand         *rand.Rand
	cachedPacket []byte                                 // Cached packet data for this chunk.
	subscribers  map[EntityId]gamerules.IPlayerClient   // Players getting updates from the chunk.
	playersData  map[EntityId]*playerData               // Some player data for player(s) in the chunk.
	onUnsub      map[EntityId][]gamerules.IUnsubscribed // Functions to be called when unsubscribed.

	activeBlocks    map[BlockIndex]bool // Blocks that need to "tick".
	newActiveBlocks map[BlockIndex]bool // Blocks added as active for next "tick".
}

func newChunkFromReader(reader chunkstore.IChunkReader, shard *ChunkShard) (chunk *Chunk) {
	chunk = &Chunk{
		shard:       shard,
		loc:         reader.ChunkLoc(),
		blocks:      reader.Blocks(),
		blockData:   reader.BlockData(),
		skyLight:    reader.SkyLight(),
		blockLight:  reader.BlockLight(),
		heightMap:   reader.HeightMap(),
		entities:    make(map[EntityId]object.INonPlayerEntity),
		blockExtra:  make(map[BlockIndex]interface{}),
		rand:        rand.New(rand.NewSource(time.UTC().Seconds())),
		subscribers: make(map[EntityId]gamerules.IPlayerClient),
		playersData: make(map[EntityId]*playerData),
		onUnsub:     make(map[EntityId][]gamerules.IUnsubscribed),

		activeBlocks:    make(map[BlockIndex]bool),
		newActiveBlocks: make(map[BlockIndex]bool),
	}

	chunk.addEntities(reader.Entities())
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

	return
}

func (chunk *Chunk) blockId(index BlockIndex) BlockId {
	return BlockId(index.BlockData(chunk.blocks))
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

func (chunk *Chunk) ItemType(itemTypeId ItemTypeId) (itemType *gamerules.ItemType, ok bool) {
	itemType, ok = gamerules.Items[itemTypeId]
	return
}

// Tells the chunk to take posession of the item/mob from another chunk.
func (chunk *Chunk) transferEntity(s object.INonPlayerEntity) {
	chunk.entities[s.GetEntityId()] = s
}

// AddEntity creates a mob or item in this chunk and notifies all chunk
// subscribers of the new entity
func (chunk *Chunk) AddEntity(s object.INonPlayerEntity) {
	newEntityId := chunk.shard.entityMgr.NewEntity()
	s.SetEntityId(newEntityId)
	chunk.entities[newEntityId] = s

	// Spawn new item/mob for players.
	buf := &bytes.Buffer{}
	s.SendSpawn(buf)
	chunk.reqMulticastPlayers(-1, buf.Bytes())
}

func (chunk *Chunk) removeEntity(s object.INonPlayerEntity) {
	e := s.GetEntityId()
	chunk.shard.entityMgr.RemoveEntityById(e)
	chunk.entities[e] = nil, false
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

func (chunk *Chunk) blockTypeAndData(index BlockIndex) (blockType *gamerules.BlockType, blockData byte, ok bool) {
	blockTypeId := index.BlockId(chunk.blocks)

	blockType, ok = gamerules.Blocks.Get(blockTypeId)
	if !ok {
		log.Printf(
			"%v.blockTypeAndData: unknown block type %d at index %d",
			chunk, blockTypeId, index,
		)
		return nil, 0, false
	}

	blockData = index.BlockData(chunk.blockData)
	return
}

func (chunk *Chunk) blockInstanceAndType(blockLoc *BlockXyz) (blockInstance *gamerules.BlockInstance, blockType *gamerules.BlockType, ok bool) {
	index, subLoc, ok := chunk.getBlockIndexByBlockXyz(blockLoc)
	if !ok {
		return
	}

	blockType, blockData, ok := chunk.blockTypeAndData(index)
	if !ok {
		return
	}

	blockInstance = &gamerules.BlockInstance{
		Chunk:     chunk,
		BlockLoc:  *blockLoc,
		SubLoc:    *subLoc,
		Index:     index,
		BlockType: blockType,
		Data:      blockData,
	}

	return
}

func (chunk *Chunk) reqHitBlock(player gamerules.IPlayerClient, held gamerules.Slot, digStatus DigStatus, target *BlockXyz, face Face) {

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

func (chunk *Chunk) reqInteractBlock(player gamerules.IPlayerClient, held gamerules.Slot, target *BlockXyz, againstFace Face) {
	// TODO use held item to better check of if the player is trying to place a
	// block vs. perform some other interaction (e.g hoeing dirt). This is
	// perhaps best solved by sending held item type and the face to
	// blockType.Aspect.Interact()

	blockInstance, blockType, ok := chunk.blockInstanceAndType(target)
	if !ok {
		return
	}

	if _, isBlockHeld := held.ItemTypeId.ToBlockId(); isBlockHeld && blockType.Attachable {
		// The player is interacting with a block that can be attached to.

		// Work out the position to put the block at.
		dx, dy, dz := againstFace.Dxyz()
		destLoc := target.AddXyz(dx, dy, dz)
		if destLoc == nil {
			// there is overflow with the translation, so do nothing
			return
		}

		player.PlaceHeldItem(*destLoc, held)
	} else {
		// Player is otherwise interacting with the block.
		blockType.Aspect.Interact(blockInstance, player)
	}

	return
}

// placeBlock attempts to place a block. This is called by PlayerBlockInteract
// in the situation where the player interacts with an attachable block
// (potentially in a different chunk to the one where the block gets placed).
func (chunk *Chunk) reqPlaceItem(player gamerules.IPlayerClient, target *BlockXyz, slot *gamerules.Slot) {
	// TODO defer a check for remaining items in slot, and do something with them
	// (send to player or drop on the ground).

	// TODO more flexible item checking for block placement (e.g placing seed
	// items on farmland doesn't fit this current simplistic model). The block
	// type for the block being placed against should probably contain this logic
	// (i.e farmland block should know about the seed item).
	heldBlockType, ok := slot.ItemTypeId.ToBlockId()
	if !ok || slot.Count < 1 {
		// Not a placeable item.
		return
	}

	index, subLoc, ok := chunk.getBlockIndexByBlockXyz(target)
	if !ok {
		return
	}

	// Blocks can only replace certain blocks.
	blockTypeId := index.BlockId(chunk.blocks)
	blockType, ok := gamerules.Blocks.Get(blockTypeId)
	if !ok || !blockType.Replaceable {
		return
	}

	// Safe to replace block.
	chunk.setBlock(target, subLoc, index, heldBlockType, byte(slot.Data))

	slot.Decrement()
}

func (chunk *Chunk) reqTakeItem(player gamerules.IPlayerClient, entityId EntityId) {
	if entity, ok := chunk.entities[entityId]; ok {
		if item, ok := entity.(*gamerules.Item); ok {
			player.GiveItemAtPosition(*item.Position(), *item.GetSlot())

			// Tell all subscribers to animate the item flying at the
			// player.
			buf := new(bytes.Buffer)
			proto.WriteItemCollect(buf, entityId, player.GetEntityId())
			chunk.reqMulticastPlayers(-1, buf.Bytes())
			chunk.removeEntity(item)
		}
	}
}

func (chunk *Chunk) reqDropItem(player gamerules.IPlayerClient, content *gamerules.Slot, position *AbsXyz, velocity *AbsVelocity, pickupImmunity Ticks) {
	spawnedItem := gamerules.NewItem(
		content.ItemTypeId,
		content.Count,
		content.Data,
		position,
		velocity,
		pickupImmunity,
	)

	chunk.AddEntity(spawnedItem)
}

func (chunk *Chunk) reqInventoryClick(player gamerules.IPlayerClient, blockLoc *BlockXyz, click *gamerules.Click) {
	blockInstance, blockType, ok := chunk.blockInstanceAndType(blockLoc)
	if !ok {
		return
	}

	blockType.Aspect.InventoryClick(blockInstance, player, click)
}

func (chunk *Chunk) reqInventoryUnsubscribed(player gamerules.IPlayerClient, blockLoc *BlockXyz) {
	blockInstance, blockType, ok := chunk.blockInstanceAndType(blockLoc)
	if !ok {
		return
	}

	blockType.Aspect.InventoryUnsubscribed(blockInstance, player)
}

// Used to read the BlockId of a block that's either in the chunk, or
// immediately adjoining it in a neighbouring chunk. In cases where the block
// type can't be determined we assume that the block asked about is solid
// (this way objects don't fly off the side of the map needlessly).
func (chunk *Chunk) BlockQuery(blockLoc BlockXyz) (isSolid bool, isWithinChunk bool) {
	chunkLoc, subLoc := blockLoc.ToChunkLocal()

	var blockTypeId BlockId
	var ok bool

	if chunkLoc.X == chunk.loc.X && chunkLoc.Z == chunk.loc.Z {
		// The item is asking about this chunk.
		index, ok := subLoc.BlockIndex()
		if !ok {
			log.Printf("%s.PhysicsBlockQuery(%#v) got bad block index", chunk, blockLoc)
			isSolid = true
			return
		}

		blockTypeId = index.BlockId(chunk.blocks)
		isWithinChunk = true
	} else {
		// The item is asking about a separate chunk.
		isWithinChunk = false

		blockTypeId, ok = chunk.shard.blockQuery(*chunkLoc, subLoc)

		if !ok {
			// The block isn't known.
			isSolid = true
			return
		}
	}

	if blockType, ok := gamerules.Blocks.Get(blockTypeId); ok {
		isSolid = blockType.Solid
	} else {
		log.Printf(
			"%s.PhysicsBlockQuery found unknown block type Id %d at %+v",
			chunk, blockTypeId, blockLoc)
		// The block type isn't known.
		isSolid = true
	}

	return
}

func (chunk *Chunk) tick() {
	chunk.spawnTick()

	chunk.blockTick()
}

// spawnTick runs all spawns for a tick.
func (chunk *Chunk) spawnTick() {
	if len(chunk.entities) == 0 {
		// Nothing to do, bail out early.
		return
	}

	outgoingEntities := []object.INonPlayerEntity{}

	for _, e := range chunk.entities {
		if e.Tick(chunk) {
			if e.Position().Y <= 0 {
				// Item or mob fell out of the world.
				chunk.removeEntity(e)
			} else {
				outgoingEntities = append(outgoingEntities, e)
			}
		}
	}

	if len(outgoingEntities) > 0 {
		// Transfer spawns to new chunk.
		for _, e := range outgoingEntities {
			// Remove mob/items from this chunk.
			chunk.entities[e.GetEntityId()] = nil, false

			// Transfer to other chunk.
			chunkLoc := e.Position().ToChunkXz()
			shardLoc := chunkLoc.ToShardXz()

			// TODO Batch spawns up into a request per shard if there are efficiency
			// concerns in sending them individually.
			shardClient := chunk.shard.clientForShard(shardLoc)
			if shardClient != nil {
				shardClient.ReqTransferEntity(chunkLoc, e)
			}
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
					m := mob.NewPig(&playerData.position, &AbsVelocity{5, 5, 5}, &LookDegrees{0, 0})
					chunk.AddEntity(&m.Mob)
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
	var blockInstance gamerules.BlockInstance
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
	for _, e := range chunk.entities {
		switch e.(type) {
		case *mob.Mob:
			s = append(s, e.(*mob.Mob))
		}
	}
	return
}

func (chunk *Chunk) items() (s []*gamerules.Item) {
	s = make([]*gamerules.Item, 0, 10)
	for _, e := range chunk.entities {
		switch e.(type) {
		case *gamerules.Item:
			s = append(s, e.(*gamerules.Item))
		}
	}
	return
}

func (chunk *Chunk) reqSubscribeChunk(entityId EntityId, player gamerules.IPlayerClient, notify bool) {
	if _, ok := chunk.subscribers[entityId]; ok {
		// Already subscribed.
		return
	}

	chunk.subscribers[entityId] = player

	buf := new(bytes.Buffer)
	proto.WritePreChunk(buf, &chunk.loc, ChunkInit)
	player.TransmitPacket(buf.Bytes())

	player.TransmitPacket(chunk.chunkPacket())
	if notify {
		player.NotifyChunkLoad()
	}

	// Send spawns packets for all entities in the chunk.
	if len(chunk.entities) > 0 {
		buf := new(bytes.Buffer)
		for _, e := range chunk.entities {
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
func (chunk *Chunk) AddOnUnsubscribe(entityId EntityId, observer gamerules.IUnsubscribed) {
	observers := chunk.onUnsub[entityId]
	observers = append(observers, observer)
	chunk.onUnsub[entityId] = observers
}

// RemoveOnUnsubscribe removes a function previously registered
func (chunk *Chunk) RemoveOnUnsubscribe(entityId EntityId, observer gamerules.IUnsubscribed) {
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

func (chunk *Chunk) reqSetPlayerPosition(entityId EntityId, pos AbsXyz) {
	data, ok := chunk.playersData[entityId]

	if !ok {
		log.Printf(
			"%v.reqSetPlayerPosition: called for EntityId (%d) not present as playerData.",
			chunk, entityId,
		)
		return
	}

	data.position = pos

	// Update subscribers.
	buf := new(bytes.Buffer)
	data.sendPositionLook(buf)
	chunk.reqMulticastPlayers(entityId, buf.Bytes())

	player, ok := chunk.subscribers[entityId]

	if ok {
		// Does the player overlap with any items?
		for _, item := range chunk.items() {
			if item.PickupImmunity > 0 {
				item.PickupImmunity--
				continue
			}
			// TODO This check should be performed when items move as well.
			if data.OverlapsItem(item) {
				slot := item.GetSlot()
				player.OfferItem(chunk.loc, item.EntityId, *slot)
			}
		}
	}
}

func (chunk *Chunk) reqSetPlayerLook(entityId EntityId, look LookBytes) {
	data, ok := chunk.playersData[entityId]

	if !ok {
		log.Printf(
			"%v.reqSetPlayerLook: called for EntityId (%d) not present as playerData.",
			chunk, entityId,
		)
		return
	}

	data.look = look

	// Update subscribers.
	buf := new(bytes.Buffer)
	data.sendPositionLook(buf)
	chunk.reqMulticastPlayers(entityId, buf.Bytes())
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
	for _, e := range chunk.entities {
		e.SendUpdate(buf)
	}
	chunk.reqMulticastPlayers(-1, buf.Bytes())
}

func (chunk *Chunk) isSameChunk(otherChunkLoc *ChunkXz) bool {
	return otherChunkLoc.X == chunk.loc.X && otherChunkLoc.Z == chunk.loc.Z
}

func (chunk *Chunk) addEntities(entities []nbt.ITag) {
	for _, entity := range entities {
		// Position within the chunk
		pos, err := nbtutil.ReadAbsXyz(entity, "Pos")
		if err != nil {
			continue
		}

		// Motion
		velocity, err := nbtutil.ReadAbsVelocity(entity, "Motion")
		if err != nil {
			continue
		}

		// Look
		look, err := nbtutil.ReadLookDegrees(entity, "Rotation")
		if err != nil {
			continue
		}

		_ = entity.Lookup("OnGround").(*nbt.Byte).Value
		_ = entity.Lookup("FallDistance").(*nbt.Float).Value
		_ = entity.Lookup("Air").(*nbt.Short).Value
		_ = entity.Lookup("Fire").(*nbt.Short).Value

		var newEntity object.INonPlayerEntity
		entityObjectId := entity.Lookup("id").(*nbt.String).Value

		switch entityObjectId {
		case "Item":
			itemInfo := entity.Lookup("Item").(*nbt.Compound)

			// Grab the basic item data
			id := ItemTypeId(itemInfo.Lookup("id").(*nbt.Short).Value)
			count := ItemCount(itemInfo.Lookup("Count").(*nbt.Byte).Value)
			data := ItemData(itemInfo.Lookup("Damage").(*nbt.Short).Value)
			newEntity = gamerules.NewItem(id, count, data, &pos, &velocity, 0)
		case "Chicken":
			newEntity = mob.NewHen(&pos, &velocity, &look)
		case "Cow":
			newEntity = mob.NewCow(&pos, &velocity, &look)
		case "Creeper":
			newEntity = mob.NewCreeper(&pos, &velocity, &look)
		case "Pig":
			newEntity = mob.NewPig(&pos, &velocity, &look)
		case "Sheep":
			newEntity = mob.NewSheep(&pos, &velocity, &look)
		case "Skeleton":
			newEntity = mob.NewSkeleton(&pos, &velocity, &look)
		case "Squid":
			newEntity = mob.NewSquid(&pos, &velocity, &look)
		case "Spider":
			newEntity = mob.NewSpider(&pos, &velocity, &look)
		case "Wolf":
			newEntity = mob.NewWolf(&pos, &velocity, &look)
		case "Zombie":
			newEntity = mob.NewZombie(&pos, &velocity, &look)
		default:
			// Handle all other objects
			objType, ok := ObjTypeMap[entityObjectId]
			if ok {
				newEntity = object.NewObject(objType, &pos, &velocity)
			} else {
				log.Printf("Found unhandled entity type: %s", entityObjectId)
			}
		}

		if newEntity != nil {
			entityId := chunk.shard.entityMgr.NewEntity()
			newEntity.SetEntityId(entityId)
			chunk.entities[entityId] = newEntity
		}

	}
	return
}
