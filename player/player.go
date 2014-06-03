package player

import (
	"bytes"
	"errors"
	"expvar"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net"
	"sync"
	"time"

	"github.com/huin/chunkymonkey/gamerules"
	"github.com/huin/chunkymonkey/nbtutil"
	"github.com/huin/chunkymonkey/physics"
	"github.com/huin/chunkymonkey/proto"
	. "github.com/huin/chunkymonkey/types"
	"github.com/huin/chunkymonkey/window"
	"github.com/huin/chunkymonkey/nbt"
)

var (
	expVarPlayerConnectionCount    *expvar.Int
	expVarPlayerDisconnectionCount *expvar.Int
	errUnknownItemID               error

	playerPingNoCheck = flag.Bool(
		"player_ping_no_check", false,
		"Relax checks on player keep-alive packets. This can be useful for "+
			"recorded/replayed sessions.")
)

const (
	StanceNormal = AbsCoord(1.62)
	MaxHealth    = Health(20)
	MaxFoodUnits = FoodUnits(20)

	PingTimeoutNs  = 1e9 * 60 // Player connection times out after 60 seconds.
	PingIntervalNs = 1e9 * 20 // Time between receiving keep alive response from client and sending new request.
)

func init() {
	expVarPlayerConnectionCount = expvar.NewInt("player-connection-count")
	expVarPlayerDisconnectionCount = expvar.NewInt("player-disconnection-count")
	errUnknownItemID = errors.New("Unknown item ID")
}

type Player struct {
	// First attributes are for housekeeping etc.

	EntityId
	playerClient   playerClient
	shardConnecter gamerules.IShardConnecter
	conn           net.Conn
	name           string
	loginComplete  bool
	spawnComplete  bool

	game gamerules.IGame

	// ping is used to determine the player's current roundtrip latency, and to
	// determine if the player should be disconnected for not responding.
	ping struct {
		running     bool
		id          int32       // Last ID sent in keep-alive, or 0 if no current ping.
		timestampNs int64       // Nanoseconds since epoch since last keep-alive sent.
		timer       *time.Timer // Time until next ping, or timeout of current.
	}

	// TODO remove this lock, packet handling shouldn't use a lock, it should use
	// a channel instead (ideally).
	lock sync.Mutex

	onDisconnect chan<- EntityId
	mainQueue    chan func(*Player)
	txQueue      chan []byte
	txErrChan    chan error
	rxErrChan    chan error
	rxRunning    bool // Only used by the receiveLoop.
	stopPlayer   chan bool

	// The following attributes are game-logic related.

	// Data entries that may change
	spawnBlock BlockXyz
	position   AbsXyz
	height     AbsCoord
	look       LookDegrees
	chunkSubs  chunkSubscriptions
	health     Health
	food       FoodUnits

	// The following data fields are loaded, but not used yet
	dimension    int32
	onGround     int8
	sleeping     int8
	fallDistance float32
	sleepTimer   int16
	attackTime   int16
	deathTime    int16
	hurtTime     int16
	motion       AbsVelocity
	air          int16
	fire         int16

	cursor       gamerules.Slot // Item being moved by mouse cursor.
	inventory    window.PlayerInventory
	curWindow    window.IWindow
	nextWindowId WindowId
	remoteInv    *RemoteInventory
}

func NewPlayer(entityId EntityId, shardConnecter gamerules.IShardConnecter, conn net.Conn, name string, spawnBlock BlockXyz, onDisconnect chan<- EntityId, game gamerules.IGame) *Player {
	player := &Player{
		EntityId:       entityId,
		shardConnecter: shardConnecter,
		conn:           conn,
		name:           name,
		spawnBlock:     spawnBlock,
		position: AbsXyz{
			X: AbsCoord(spawnBlock.X),
			Y: AbsCoord(spawnBlock.Y),
			Z: AbsCoord(spawnBlock.Z),
		},
		height: StanceNormal,
		look:   LookDegrees{0, 0},

		health: MaxHealth,
		food:   MaxFoodUnits, // TODO: Check what initial level should be.

		curWindow:    nil,
		nextWindowId: WindowIdFreeMin,

		mainQueue:  make(chan func(*Player), 128),
		txQueue:    make(chan []byte, 128),
		txErrChan:  make(chan error, 1),
		rxErrChan:  make(chan error, 1),
		stopPlayer: make(chan bool, 1),

		game: game,

		onDisconnect: onDisconnect,
	}

	player.playerClient.Init(player)
	player.inventory.Init(player.EntityId, player)

	return player
}

