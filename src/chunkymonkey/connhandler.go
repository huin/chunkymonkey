package chunkymonkey

import (
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net"

	. "chunkymonkey/entity"
	"chunkymonkey/gamerules"
	"chunkymonkey/player"
	"chunkymonkey/proto"
	"chunkymonkey/server_auth"
	"chunkymonkey/shardserver"
	"chunkymonkey/worldstore"
	"nbt"
)

const (
	connTypeUnknown = iota
	connTypeLogin
	connTypeServerQuery
)

var (
	clientErrGeneral      = errors.New("Server error.")
	clientErrUsername     = errors.New("Bad username.")
	clientErrLoginDenied  = errors.New("You do not have access to this server.")
	clientErrHandshake    = errors.New("Handshake error.")
	clientErrLoginGeneral = errors.New("Login error.")
	clientErrAuthFailed   = errors.New("Minecraft authentication failed.")
	clientErrUserData     = errors.New("Error reading user data. Please contact the server administrator.")

	loginErrorConnType    = errors.New("unknown/bad connection type")
	loginErrorMaintenance = errors.New("server under maintenance")
	loginErrorServerList  = errors.New("server list poll")
)

type GameInfo struct {
	game           *Game
	maxPlayerCount int
	serverDesc     string
	maintenanceMsg string
	shardManager   *shardserver.LocalShardManager
	entityManager  *EntityManager
	worldStore     *worldstore.WorldStore
	authserver     server_auth.IAuthenticator
}

// Handles connections for a game on the given socket.
type ConnHandler struct {
	// UpdateGameInfo is used to reconfigure a running ConnHandler. A Game must
	// pass something in before this ConnHandler will accept connections.
	UpdateGameInfo chan *GameInfo

	listener net.Listener
	gameInfo *GameInfo
}

// NewConnHandler creates and starts a ConnHandler.
func NewConnHandler(listener net.Listener, gameInfo *GameInfo) *ConnHandler {
	ch := &ConnHandler{
		UpdateGameInfo: make(chan *GameInfo),
		listener:       listener,
		gameInfo:       gameInfo,
	}

	go ch.run()

	return ch
}

// Stop stops the connection handler from accepting any further connections.
func (ch *ConnHandler) Stop() {
	close(ch.UpdateGameInfo)
	ch.listener.Close()
}

func (ch *ConnHandler) run() {
	defer ch.listener.Close()
	var ok bool

	for {
		conn, err := ch.listener.Accept()
		if err != nil {
			log.Print("Accept: ", err)
			return
		}

		// Check for updated game info.
		select {
		case ch.gameInfo, ok = <-ch.UpdateGameInfo:
			if !ok {
				log.Print("Connection handler shut down.")
				return
			}
		default:
		}

		newLogin := &pktHandler{
			gameInfo: ch.gameInfo,
			conn:     conn,
		}
		go newLogin.handle()
	}
}

type pktHandler struct {
	gameInfo *GameInfo
	conn     net.Conn
	ps       proto.PacketSerializer
}

func (l *pktHandler) handle() {
	var err, clientErr error

	defer func() {
		if err != nil {
			log.Print("Connection closed ", err.Error())
			if clientErr == nil {
				clientErr = clientErrGeneral
			}
			l.ps.WritePacket(l.conn, &proto.PacketDisconnect{
				Reason: clientErr.Error(),
			})
			l.conn.Close()
		}
	}()

	pkt, err := l.ps.ReadPacketExpect(l.conn, true, 0x02, 0xfe)
	if err != nil {
		clientErr = clientErrLoginGeneral
		return
	}

	switch p := pkt.(type) {
	case *proto.PacketHandshake:
		err, clientErr = l.handleLogin(p)
	case *proto.PacketServerListPing:
		err, clientErr = l.handleServerQuery()
	default:
		err = loginErrorConnType
	}
}

func (l *pktHandler) handleLogin(pktHandshake *proto.PacketHandshake) (err, clientErr error) {
	username := pktHandshake.UsernameOrHash
	if !validPlayerUsername.MatchString(username) {
		err = clientErrUsername
		clientErr = err
		return
	}

	log.Print("Client ", l.conn.RemoteAddr(), " connected as ", username)

	// TODO Allow admins to connect.
	if l.gameInfo.maintenanceMsg != "" {
		err = loginErrorMaintenance
		clientErr = errors.New(l.gameInfo.maintenanceMsg)
		return
	}

	// Load player permissions.
	permissions := gamerules.Permissions.UserPermissions(username)
	if !permissions.Has("login") {
		err = fmt.Errorf("Player %q does not have login permission", username)
		clientErr = clientErrLoginDenied
		return
	}

	sessionId := fmt.Sprintf("%016x", rand.Int63())
	log.Printf("Player %q has sessionId %s", username, sessionId)

	if err = l.ps.WritePacket(l.conn, &proto.PacketHandshake{sessionId}); err != nil {
		clientErr = clientErrHandshake
		return
	}

	if _, err = l.ps.ReadPacketExpect(l.conn, true, 0x01); err != nil {
		clientErr = clientErrLoginGeneral
		return
	}

	authenticated, err := l.gameInfo.authserver.Authenticate(sessionId, username)
	if !authenticated || err != nil {
		var reason string
		if err != nil {
			reason = "Authentication check failed: " + err.Error()
		} else {
			reason = "Failed authentication"
		}
		err = fmt.Errorf("Client %v: %s", l.conn.RemoteAddr(), reason)
		clientErr = clientErrAuthFailed
		return
	}
	log.Print("Client ", l.conn.RemoteAddr(), " passed minecraft.net authentication")

	entityId := l.gameInfo.entityManager.NewEntity()

	var playerData nbt.Compound
	if playerData, err = l.gameInfo.game.worldStore.PlayerData(username); err != nil {
		clientErr = clientErrUserData
		return
	}

	player := player.NewPlayer(entityId, l.gameInfo.shardManager, l.conn, username, l.gameInfo.worldStore.SpawnPosition, l.gameInfo.game.playerDisconnect, l.gameInfo.game)
	if playerData != nil {
		if err = player.UnmarshalNbt(playerData); err != nil {
			// Don't let the player log in, as they will only have default inventory
			// etc., which could lose items from them. Better for an administrator to
			// sort this out.
			err = fmt.Errorf("Error parsing player data for %q: %v", username, err)
			clientErr = clientErrUserData
			return
		}
	}

	l.gameInfo.game.playerConnect <- player
	player.Run()

	return
}

func (l *pktHandler) handleServerQuery() (err, clientErr error) {
	err = loginErrorServerList
	clientErr = fmt.Errorf(
		"%s§%d§%d",
		l.gameInfo.serverDesc,
		l.gameInfo.game.PlayerCount(), l.gameInfo.maxPlayerCount)
	return
}
