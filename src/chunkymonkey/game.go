package chunkymonkey

import (
	"bytes"
	"fmt"
	"log"
	"net"
	"os"
	"rand"
	"regexp"
	"time"

	. "chunkymonkey/entity"
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
	chunkManager     *shardserver.LocalShardManager
	mainQueue        chan func(*Game)
	playerDisconnect chan EntityId
	entityManager    EntityManager
	players          map[EntityId]*player.Player
	time             Ticks
	serverId         string
	worldStore       *worldstore.WorldStore
	// If set, logins are not allowed.
	UnderMaintenanceMsg string
}

func NewGame(worldPath string) (game *Game, err os.Error) {
	worldStore, err := worldstore.LoadWorldStore(worldPath)
	if err != nil {
		return nil, err
	}

	game = &Game{
		mainQueue:        make(chan func(*Game), 256),
		playerDisconnect: make(chan EntityId),
		players:          make(map[EntityId]*player.Player),
		time:             worldStore.Time,
		worldStore:       worldStore,
	}

	game.entityManager.Init()

	game.serverId = fmt.Sprintf("%016x", rand.NewSource(worldStore.Seed).Int63())
	//game.serverId = "-"

	game.chunkManager = shardserver.NewLocalShardManager(worldStore.ChunkStore, &game.entityManager)

	go game.mainLoop()
	return
}

// login negotiates a player client login, and adds a new player if successful.
// Note that it does not run in the game's goroutine.
func (game *Game) login(conn net.Conn) {
	username, err := proto.ServerReadHandshake(conn)

	if !validPlayerUsername.MatchString(username) {
		proto.WriteDisconnect(conn, "Bad username")
		conn.Close()
		return
	}

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

	entityId := game.entityManager.NewEntity()

	playerData, err := game.worldStore.PlayerData(username)
	var startPosition AbsXyz
	if err != nil {
		log.Printf("Error reading player data for %q: %v", username, err)
		proto.WriteDisconnect(conn, "error reading your user data")
		conn.Close()
		return
	} else if playerData != nil {
		// Data for this player already exists in the world, so attempt to
		// load that and use the player's last position instead of the spawn
		// position.
		posData, ok := playerData.Lookup("/Pos").(*nbt.List)

		if ok {
			posList := posData.Value
			if len(posList) == 3 { // Paranoid check for valid data
				x, xok := posList[0].(*nbt.Double)
				y, yok := posList[1].(*nbt.Double)
				z, zok := posList[2].(*nbt.Double)
				if xok && yok && zok {
					startPosition = AbsXyz{
						AbsCoord(x.Value),
						AbsCoord(y.Value),
						AbsCoord(z.Value),
					}
				}
			}
		}
	} else {
		// Player hasn't logged in before.
		spawnPos := &game.worldStore.SpawnPosition
		startPosition.X = AbsCoord(spawnPos.X)
		startPosition.Y = AbsCoord(spawnPos.Y)
		startPosition.Z = AbsCoord(spawnPos.Z)
	}

	// Player seems to fall through block unless elevated very slightly.
	startPosition.Y += 0.01

	player := player.NewPlayer(entityId, game.chunkManager, conn, username, startPosition, game.playerDisconnect)

	addedChan := make(chan struct{})
	game.enqueue(func(_ *Game) {
		game.addPlayer(player)
		addedChan <- struct{}{}
	})
	_ = <-addedChan

	buf := &bytes.Buffer{}
	// TODO pass proper dimension. This is low priority, because we don't yet
	// support multiple dimensions.
	proto.ServerWriteLogin(buf, player.EntityId, 0, DimensionNormal)
	proto.WriteSpawnPosition(buf, &game.worldStore.SpawnPosition)
	player.TransmitPacket(buf.Bytes())

	player.Start()
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

// addPlayer adds the player to the set of connected players.
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
