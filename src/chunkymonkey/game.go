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
	chunkManager  *shardserver.LocalShardManager
	entityManager EntityManager
	worldStore    *worldstore.WorldStore

	// Mapping between entityId/name and player object
	players     map[EntityId]*player.Player
	playerNames map[string]*player.Player

	// Channels for events/actions
	workQueue        chan func(*Game)
	playerConnect    chan *player.Player
	playerDisconnect chan EntityId

	// Server information
	time                Ticks
	serverId            string
	UnderMaintenanceMsg string // if set, logins are disallowed.
}

func NewGame(worldPath string) (game *Game, err os.Error) {
	worldStore, err := worldstore.LoadWorldStore(worldPath)
	if err != nil {
		return nil, err
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

	game.chunkManager = shardserver.NewLocalShardManager(worldStore.ChunkStore, &game.entityManager)

	// TODO: Load the prefix from a config file
	gamerules.CommandFramework = command.NewCommandFramework("/")

	go game.mainLoop()
	return
}

// Fetch external events and respond appropriately
func (game *Game) mainLoop() {
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
	game.players[entityId] = nil, false
	game.playerNames[oldPlayer.Name()] = nil, false
	game.entityManager.RemoveEntityById(entityId)

	playerData := oldPlayer.WriteNbt()

	if err := game.worldStore.WritePlayerData(oldPlayer.Name(), playerData); err != nil {
		log.Printf("Failed when writing player data: %s", err)
	}
}

func (game *Game) onTick() {
	game.time++
	if game.time%TicksPerSecond == 0 {
		game.sendTimeUpdate()
	}
}

// Negotiate a new player client login. This function runs in a new goroutine
// for each player connection and therefore should not attempt to alter the
// game structure without enqueue().
func (game *Game) login(conn net.Conn) {
	// TODO: This function should access game in a thread-safe way for reading
	var err, clientErr os.Error

	defer func() {
		if err != nil {
			log.Print(err.String())
			if clientErr == nil {
				clientErr = os.NewError("Server error.")
			}
			proto.WriteDisconnect(conn, clientErr.String())
			conn.Close()
		}
	}()

	var username string
	if username, err = proto.ServerReadHandshake(conn); err != nil {
		clientErr = os.NewError("Handshake error.")
		return
	}

	if !validPlayerUsername.MatchString(username) {
		err = os.NewError("Bad username")
		clientErr = err
		return
	}

	log.Print("Client ", conn.RemoteAddr(), " connected as ", username)

	if game.UnderMaintenanceMsg != "" {
		err = fmt.Errorf("Server under maintenance, kicking player: %q", username)
		clientErr = os.NewError(game.UnderMaintenanceMsg)
		return
	}

	// Load player permissions.
	permissions := gamerules.Permissions.UserPermissions(username)
	if !permissions.Has("login") {
		err = fmt.Errorf("Player %q does not have login permission", username)
		clientErr = os.NewError("You do not have access to this server.")
		return
	}

	if err = proto.ServerWriteHandshake(conn, game.serverId); err != nil {
		clientErr = os.NewError("Handshake error.")
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
			err = fmt.Errorf("Client %v: %s", conn.RemoteAddr(), reason)
			clientErr = os.NewError(reason)
			return
		}
		log.Print("Client ", conn.RemoteAddr(), " passed minecraft.net authentication")
	}

	if _, err = proto.ServerReadLogin(conn); err != nil {
		clientErr = os.NewError("Login error.")
		return
	}

	entityId := game.entityManager.NewEntity()

	var playerData nbt.ITag
	if playerData, err = game.worldStore.PlayerData(username); err != nil {
		clientErr = os.NewError("Error reading user data. Please contact the server administrator.")
		return
	}

	player := player.NewPlayer(entityId, game.chunkManager, conn, username, game.worldStore.SpawnPosition, game.playerDisconnect, game)
	if playerData != nil {
		if err = player.ReadNbt(playerData); err != nil {
			// Don't let the player log in, as they will only have default inventory
			// etc., which could lose items from them. Better for an administrator to
			// sort this out.
			err = fmt.Errorf("Error parsing player data for %q: %v", username, err)
			clientErr = os.NewError("Error reading user data. Please contact the server administrator.")
			return
		}
	}

	game.playerConnect <- player
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
			break
		}

		go game.login(conn)
	}
}

// Utility functions

// Send a time/keepalive packet
func (game *Game) sendTimeUpdate() {
	buf := new(bytes.Buffer)
	proto.ServerWriteTimeUpdate(buf, game.time)

	// The "keep-alive" packet to client(s) sent here as well, as there
	// seems no particular reason to send time and keep-alive separately
	// for now.
	proto.WriteKeepAlive(buf)

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

func (game *Game) BroadcastMessage(msg string) {
	buf := new(bytes.Buffer)
	proto.WriteChatMessage(buf, msg)

	game.enqueue(func(_ *Game) {
		game.multicastPacket(buf.Bytes(), nil)
	})
}

func (game *Game) ItemTypeById(id int) (gamerules.ItemType, bool) {
	itemType, ok := gamerules.Items[ItemTypeId(id)]
	return *itemType, ok
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
