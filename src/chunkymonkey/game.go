package chunkymonkey

import (
	"bytes"
	"fmt"
	"log"
	"math/rand"
	"net"
	"regexp"
	"time"

	"chunkymonkey/command"
	. "chunkymonkey/entity"
	"chunkymonkey/gamerules"
	"chunkymonkey/player"
	"chunkymonkey/proto"
	"chunkymonkey/server_auth"
	"chunkymonkey/shardserver"
	. "chunkymonkey/types"
	"chunkymonkey/worldstore"
	"nbt"
)

// We regard usernames as valid if they don't contain "dangerous" characters.
// That is: characters that might be abused in filename components, etc.
var validPlayerUsername = regexp.MustCompile(`^[\-a-zA-Z0-9_]+$`)

type Game struct {
	shardManager  *shardserver.LocalShardManager
	entityManager EntityManager
	worldStore    *worldstore.WorldStore
	connHandler   *ConnHandler

	// Mapping between entityId/name and player object
	players     map[EntityId]*player.Player
	playerNames map[string]*player.Player

	// Channels for events/actions
	workQueue        chan func(*Game)
	playerConnect    chan *player.Player
	playerDisconnect chan EntityId

	// Server information
	time           Ticks
	serverId       string
	maintenanceMsg string // if set, logins are disallowed.
}

func NewGame(worldPath string, listener net.Listener, serverDesc, maintenanceMsg string, maxPlayerCount int) (game *Game, err error) {
	worldStore, err := worldstore.LoadWorldStore(worldPath)
	if err != nil {
		return nil, err
	}

	authserver, err := server_auth.NewServerAuth("http://www.minecraft.net/game/checkserver.jsp")
	if err != nil {
		return
	}

	game = &Game{
		players:          make(map[EntityId]*player.Player),
		playerNames:      make(map[string]*player.Player),
		workQueue:        make(chan func(*Game), 256),
		playerConnect:    make(chan *player.Player),
		playerDisconnect: make(chan EntityId),
		time:             worldStore.Time,
		worldStore:       worldStore,
	}

	game.entityManager.Init()

	game.serverId = fmt.Sprintf("%016x", rand.NewSource(worldStore.Seed).Int63())
	//game.serverId = "-"

	game.shardManager = shardserver.NewLocalShardManager(worldStore.ChunkStore, &game.entityManager)

	// TODO: Load the prefix from a config file
	gamerules.CommandFramework = command.NewCommandFramework("/")

	// Start accepting connections.
	game.connHandler = NewConnHandler(listener, &GameInfo{
		game:           game,
		maxPlayerCount: maxPlayerCount,
		serverDesc:     serverDesc,
		maintenanceMsg: maintenanceMsg,
		serverId:       game.serverId,
		shardManager:   game.shardManager,
		entityManager:  &game.entityManager,
		worldStore:     game.worldStore,
		authserver:     authserver,
	})

	return
}

// Fetch external events and respond appropriately.
func (game *Game) Serve() {
	defer game.connHandler.Stop()

	ticker := time.NewTicker(NanosecondsInSecond / TicksPerSecond)

	for {
		select {
		case f := <-game.workQueue:
			f(game)
		case <-ticker.C:
			game.onTick()
		case player := <-game.playerConnect:
			game.onPlayerConnect(player)
		case entityId := <-game.playerDisconnect:
			game.onPlayerDisconnect(entityId)
		}
	}
}

// A new player has connected to the server
func (game *Game) onPlayerConnect(newPlayer *player.Player) {
	game.players[newPlayer.GetEntityId()] = newPlayer
	game.playerNames[newPlayer.Name()] = newPlayer
}

// A player has disconnected from the server
func (game *Game) onPlayerDisconnect(entityId EntityId) {
	oldPlayer := game.players[entityId]
	delete(game.players, entityId)
	delete(game.playerNames, oldPlayer.Name())
	game.entityManager.RemoveEntityById(entityId)

	playerData := nbt.NewCompound()
	if err := oldPlayer.MarshalNbt(playerData); err != nil {
		log.Printf("Failed to marshal player data: %v", err)
		return
	}

	if err := game.worldStore.WritePlayerData(oldPlayer.Name(), playerData); err != nil {
		log.Printf("Failed when writing player data: %v", err)
	}
}

func (game *Game) onTick() {
	game.time++
	if game.time%TicksPerSecond == 0 {
		game.sendTimeUpdate()
	}
}

// Utility functions

// Send a time/keepalive packet
func (game *Game) sendTimeUpdate() {
	buf := new(bytes.Buffer)
	proto.ServerWriteTimeUpdate(buf, game.time)

	game.multicastPacket(buf.Bytes(), nil)
}

// Send a packet to every player connected to the server
func (game *Game) multicastPacket(packet []byte, except interface{}) {
	for _, player := range game.players {
		if player == except {
			continue
		}

		player.TransmitPacket(packet)
	}
}

// Safely enqueue some work to be executed at some point in the future
func (game *Game) enqueue(f func(*Game)) {
	game.workQueue <- f
}

// The following functions implement the IGame interface

func (game *Game) BroadcastPacket(packet []byte) {
	game.enqueue(func(_ *Game) {
		game.multicastPacket(packet, nil)
	})
}

func (game *Game) BroadcastMessage(msg string) {
	buf := new(bytes.Buffer)
	proto.WriteChatMessage(buf, msg)
	game.BroadcastPacket(buf.Bytes())
}

func (game *Game) ItemTypeById(id int) (gamerules.ItemType, bool) {
	itemType, ok := gamerules.Items[ItemTypeId(id)]
	return *itemType, ok
}

func (game *Game) PlayerCount() int {
	result := make(chan int)
	game.enqueue(func(_ *Game) {
		result <- len(game.players)
	})
	return <-result
}

func (game *Game) PlayerByEntityId(id EntityId) gamerules.IPlayerClient {
	result := make(chan gamerules.IPlayerClient)
	game.enqueue(func(_ *Game) {
		player, ok := game.players[id]
		if ok {
			result <- player.Client()
		} else {
			result <- nil
		}
		close(result)
	})
	return <-result
}

func (game *Game) PlayerByName(name string) gamerules.IPlayerClient {
	result := make(chan gamerules.IPlayerClient)
	game.enqueue(func(_ *Game) {
		player, ok := game.playerNames[name]
		if ok {
			result <- player.Client()
		} else {
			result <- nil
		}
		close(result)
	})
	return <-result
}