func (player *Player) Name() string {
	return player.name
}

func (player *Player) String() string {
	return fmt.Sprintf("Player(%q)", player.name)
}

func (player *Player) Position() AbsXyz {
	return player.position
}

func (player *Player) SetPosition(pos AbsXyz) {
	player.position = pos
}

func (player *Player) Client() gamerules.IPlayerClient {
	return &player.playerClient
}

func (player *Player) Look() LookDegrees {
	return player.look
}

// UnmarshalNbt unpacks the player data from their persistantly stored NBT
// data. It must only be called before Player.Run().
func (player *Player) UnmarshalNbt(tag *nbt.Compound) (err error) {
	if player.position, err = nbtutil.ReadAbsXyz(tag, "Pos"); err != nil {
		return
	}

	if player.look, err = nbtutil.ReadLookDegrees(tag, "Rotation"); err != nil {
		return
	}

	health, err := nbtutil.ReadShort(tag, "Health")
	if err != nil {
		return
	}
	player.health = Health(health)

	if err = player.inventory.UnmarshalNbt(tag.Lookup("Inventory")); err != nil {
		return
	}

	if player.onGround, err = nbtutil.ReadByte(tag, "OnGround"); err != nil {
		return
	}

	if player.dimension, err = nbtutil.ReadInt(tag, "Dimension"); err != nil {
		return
	}

	if player.sleeping, err = nbtutil.ReadByte(tag, "Sleeping"); err != nil {
		return
	}

	if player.fallDistance, err = nbtutil.ReadFloat(tag, "FallDistance"); err != nil {
		return
	}

	if player.sleepTimer, err = nbtutil.ReadShort(tag, "SleepTimer"); err != nil {
		return
	}

	if player.attackTime, err = nbtutil.ReadShort(tag, "AttackTime"); err != nil {
		return
	}

	if player.deathTime, err = nbtutil.ReadShort(tag, "DeathTime"); err != nil {
		return
	}

	if player.motion, err = nbtutil.ReadAbsVelocity(tag, "Motion"); err != nil {
		return
	}

	if player.hurtTime, err = nbtutil.ReadShort(tag, "HurtTime"); err != nil {
		return
	}

	if player.air, err = nbtutil.ReadShort(tag, "Air"); err != nil {
		return
	}

	if player.fire, err = nbtutil.ReadShort(tag, "Fire"); err != nil {
		return
	}

	return nil
}

// MarshalNbt packs the player data into a nbt.Compound so it can be written to
// persistant storage.
func (player *Player) MarshalNbt(tag *nbt.Compound) (err error) {
	if err = player.inventory.MarshalNbt(tag); err != nil {
		return
	}

	tag.Set("OnGround", &nbt.Byte{player.onGround})
	tag.Set("Dimension", &nbt.Int{player.dimension})
	tag.Set("Sleeping", &nbt.Byte{player.sleeping})
	tag.Set("FallDistance", &nbt.Float{player.fallDistance})
	tag.Set("SleepTimer", &nbt.Short{player.sleepTimer})
	tag.Set("AttackTime", &nbt.Short{player.attackTime})
	tag.Set("DeathTime", &nbt.Short{player.deathTime})
	tag.Set("Motion", &nbt.List{nbt.TagDouble, []nbt.ITag{
		&nbt.Double{float64(player.motion.X)},
		&nbt.Double{float64(player.motion.Y)},
		&nbt.Double{float64(player.motion.Z)},
	}})
	tag.Set("HurtTime", &nbt.Short{player.hurtTime})
	tag.Set("Air", &nbt.Short{player.air})
	tag.Set("Rotation", &nbt.List{nbt.TagFloat, []nbt.ITag{
		&nbt.Float{float32(player.look.Yaw)},
		&nbt.Float{float32(player.look.Pitch)},
	}})
	tag.Set("Pos", &nbt.List{nbt.TagDouble, []nbt.ITag{
		&nbt.Double{float64(player.position.X)},
		&nbt.Double{float64(player.position.Y)},
		&nbt.Double{float64(player.position.Z)},
	}})
	tag.Set("Fire", &nbt.Short{player.fire})
	tag.Set("Health", &nbt.Short{int16(player.health)})

	return nil
}

