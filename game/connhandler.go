package chunkymonkey

import (
	"errors"
	"fmt"
	"log"
	"net"

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

// TODO Refactor this more simply after a good re-working of the chunkymonkey/proto package.

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
	serverId       string
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

	connType int
	username string
}

func (l *pktHandler) handle() {
	var err, clientErr error

	defer func() {
		if err != nil {
			log.Print("Connection closed ", err.Error())
			if clientErr == nil {
				clientErr = clientErrGeneral
			}
			proto.WriteDisconnect(l.conn, clientErr.Error())
			l.conn.Close()
		}
	}()

	err = proto.ServerReadPacketExpect(l.conn, l, []byte{
		proto.PacketIdHandshake,
		proto.PacketIdServerListPing,
	})
	if err != nil {
		clientErr = clientErrLoginGeneral
		return
	}

	switch l.connType {
	case connTypeLogin:
		err, clientErr = l.handleLogin(l.conn)
	case connTypeServerQuery:
		err, clientErr = l.handleServerQuery(l.conn)
	default:
		err = loginErrorConnType
	}
}

func (l *pktHandler) handleLogin(conn net.Conn) (err, clientErr error) {
	if !validPlayerUsername.MatchString(l.username) {
		err = clientErrUsername
		clientErr = err
		return
	}

	log.Print("Client ", conn.RemoteAddr(), " connected as ", l.username)

	// TODO Allow admins to connect.
	if l.gameInfo.maintenanceMsg != "" {
		err = loginErrorMaintenance
		clientErr = errors.New(l.gameInfo.maintenanceMsg)
		return
	}

	// Load player permissions.
	permissions := gamerules.Permissions.UserPermissions(l.username)
	if !permissions.Has("login") {
		err = fmt.Errorf("Player %q does not have login permission", l.username)
		clientErr = clientErrLoginDenied
		return
	}

	if err = proto.ServerWriteHandshake(conn, l.gameInfo.serverId); err != nil {
		clientErr = clientErrHandshake
		return
	}

	if l.gameInfo.serverId != "-" {
		var authenticated bool
		authenticated, err = l.gameInfo.authserver.Authenticate(l.gameInfo.serverId, l.username)
		if !authenticated || err != nil {
			var reason string
			if err != nil {
				reason = "Authentication check failed: " + err.Error()
			} else {
				reason = "Failed authentication"
			}
			err = fmt.Errorf("Client %v: %s", conn.RemoteAddr(), reason)
			clientErr = clientErrAuthFailed
			return
		}
		log.Print("Client ", conn.RemoteAddr(), " passed minecraft.net authentication")
	}

	err = proto.ServerReadPacketExpect(conn, l, []byte{
		proto.PacketIdLogin,
	})
	if err != nil {
		clientErr = clientErrLoginGeneral
		return
	}

	entityId := l.gameInfo.entityManager.NewEntity()

	var playerData *nbt.Compound
	if playerData, err = l.gameInfo.game.worldStore.PlayerData(l.username); err != nil {
		clientErr = clientErrUserData
		return
	}

	player := player.NewPlayer(entityId, l.gameInfo.shardManager, conn, l.username, l.gameInfo.worldStore.SpawnPosition, l.gameInfo.game.playerDisconnect, l.gameInfo.game)
	if playerData != nil {
		if err = player.UnmarshalNbt(playerData); err != nil {
			// Don't let the player log in, as they will only have default inventory
			// etc., which could lose items from them. Better for an administrator to
			// sort this out.
			err = fmt.Errorf("Error parsing player data for %q: %v", l.username, err)
			clientErr = clientErrUserData
			return
		}
	}

	l.gameInfo.game.playerConnect <- player
	player.Run()

	return
}

func (l *pktHandler) handleServerQuery(conn net.Conn) (err, clientErr error) {
	err = loginErrorServerList
	clientErr = fmt.Errorf(
		"%s§%d§%d",
		l.gameInfo.serverDesc,
		l.gameInfo.game.PlayerCount(), l.gameInfo.maxPlayerCount)
	return
}

func (l *pktHandler) PacketServerLogin(username string) {
}

func (l *pktHandler) PacketServerHandshake(username string) {
	l.connType = connTypeLogin
	l.username = username
}

func (l *pktHandler) PacketServerListPing() {
	l.connType = connTypeServerQuery
}

func (l *pktHandler) PacketKeepAlive(id int32) {}

func (l *pktHandler) PacketChatMessage(message string) {}

func (l *pktHandler) PacketPlayer(onGround bool) {}

func (l *pktHandler) PacketHoldingChange(slotId SlotId) {}

func (l *pktHandler) PacketEntityAction(entityId EntityId, action EntityAction) {}

func (l *pktHandler) PacketUseEntity(user EntityId, target EntityId, leftClick bool) {}

func (l *pktHandler) PacketRespawn(dimension DimensionId, unknown int8, gameType GameType, worldHeight int16, mapSeed RandomSeed) {
}

func (l *pktHandler) PacketPlayerPosition(position *AbsXyz, stance AbsCoord, onGround bool) {}

func (l *pktHandler) PacketPlayerLook(look *LookDegrees, onGround bool) {}

func (l *pktHandler) PacketPlayerBlockHit(status DigStatus, blockLoc *BlockXyz, face Face) {}

func (l *pktHandler) PacketPlayerBlockInteract(itemTypeId ItemTypeId, blockLoc *BlockXyz, face Face, amount ItemCount, data ItemData) {
}

func (l *pktHandler) PacketEntityAnimation(entityId EntityId, animation EntityAnimation) {}

func (l *pktHandler) PacketUnknown0x1b(field1, field2 float32, field3, field4 bool, field5, field6 float32) {
}

func (l *pktHandler) PacketUnknown0x3d(field1, field2 int32, field3 int8, field4, field5 int32) {}

func (l *pktHandler) PacketWindowClose(windowId WindowId) {}

func (l *pktHandler) PacketWindowClick(windowId WindowId, slot SlotId, rightClick bool, txId TxId, shiftClick bool, expectedSlot *proto.WindowSlot) {
}

func (l *pktHandler) PacketWindowTransaction(windowId WindowId, txId TxId, accepted bool) {}

func (l *pktHandler) PacketSignUpdate(position *BlockXyz, lines [4]string) {}

func (l *pktHandler) PacketDisconnect(reason string) {}
