package chunkymonkey

import (
	"bytes"
	"fmt"
	"log"
	"net"
	"os"
	"rand"
	"time"

	. "chunkymonkey/entity"
	"chunkymonkey/gamerules"
	"chunkymonkey/itemtype"
	"chunkymonkey/player"
	"chunkymonkey/proto"
	"chunkymonkey/server_auth"
	"chunkymonkey/shardserver"
	. "chunkymonkey/types"
	"chunkymonkey/worldstore"
)

type Game struct {
	chunkManager     *shardserver.LocalShardManager
	mainQueue        chan func(*Game)
	playerDisconnect chan EntityId
	entityManager    EntityManager
	players          map[EntityId]*player.Player
	time             TimeOfDay
	gameRules        gamerules.GameRules
	itemTypes        itemtype.ItemTypeMap
	rand             *rand.Rand
	serverId         string
	worldStore       *worldstore.WorldStore
	// If set, logins are not allowed.
	UnderMaintenanceMsg string
}

func NewGame(worldPath string, gameRules *gamerules.GameRules) (game *Game, err os.Error) {
	worldStore, err := worldstore.LoadWorldStore(worldPath)

	game = &Game{
		mainQueue:        make(chan func(*Game), 256),
		playerDisconnect: make(chan EntityId),
		players:          make(map[EntityId]*player.Player),
		gameRules:        *gameRules,
		rand:             rand.New(rand.NewSource(time.UTC().Seconds())),
		worldStore:       worldStore,
	}

	game.entityManager.Init()

	game.serverId = fmt.Sprintf("%x", game.rand.Int63())
	//game.serverId = "-"

	game.chunkManager = shardserver.NewLocalShardManager(worldStore.ChunkStore, &game.entityManager, &game.gameRules)

	go game.mainLoop()
	return
}

// login negotiates a player client login, and adds a new player if successful.
// Note that it does not run in the game's goroutine.
func (game *Game) login(conn net.Conn) {
	username, err := proto.ServerReadHandshake(conn)
	if err != nil {
		log.Print("ServerReadHandshake: ", err.String())
		proto.WriteDisconnect(conn, err.String())
		conn.Close()
		return
	}
	log.Print("Client ", conn.RemoteAddr(), " connected as ", username)
	if game.UnderMaintenanceMsg != "" {
		log.Println("Server under maintenance, kicking player:", username)
		proto.WriteDisconnect(conn, game.UnderMaintenanceMsg)
		return
	}

	err = proto.ServerWriteHandshake(conn, game.serverId)
	if err != nil {
		log.Print("ServerWriteHandshake: ", err.String())
		proto.WriteDisconnect(conn, err.String())
		conn.Close()
		return
	}

	if game.serverId != "-" {
		var authenticated bool
		authserver := &server_auth.ServerAuth{"http://www.minecraft.net/game/checkserver.jsp"}
		authenticated, err = authserver.Authenticate(game.serverId, username)
		if !authenticated || err != nil {
			var reason string
			if err != nil {
				reason = "Authentication check failed: " + err.String()
			} else {
				reason = "Failed authentication"
			}
			log.Print("Client ", conn.RemoteAddr(), " ", reason)
			proto.WriteDisconnect(conn, reason)
			conn.Close()
			return
		}
		log.Print("Client ", conn.RemoteAddr(), " passed minecraft.net authentication")
	}

	_, err = proto.ServerReadLogin(conn)
	if err != nil {
		log.Print("ServerReadLogin: ", err.String())
		proto.WriteDisconnect(conn, err.String())
		conn.Close()
		return
	}

	startPosition := game.worldStore.StartPosition

	entityId := game.entityManager.NewEntity()

	// TODO pass player's last position in the world, not necessarily the spawn
	// position.
	player := player.NewPlayer(entityId, game.chunkManager, &game.gameRules, conn, username, startPosition, game.playerDisconnect)

	addedChan := make(chan struct{})
	game.enqueue(func(_ *Game) {
		game.addPlayer(player)
		addedChan <- struct{}{}
	})
	_ = <-addedChan

	player.Start()

	buf := &bytes.Buffer{}
	// TODO pass proper dimension. This is low priority, because there is
	// currently no way to update the client's dimension after login.
	proto.ServerWriteLogin(buf, player.EntityId, 0, DimensionNormal)
	proto.WriteSpawnPosition(buf, startPosition.ToBlockXyz())
	player.TransmitPacket(buf.Bytes())
}

func (game *Game) Serve(addr string) {
	listener, e := net.Listen("tcp", addr)
	if e != nil {
		log.Fatalf("Listen: %s", e.String())
	}
	log.Print("Listening on ", addr)

	for {
		conn, e2 := listener.Accept()
		if e2 != nil {
			log.Print("Accept: ", e2.String())
			continue
		}

		go game.login(conn)
	}
}

// addPlayer sends spawn messages to all players in range. It also spawns all
// existing players so the new player can see them.
func (game *Game) addPlayer(newPlayer *player.Player) {
	game.players[newPlayer.GetEntityId()] = newPlayer
}

func (game *Game) removePlayer(entityId EntityId) {
	game.players[entityId] = nil, false
	game.entityManager.RemoveEntityById(entityId)
}

func (game *Game) multicastPacket(packet []byte, except interface{}) {
	for _, player := range game.players {
		if player == except {
			continue
		}

		player.TransmitPacket(packet)
	}
}

func (game *Game) enqueue(f func(*Game)) {
	game.mainQueue <- f
}

func (game *Game) mainLoop() {
	ticker := time.NewTicker(NanosecondsInSecond / TicksPerSecond)

	for {
		select {
		case f := <-game.mainQueue:
			f(game)
		case <-ticker.C:
			game.tick()
		case entityId := <-game.playerDisconnect:
			game.removePlayer(entityId)
		}
	}
}

func (game *Game) sendTimeUpdate() {
	buf := new(bytes.Buffer)
	proto.ServerWriteTimeUpdate(buf, game.time)

	// The "keep-alive" packet to client(s) sent here as well, as there
	// seems no particular reason to send time and keep-alive separately
	// for now.
	proto.WriteKeepAlive(buf)

	game.multicastPacket(buf.Bytes(), nil)
}

func (game *Game) tick() {
	game.time++
	if game.time%TicksPerSecond == 0 {
		game.sendTimeUpdate()
	}
}