func (player *Player) getHeldItemTypeId() ItemTypeId {
	heldSlot, _ := player.inventory.HeldItem()
	heldItemId := heldSlot.ItemTypeId
	if heldItemId < 0 {
		return 0
	}
	return heldItemId
}

func (player *Player) Run() {
	buf := &bytes.Buffer{}
	// TODO pass proper dimension. This is low priority, because we don't yet
	// support multiple dimensions.
	// TODO pass proper map seed.
	// TODO pass proper values for the difficulty.
	// TODO proper max number of players.
	proto.ServerWriteLogin(buf, player.EntityId, 0, 0, DimensionNormal, GameDifficultyNormal, MaxYCoord+1, 8)
	proto.WriteSpawnPosition(buf, &player.spawnBlock)
	player.TransmitPacket(buf.Bytes())

	go player.receiveLoop()
	go player.transmitLoop()
	go player.mainLoop()
}

func (player *Player) Stop() {
	// Don't block. If the channel has a message in already, then that's good
	// enough.
	select {
	case player.stopPlayer <- true:
	default:
	}
}

// Start of packet handling code
// Note: any packet handlers that could change the player state or read a
// changeable state must use player.lock

func (player *Player) PacketKeepAlive(id int32) {
	player.pingReceived(id)
}

func (player *Player) PacketServerLogin(username string) {
	// Unexpected packet.
	player.Stop()
}

func (player *Player) PacketServerHandshake(username string) {
	// Unexpected packet.
	player.Stop()
}

func (player *Player) PacketChatMessage(message string) {
	prefix := gamerules.CommandFramework.Prefix()
	if message[0:len(prefix)] == prefix {
		// We pass the IPlayerClient to the command framework to avoid having
		// to fetch it as the first part of every command.
		gamerules.CommandFramework.Process(&player.playerClient, message, player.game)
	} else {
		player.sendChatMessage(fmt.Sprintf("<%s> %s", player.name, message), true)
	}
}

func (player *Player) PacketEntityAction(entityId EntityId, action EntityAction) {
}

func (player *Player) PacketUseEntity(user EntityId, target EntityId, leftClick bool) {
}

func (player *Player) PacketRespawn(dimension DimensionId, unknown int8, gameType GameType, worldHeight int16, mapSeed RandomSeed) {
}

func (player *Player) PacketPlayer(onGround bool) {
}

func (player *Player) PacketPlayerPosition(position *AbsXyz, stance AbsCoord, onGround bool) {
	player.lock.Lock()
	defer player.lock.Unlock()

	if !player.spawnComplete {
		// Ignore position packets from player until spawned at initial position
		// with chunk loaded.
		return
	}

	if !player.position.IsWithinDistanceOf(position, 10) {
		log.Printf("Discarding player position that is too far removed (%.2f, %.2f, %.2f)",
			position.X, position.Y, position.Z)
		return
	}
	player.position = *position
	player.height = stance - position.Y
	player.chunkSubs.Move(position)

	// TODO: Should keep track of when players enter/leave their mutual radius
	// of "awareness". I.e a client should receive a RemoveEntity packet when
	// the player walks out of range, and no longer receive WriteEntityTeleport
	// packets for them. The converse should happen when players come in range
	// of each other.
}

func (player *Player) PacketPlayerLook(look *LookDegrees, onGround bool) {
	player.lock.Lock()
	defer player.lock.Unlock()

	// TODO input validation
	player.look = *look

	// Update playerData on current chunk.
	if shard, ok := player.chunkSubs.CurrentShardClient(); ok {
		shard.ReqSetPlayerLook(player.chunkSubs.curChunkLoc, *look.ToLookBytes())
	}
}

func (player *Player) PacketPlayerBlockHit(status DigStatus, target *BlockXyz, face Face) {
	player.lock.Lock()
	defer player.lock.Unlock()

	// This packet handles 'throwing' an item as well, with status = 4, and
	// the zero values for target and face, so check for that.
	if status == DigDropItem && target.IsZero() && face == 0 {
		blockLoc := player.position.ToBlockXyz()
		shardClient, _, ok := player.chunkSubs.ShardClientForBlockXyz(blockLoc)
		if !ok {
			return
		}

		var itemToThrow gamerules.Slot
		player.inventory.TakeOneHeldItem(&itemToThrow)
		if !itemToThrow.IsEmpty() {
			velocity := physics.VelocityFromLook(player.look, 0.50)
			position := player.position
			position.Y += player.height
			shardClient.ReqDropItem(itemToThrow, position, velocity, TicksPerSecond/2)
		}
		return
	}

	// Validate that the player is actually somewhere near the block.
	targetAbsPos := target.MidPointToAbsXyz()
	if !targetAbsPos.IsWithinDistanceOf(&player.position, MaxInteractDistance) {
		log.Printf("Player/PacketPlayerBlockHit: ignoring player dig at %v (too far away)", target)
		return
	}

	// TODO measure the dig time on the target block and relay to the shard to
	// stop speed hacking (based on block type and tool used - non-trivial).

	shardClient, _, ok := player.chunkSubs.ShardClientForBlockXyz(target)
	if ok {
		held, _ := player.inventory.HeldItem()
		shardClient.ReqHitBlock(held, *target, status, face)
	}
}

func (player *Player) PacketPlayerBlockInteract(itemId ItemTypeId, target *BlockXyz, face Face, amount ItemCount, uses ItemData) {
	if face < FaceMinValid || face > FaceMaxValid {
		// TODO sometimes FaceNull means something. This case should be covered.
		log.Printf("Player/PacketPlayerBlockInteract: invalid face %d", face)
		return
	}

	player.lock.Lock()
	defer player.lock.Unlock()

	// Validate that the player is actually somewhere near the block.
	targetAbsPos := target.MidPointToAbsXyz()
	if !targetAbsPos.IsWithinDistanceOf(&player.position, MaxInteractDistance) {
		log.Printf("Player/PacketPlayerBlockInteract: ignoring player interact at %v (too far away)", target)
		return
	}

	shardClient, _, ok := player.chunkSubs.ShardClientForBlockXyz(target)
	if ok {
		held, _ := player.inventory.HeldItem()
		shardClient.ReqInteractBlock(held, *target, face)
	}
}

func (player *Player) PacketHoldingChange(slotId SlotId) {
	player.lock.Lock()
	defer player.lock.Unlock()
	player.inventory.SetHolding(slotId)
}

func (player *Player) PacketEntityAnimation(entityId EntityId, animation EntityAnimation) {
}

func (player *Player) PacketWindowClose(windowId WindowId) {
	player.lock.Lock()
	defer player.lock.Unlock()

	player.closeCurrentWindow(false)
}

func (player *Player) PacketWindowClick(windowId WindowId, slotId SlotId, rightClick bool, txId TxId, shiftClick bool, expectedSlot *proto.WindowSlot) {
	player.lock.Lock()
	defer player.lock.Unlock()

	// Note that the expectedSlot parameter is currently ignored. The item(s)
	// involved are worked out from the server-side data.
	// TODO use the expectedSlot as a conditions for the click, and base the
	// transaction result on that.

	// Determine which inventory window is involved.
	// TODO support for more windows

	var clickedWindow window.IWindow
	if windowId == WindowIdInventory {
		clickedWindow = &player.inventory
	} else if player.curWindow != nil && player.curWindow.WindowId() == windowId {
		clickedWindow = player.curWindow
	} else {
		log.Printf(
			"Warning: ignored window click on unknown window ID %d",
			windowId)
	}

	expectedSlotContent := &gamerules.Slot{
		ItemTypeId: expectedSlot.ItemTypeId,
		Count:      expectedSlot.Count,
		Data:       expectedSlot.Data,
	}
	// The client tends to send item IDs even when the count is zero.
	expectedSlotContent.Normalize()

	txState := TxStateRejected

	click := gamerules.Click{
		SlotId:     slotId,
		Cursor:     player.cursor,
		RightClick: rightClick,
		ShiftClick: shiftClick,
		TxId:       txId,
	}
	click.ExpectedSlot.SetWindowSlot(expectedSlot)

	if clickedWindow != nil {
		txState = clickedWindow.Click(&click)
	}

	switch txState {
	case TxStateAccepted, TxStateRejected:
		// Inform client of operation status.
		buf := new(bytes.Buffer)
		proto.WriteWindowTransaction(buf, windowId, txId, txState == TxStateAccepted)
		player.cursor = click.Cursor
		player.cursor.SendUpdate(buf, WindowIdCursor, SlotIdCursor)
		player.TransmitPacket(buf.Bytes())
	case TxStateDeferred:
		// The remote inventory should send the transaction outcome.
	}
}

func (player *Player) PacketWindowTransaction(windowId WindowId, txId TxId, accepted bool) {
	// TODO investigate when this packet is sent from the client and what it
	// means when it does get sent.
	log.Printf(
		"Got PacketWindowTransaction from player %q: windowId=%d txId=%d accepted=%t",
		player.name, windowId, txId, accepted)
}

func (player *Player) PacketSignUpdate(position *BlockXyz, lines [4]string) {
}

func (player *Player) PacketServerListPing() {
	// Shouldn't receive this packet once logged in.
}

func (player *Player) PacketDisconnect(reason string) {
	log.Printf("Player %s disconnected reason=%s", player.name, reason)

	player.sendChatMessage(fmt.Sprintf("%s has left", player.name), false)

	player.Stop()
}

func (player *Player) receiveLoop() {
	player.rxRunning = true
	for player.rxRunning {
		err := proto.ServerReadPacket(player.conn, player)
		if err != nil {
			player.rxErrChan <- err
			return
		}
	}
}

// End of packet handling code

func (player *Player) transmitLoop() {
	for {
		bs := <-player.txQueue

		if bs == nil {
			player.txErrChan <- nil
			return // txQueue closed
		}
		_, err := player.conn.Write(bs)
		if err != nil {
			player.txErrChan <- err
			return
		}
	}
}

func (player *Player) TransmitPacket(packet []byte) {
	if packet == nil {
		return // skip empty packets
	}
	player.txQueue <- packet
}

func (player *Player) runQueuedCall(f func(*Player)) {
	player.lock.Lock()
	defer player.lock.Unlock()
	f(player)
}

// pingNew starts a new "keep-alive" ping.
func (player *Player) pingNew() {
	if player.ping.running {
		log.Printf("%v: Attempted to start a ping while another is running.", player)
	} else {
		if player.ping.timer != nil {
			player.ping.timer.Stop()
		}

		player.ping.running = true
		player.ping.id = rand.Int31()
		if player.ping.id == 0 {
			// Ping ID 0 is used by the client on occasion. Don't use this ID to
			// avoid misreading keep alive IDs.
			player.ping.id = 1
		}
		player.ping.timestampNs = time.Now().Unix()

		buf := new(bytes.Buffer)
		proto.WriteKeepAlive(buf, player.ping.id)
		player.TransmitPacket(buf.Bytes())

		player.ping.timer = time.NewTimer(PingTimeoutNs)
	}
}

// pingTimeout handles pinging the client, or timing out the connection.
func (player *Player) pingTimeout() {
	if player.ping.running {
		// Current ping timed out. Disconnect the player.
		player.Stop()
	} else {
		// No ping in progress. Send a new one.
		player.pingNew()
	}
}

// pingReceived is called when a keep alive packet is received.
func (player *Player) pingReceived(id int32) {
	if id == 0 {
		// Client-initiated keep-alive.
		return
	}

	if *playerPingNoCheck {
		if !player.ping.running {
			return
		}
	} else {
		if !player.ping.running {
			log.Printf("%v: Received keep-alive id=%d when none was running", player, id)
			player.Stop()
			return
		} else if id != player.ping.id {
			log.Printf("%v: Bad keep alive ID received", player)
			player.Stop()
			return
		}
	}

	// Received valid keep-alive.
	now := time.Now().Unix()

	if player.ping.timer != nil {
		player.ping.timer.Stop()
	}

	latencyNs := now - player.ping.timestampNs
	// Check that there wasn't an apparent time-shift on this before broadcasting
	// this latency value.
	if latencyNs >= 0 && latencyNs < PingTimeoutNs {
		buf := new(bytes.Buffer)
		proto.WriteUserListItem(buf, player.name, true, int16(latencyNs/1e6))
		player.game.BroadcastPacket(buf.Bytes())
	}

	player.ping.running = false
	player.ping.id = 0
	player.ping.timer = time.NewTimer(PingIntervalNs)
}

func (player *Player) mainLoop() {
	defer func() {
		// Close the transmitLoop and receiveLoop cleanly.
		player.txQueue <- nil
		player.conn.Close()

		player.onDisconnect <- player.EntityId

		if player.ping.timer != nil {
			player.ping.timer.Stop()
		}

		buf := new(bytes.Buffer)
		proto.WriteUserListItem(buf, player.name, false, 0)
		player.game.BroadcastPacket(buf.Bytes())
	}()

	expVarPlayerConnectionCount.Add(1)
	defer expVarPlayerDisconnectionCount.Add(1)

	player.chunkSubs.Init(player)
	defer player.chunkSubs.Close()

	// Start the keep-alive/latency pings.
	player.pingNew()

	player.sendChatMessage(fmt.Sprintf("%s has joined", player.name), false)

MAINLOOP:
	for {
		select {
		case _ = <-player.stopPlayer:
			break MAINLOOP

		case f, ok := <-player.mainQueue:
			if !ok {
				return
			}
			player.runQueuedCall(f)

		case _ = <-player.ping.timer.C:
			player.pingTimeout()

		case err := <-player.rxErrChan:
			log.Printf("%v: receive loop failed: %v", player, err)
			player.Stop()

		case err := <-player.txErrChan:
			log.Printf("%v: send loop failed: %v", player, err)
			player.Stop()
		}
	}
}

func (player *Player) notifyChunkLoad() {
	if !player.spawnComplete {
		player.spawnComplete = true

		// Player seems to fall through block unless elevated very slightly.
		player.position.Y += 0.01

		// Send player start position etc.
		buf := new(bytes.Buffer)
		proto.ServerWritePlayerPositionLook(
			buf,
			&player.position, player.position.Y+player.height,
			&player.look, false)
		player.inventory.WriteWindowItems(buf)
		proto.WriteUpdateHealth(buf, player.health, player.food, 0)

		player.TransmitPacket(buf.Bytes())
	}
}

func (player *Player) inventorySubscribed(block *BlockXyz, invTypeId InvTypeId, slots []proto.WindowSlot) {
	if player.remoteInv != nil {
		player.closeCurrentWindow(true)
	}

	remoteInv := NewRemoteInventory(block, &player.chunkSubs, slots)

	window := player.inventory.NewWindow(invTypeId, player.nextWindowId, remoteInv)
	if window == nil {
		return
	}

	player.remoteInv = remoteInv
	player.curWindow = window

	if player.nextWindowId >= WindowIdFreeMax {
		player.nextWindowId = WindowIdFreeMin
	} else {
		player.nextWindowId++
	}

	buf := new(bytes.Buffer)
	window.WriteWindowOpen(buf)
	window.WriteWindowItems(buf)
	player.TransmitPacket(buf.Bytes())
}

func (player *Player) inventorySlotUpdate(block *BlockXyz, slot *gamerules.Slot, slotId SlotId) {
	if player.remoteInv == nil || !player.remoteInv.IsForBlock(block) {
		return
	}

	player.remoteInv.slotUpdate(slot, slotId)
}

func (player *Player) inventoryProgressUpdate(block *BlockXyz, prgBarId PrgBarId, value PrgBarValue) {
	if player.remoteInv == nil || !player.remoteInv.IsForBlock(block) {
		return
	}

	player.remoteInv.progressUpdate(prgBarId, value)
}

func (player *Player) inventoryCursorUpdate(block *BlockXyz, cursor *gamerules.Slot) {
	if player.remoteInv == nil || !player.remoteInv.IsForBlock(block) {
		return
	}

	player.cursor = *cursor
	buf := new(bytes.Buffer)
	player.cursor.SendUpdate(buf, WindowIdCursor, SlotIdCursor)
	player.TransmitPacket(buf.Bytes())
}

func (player *Player) inventoryTxState(block *BlockXyz, txId TxId, accepted bool) {
	if player.remoteInv == nil || !player.remoteInv.IsForBlock(block) || player.curWindow == nil {
		return
	}

	buf := new(bytes.Buffer)
	proto.WriteWindowTransaction(buf, player.curWindow.WindowId(), txId, accepted)
	player.TransmitPacket(buf.Bytes())
}

func (player *Player) inventoryUnsubscribed(block *BlockXyz) {
	if player.remoteInv == nil || !player.remoteInv.IsForBlock(block) {
		return
	}

	player.closeCurrentWindow(true)
}

func (player *Player) placeHeldItem(target *BlockXyz, wasHeld *gamerules.Slot) {
	curHeld, _ := player.inventory.HeldItem()

	// Currently held item has changed since chunk saw it.
	// TODO think about having the slot index passed as well so if that changes,
	// we can still track the original item and improve placement success rate.
	if !curHeld.IsSameType(wasHeld) {
		return
	}

	shardClient, _, ok := player.chunkSubs.ShardClientForBlockXyz(target)
	if ok {
		var into gamerules.Slot

		player.inventory.TakeOneHeldItem(&into)

		shardClient.ReqPlaceItem(*target, into)
	}
}

// Used to receive items picked up from chunks. It is synchronous so that the
// passed item can be looked at by the caller afterwards to see if it has been
// consumed.
func (player *Player) offerItem(fromChunk *ChunkXz, entityId EntityId, item *gamerules.Slot) {
	if player.inventory.CanTakeItem(item) {
		shardClient, ok := player.chunkSubs.ShardClientForChunkXz(fromChunk)
		if ok {
			shardClient.ReqTakeItem(*fromChunk, entityId)
		}
	}

	return
}

func (player *Player) giveItem(atPosition *AbsXyz, item *gamerules.Slot) {
	defer func() {
		// Check if item not fully consumed. If it is not, then throw the remains
		// back to the chunk.
		if item.Count > 0 {
			chunkLoc := atPosition.ToChunkXz()
			shardClient, ok := player.chunkSubs.ShardClientForChunkXz(&chunkLoc)
			if ok {
				shardClient.ReqDropItem(*item, *atPosition, AbsVelocity{}, TicksPerSecond)
			}
		}
	}()

	player.inventory.PutItem(item)
}

// Enqueue queues a function to run with the player lock within the player's
// mainloop.
func (player *Player) Enqueue(f func(*Player)) {
	if f == nil {
		return
	}
	player.mainQueue <- f
}

func (player *Player) sendChatMessage(message string, sendToSelf bool) {
	buf := new(bytes.Buffer)
	proto.WriteChatMessage(buf, message)

	packet := buf.Bytes()

	if sendToSelf {
		player.TransmitPacket(packet)
	}

	player.chunkSubs.curShard.ReqMulticastPlayers(
		player.chunkSubs.curChunkLoc,
		player.EntityId,
		packet,
	)
}

// closeCurrentWindow closes any open window. It must be called with
// player.lock held.
func (player *Player) closeCurrentWindow(sendClosePacket bool) {
	if player.curWindow != nil {
		player.curWindow.Finalize(sendClosePacket)
		player.curWindow = nil
	}

	if player.remoteInv != nil {
		player.remoteInv.Close()
		player.remoteInv = nil
	}

	player.inventory.Resubscribe()
}

// setPositionLook sets the player's position and look angle. It also notifies
// other players in the area of interest that the player has moved.
func (player *Player) setPositionLook(pos AbsXyz, look LookDegrees) {
	player.position = pos
	player.look = look
	player.height = StanceNormal - pos.Y

	if player.chunkSubs.Move(&player.position) {
		// The destination chunk isn't loaded. Wait for it.
		player.spawnComplete = false
	} else {
		// Notify the player about their new position
		// Tell the player's client about their new position
		buf := new(bytes.Buffer)
		proto.WritePlayerPosition(buf, &pos, StanceNormal, true)
		player.TransmitPacket(buf.Bytes())
	}
}
