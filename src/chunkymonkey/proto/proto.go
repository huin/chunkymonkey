package proto

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"utf8"
	"regexp"

	. "chunkymonkey/types"
)

const (
	// Currently only this protocol version is supported.
	protocolVersion = 18

	maxUcs2Char  = 0xffff
	ucs2ReplChar = 0xfffd

	// Packet type IDs
	packetIdKeepAlive            = 0x00
	packetIdLogin                = 0x01
	packetIdHandshake            = 0x02
	packetIdChatMessage          = 0x03
	packetIdTimeUpdate           = 0x04
	packetIdEntityEquipment      = 0x05
	packetIdSpawnPosition        = 0x06
	packetIdUseEntity            = 0x07
	packetIdUpdateHealth         = 0x08
	packetIdRespawn              = 0x09
	packetIdPlayer               = 0x0a
	packetIdPlayerPosition       = 0x0b
	packetIdPlayerLook           = 0x0c
	packetIdPlayerPositionLook   = 0x0d
	packetIdPlayerBlockHit       = 0x0e
	packetIdPlayerBlockInteract  = 0x0f
	packetIdHoldingChange        = 0x10
	packetIdBedUse               = 0x11
	packetIdEntityAnimation      = 0x12
	packetIdEntityAction         = 0x13
	packetIdNamedEntitySpawn     = 0x14
	packetIdItemSpawn            = 0x15
	packetIdItemCollect          = 0x16
	packetIdObjectSpawn          = 0x17
	packetIdEntitySpawn          = 0x18
	packetIdPaintingSpawn        = 0x19
	packetIdExperienceOrb        = 0x1a
	packetIdEntityVelocity       = 0x1c
	packetIdEntityDestroy        = 0x1d
	packetIdEntity               = 0x1e
	packetIdEntityRelMove        = 0x1f
	packetIdEntityLook           = 0x20
	packetIdEntityLookAndRelMove = 0x21
	packetIdEntityTeleport       = 0x22
	packetIdEntityStatus         = 0x26
	packetIdEntityMetadata       = 0x28
	packetIdEntityEffect         = 0x29
	packetIdEntityRemoveEffect   = 0x2a
	packetIdPlayerExperience     = 0x2b
	packetIdPreChunk             = 0x32
	packetIdMapChunk             = 0x33
	packetIdBlockChangeMulti     = 0x34
	packetIdBlockChange          = 0x35
	packetIdNoteBlockPlay        = 0x36
	packetIdExplosion            = 0x3c
	packetIdSoundEffect          = 0x3d
	packetIdState                = 0x46
	packetIdWeather              = 0x47
	packetIdWindowOpen           = 0x64
	packetIdWindowClose          = 0x65
	packetIdWindowClick          = 0x66
	packetIdWindowSetSlot        = 0x67
	packetIdWindowItems          = 0x68
	packetIdWindowProgressBar    = 0x69
	packetIdWindowTransaction    = 0x6a
	packetIdQuickbarSlotUpdate   = 0x6b
	packetIdSignUpdate           = 0x82
	packetIdItemData             = 0x83
	packetIdIncrementStatistic   = 0xc8
	packetIdUserListItem         = 0xc9
	packetIdServerListPing       = 0xfe
	packetIdDisconnect           = 0xff
)

// Regexp for ChatMessages
var checkChatMessageRegexp = regexp.MustCompile("[ !\"#$%&'()*+,-./0123456789:;<=>?@ABCDEFGHIJKLMNOPQRSTUVWXYZ[\\]^_abcdefghijklmnopqrstuvwxyz{|}~⌂ÇüéâäàåçêëèïîìÄÅÉæÆôöòûùÿÖÜø£Ø×ƒáíóúñÑªº¿®¬½¼¡«»]*")
var checkColorsRegexp = regexp.MustCompile("§.$")

// Errors
var illegalCharErr = os.NewError("Found one or more illegal characters. This could crash clients.")
var colorTagEndErr = os.NewError("Found a color tag at the end of a message. This could crash clients.")

// Packets commonly received by both client and server
type IPacketHandler interface {
	PacketKeepAlive(id int32)
	PacketChatMessage(message string)
	PacketEntityAction(entityId EntityId, action EntityAction)
	PacketUseEntity(user EntityId, target EntityId, leftClick bool)
	PacketRespawn(dimension DimensionId, unknown int8, gameType GameType, worldHeight int16, mapSeed RandomSeed)
	PacketPlayerPosition(position *AbsXyz, stance AbsCoord, onGround bool)
	PacketPlayerLook(look *LookDegrees, onGround bool)
	PacketPlayerBlockHit(status DigStatus, blockLoc *BlockXyz, face Face)
	PacketPlayerBlockInteract(itemTypeId ItemTypeId, blockLoc *BlockXyz, face Face, amount ItemCount, data ItemData)
	PacketEntityAnimation(entityId EntityId, animation EntityAnimation)
	PacketWindowTransaction(windowId WindowId, txId TxId, accepted bool)
	PacketSignUpdate(position *BlockXyz, lines [4]string)
	PacketDisconnect(reason string)
}

// Servers to the protocol must implement this interface to receive packets
type IServerPacketHandler interface {
	IPacketHandler
	PacketPlayer(onGround bool)
	PacketHoldingChange(slotId SlotId)
	PacketWindowClose(windowId WindowId)
	PacketWindowClick(windowId WindowId, slot SlotId, rightClick bool, txId TxId, shiftClick bool, expectedSlot *WindowSlot)
	PacketServerListPing()
}

// Clients to the protocol must implement this interface to receive packets
type IClientPacketHandler interface {
	IPacketHandler
	ClientPacketLogin(entityId EntityId, mapSeed RandomSeed, serverMode int32, dimension DimensionId, unknown int8, worldHeight, maxPlayers byte)
	PacketTimeUpdate(time Ticks)
	PacketBedUse(flag bool, bedLoc *BlockXyz)
	PacketNamedEntitySpawn(entityId EntityId, name string, position *AbsIntXyz, look *LookBytes, currentItem ItemTypeId)
	PacketEntityEquipment(entityId EntityId, slot SlotId, itemTypeId ItemTypeId, data ItemData)
	PacketSpawnPosition(position *BlockXyz)
	PacketUpdateHealth(health Health, food FoodUnits, foodSaturation float32)
	PacketItemSpawn(entityId EntityId, itemTypeId ItemTypeId, count ItemCount, data ItemData, location *AbsIntXyz, orientation *OrientationBytes)
	PacketItemCollect(collectedItem EntityId, collector EntityId)
	PacketObjectSpawn(entityId EntityId, objType ObjTypeId, position *AbsIntXyz, objectData *ObjectData)
	PacketEntitySpawn(entityId EntityId, mobType EntityMobType, position *AbsIntXyz, look *LookBytes, data []EntityMetadata)
	PacketPaintingSpawn(entityId EntityId, title string, position *BlockXyz, paintingType PaintingTypeId)
	PacketExperienceOrb(entityId EntityId, position AbsIntXyz, count int16)
	PacketEntityVelocity(entityId EntityId, velocity *Velocity)
	PacketEntityDestroy(entityId EntityId)
	PacketEntity(entityId EntityId)
	PacketEntityRelMove(entityId EntityId, movement *RelMove)
	PacketEntityLook(entityId EntityId, look *LookBytes)
	PacketEntityTeleport(entityId EntityId, position *AbsIntXyz, look *LookBytes)
	PacketEntityStatus(entityId EntityId, status EntityStatus)
	PacketEntityMetadata(entityId EntityId, metadata []EntityMetadata)
	PacketEntityEffect(entityId EntityId, effect EntityEffect, value int8, duration int16)
	PacketEntityRemoveEffect(entityId EntityId, effect EntityEffect)
	PacketPlayerExperience(experience, level int8, totalExperience int16)

	PacketPreChunk(position *ChunkXz, mode ChunkLoadMode)
	PacketMapChunk(position *BlockXyz, size *SubChunkSize, data []byte)
	PacketBlockChangeMulti(chunkLoc *ChunkXz, blockCoords []SubChunkXyz, blockTypes []BlockId, blockMetaData []byte)
	PacketBlockChange(blockLoc *BlockXyz, blockType BlockId, blockMetaData byte)
	PacketNoteBlockPlay(position *BlockXyz, instrument InstrumentId, pitch NotePitch)

	// NOTE method signature likely to change
	PacketExplosion(position *AbsXyz, power float32, blockOffsets []ExplosionOffsetXyz)
	PacketSoundEffect(sound SoundEffect, position BlockXyz, data int32)

	PacketState(reason, gameMode byte)
	PacketWeather(entityId EntityId, raining bool, position *AbsIntXyz)

	PacketWindowOpen(windowId WindowId, invTypeId InvTypeId, windowTitle string, numSlots byte)
	PacketWindowSetSlot(windowId WindowId, slot SlotId, itemTypeId ItemTypeId, amount ItemCount, data ItemData)
	PacketWindowItems(windowId WindowId, items []WindowSlot)
	PacketWindowProgressBar(windowId WindowId, prgBarId PrgBarId, value PrgBarValue)
	PacketQuickbarSlotUpdate(slot SlotId, itemId ItemTypeId, count ItemCount, data ItemData)
	PacketItemData(itemTypeId ItemTypeId, itemDataId ItemData, data []byte)
	PacketIncrementStatistic(statisticId StatisticId, delta int8)
	PacketUserListItem(username string, unknown bool, ping int16)
}

// Common protocol helper functions

func boolToByte(b bool) byte {
	if b {
		return 1
	}
	return 0
}

func byteToBool(b byte) bool {
	return b != 0
}

// Conversion between UTF-8 and UCS-2.

func encodeUtf8(codepoints []uint16) string {
	bytesRequired := 0

	for _, cp := range codepoints {
		bytesRequired += utf8.RuneLen(int(cp))
	}

	bs := make([]byte, bytesRequired)
	curByte := 0
	for _, cp := range codepoints {
		curByte += utf8.EncodeRune(bs[curByte:], int(cp))
	}

	return string(bs)
}

func decodeUtf8(s string) []uint16 {
	codepoints := make([]uint16, 0, len(s))

	for _, cp := range s {
		// We only encode chars in the range U+0000 to U+FFFF.
		if cp > maxUcs2Char || cp < 0 {
			cp = ucs2ReplChar
		}
		codepoints = append(codepoints, uint16(cp))
	}

	return codepoints
}

// 16-bit encoded strings. (UCS-2)

func readString16(reader io.Reader) (s string, err os.Error) {
	var length uint16
	err = binary.Read(reader, binary.BigEndian, &length)
	if err != nil {
		return
	}

	bs := make([]uint16, length)
	err = binary.Read(reader, binary.BigEndian, bs)
	if err != nil {
		return
	}

	return encodeUtf8(bs), err
}

func writeString16(writer io.Writer, s string) (err os.Error) {
	bs := decodeUtf8(s)

	err = binary.Write(writer, binary.BigEndian, int16(len(bs)))
	if err != nil {
		return
	}

	err = binary.Write(writer, binary.BigEndian, bs)
	return
}

type WindowSlot struct {
	ItemTypeId ItemTypeId
	Count      ItemCount
	Data       ItemData
}

type EntityMetadata struct {
	Field1 byte
	Field2 byte
	Field3 interface{}
}

func writeEntityMetadataField(writer io.Writer, data []EntityMetadata) (err os.Error) {
	// NOTE that no checking is done upon the form of the data, so it's
	// possible to form bad data packets with this.
	var entryType byte

	for _, item := range data {
		entryType = (item.Field1 << 5) & 0xe0
		entryType |= (item.Field2 & 0x1f)

		if err = binary.Write(writer, binary.BigEndian, entryType); err != nil {
			return
		}
		switch item.Field1 {
		case 0:
			err = binary.Write(writer, binary.BigEndian, item.Field3.(byte))
		case 1:
			err = binary.Write(writer, binary.BigEndian, item.Field3.(int16))
		case 2:
			err = binary.Write(writer, binary.BigEndian, item.Field3.(int32))
		case 3:
			err = binary.Write(writer, binary.BigEndian, item.Field3.(float32))
		case 4:
			err = writeString16(writer, item.Field3.(string))
		case 5:
			type position struct {
				X int16
				Y byte
				Z int16
			}
			err = binary.Write(writer, binary.BigEndian, item.Field3.(position))
		}
		if err != nil {
			return
		}
	}

	// Mark end of metadata
	return binary.Write(writer, binary.BigEndian, byte(127))
}

// Reads entity metadata from the end of certain packets. Most of the meaning
// of the packets isn't yet known.
// TODO update to pull useful data out as it becomes understood
func readEntityMetadataField(reader io.Reader) (data []EntityMetadata, err os.Error) {
	var entryType byte

	var field1, field2 byte
	var field3 interface{}

	for {
		err = binary.Read(reader, binary.BigEndian, &entryType)
		if err != nil {
			return
		}
		if entryType == 127 {
			break
		}
		field2 = entryType & 0x1f

		switch field1 := (entryType & 0xe0) >> 5; field1 {
		case 0:
			var byteVal byte
			err = binary.Read(reader, binary.BigEndian, &byteVal)
			field3 = byteVal
		case 1:
			var int16Val int16
			err = binary.Read(reader, binary.BigEndian, &int16Val)
			field3 = int16Val
		case 2:
			var int32Val int32
			err = binary.Read(reader, binary.BigEndian, &int32Val)
			field3 = int32Val
		case 3:
			var floatVal float32
			err = binary.Read(reader, binary.BigEndian, &floatVal)
			field3 = floatVal
		case 4:
			var stringVal string
			stringVal, err = readString16(reader)
			field3 = stringVal
		case 5:
			var position struct {
				X int16
				Y byte
				Z int16
			}
			err = binary.Read(reader, binary.BigEndian, &position)
			field3 = position
		}

		data = append(data, EntityMetadata{field1, field2, field3})

		if err != nil {
			return
		}
	}
	return
}

type ObjectData struct {
	Field1 int32
	Field2 [3]uint16
}

// Start of packet reader/writer functions

// Naming convention:
// * Client* functions are specific to use by clients writing to a server, and
//   reading from it.
// * Server* functions are specific to use by servers writing to clients, and
//   reading from them.
// * Those without a client or server prefix don't differ in content or meaning
//   between client and server.

// packetIdKeepAlive

func WriteKeepAlive(writer io.Writer, id int32) os.Error {
	var packet = struct {
		PacketId byte
		Id       int32
	}{
		packetIdKeepAlive,
		id,
	}
	return binary.Write(writer, binary.BigEndian, &packet)
}

func readKeepAlive(reader io.Reader, handler IPacketHandler) (err os.Error) {
	var id int32
	if err = binary.Read(reader, binary.BigEndian, &id); err != nil {
		return
	}
	handler.PacketKeepAlive(id)
	return
}

// packetIdLogin

func commonReadLogin(reader io.Reader) (versionOrEntityId int32, str string, mapSeed RandomSeed, serverMode int32, dimension DimensionId, unknown int8, worldHeight, maxPlayers byte, err os.Error) {
	if err = binary.Read(reader, binary.BigEndian, &versionOrEntityId); err != nil {
		return
	}
	if str, err = readString16(reader); err != nil {
		return
	}

	var packetEnd struct {
		MapSeed     RandomSeed
		ServerMode  int32
		Dimension   DimensionId
		Unknown     int8
		WorldHeight byte
		MaxPlayers  byte
	}
	if err = binary.Read(reader, binary.BigEndian, &packetEnd); err != nil {
		return
	}

	mapSeed = packetEnd.MapSeed
	serverMode = packetEnd.ServerMode
	dimension = packetEnd.Dimension
	unknown = packetEnd.Unknown
	worldHeight = packetEnd.WorldHeight
	maxPlayers = packetEnd.MaxPlayers

	return
}

func ServerReadLogin(reader io.Reader) (username string, err os.Error) {
	var packetId byte
	if err = binary.Read(reader, binary.BigEndian, &packetId); err != nil {
		return
	}
	if packetId != packetIdLogin {
		err = os.NewError(fmt.Sprintf("serverLogin: invalid packet Id %#x", packetId))
		return
	}

	version, username, _, _, _, _, _, _, err := commonReadLogin(reader)
	if err != nil {
		return
	}

	if version != protocolVersion {
		err = os.NewError(fmt.Sprintf("serverLogin: unsupported protocol version %#x", version))
		return
	}
	return
}

func clientReadLogin(reader io.Reader, handler IClientPacketHandler) (err os.Error) {
	entityId, _, mapSeed, serverMode, dimension, unknown, worldHeight, maxPlayers, err := commonReadLogin(reader)
	if err != nil {
		return
	}

	handler.ClientPacketLogin(EntityId(entityId), mapSeed, serverMode, dimension, unknown, worldHeight, maxPlayers)

	return
}

func commonWriteLogin(writer io.Writer, versionOrEntityId int32, str string, mapSeed RandomSeed, serverMode int32, dimension DimensionId, unknown int8, worldHeight, maxPlayers byte) (err os.Error) {
	if err = binary.Write(writer, binary.BigEndian, versionOrEntityId); err != nil {
		return
	}

	if err = writeString16(writer, str); err != nil {
		return
	}

	var packetEnd = struct {
		MapSeed     RandomSeed
		ServerMode  int32
		Dimension   DimensionId
		Unknown     int8
		WorldHeight byte
		MaxPlayers  byte
	}{
		mapSeed,
		serverMode,
		dimension,
		unknown,
		worldHeight,
		maxPlayers,
	}
	return binary.Write(writer, binary.BigEndian, &packetEnd)
}

func ServerWriteLogin(writer io.Writer, entityId EntityId, mapSeed RandomSeed, serverMode int32, dimension DimensionId, unknown int8, worldHeight, maxPlayers byte) (err os.Error) {
	if err = binary.Write(writer, binary.BigEndian, byte(packetIdLogin)); err != nil {
		return
	}

	return commonWriteLogin(writer, int32(entityId), "", mapSeed, serverMode, dimension, unknown, worldHeight, maxPlayers)
}

func ClientWriteLogin(writer io.Writer, username, password string) (err os.Error) {
	if err = binary.Write(writer, binary.BigEndian, byte(packetIdLogin)); err != nil {
		return
	}

	return commonWriteLogin(writer, protocolVersion, username, 0, 0, 0, 0, 0, 0)
}

// packetIdHandshake

func ServerReadHandshake(reader io.Reader) (username string, err os.Error) {
	var packetId byte
	err = binary.Read(reader, binary.BigEndian, &packetId)
	if err != nil {
		return
	}
	if packetId != packetIdHandshake {
		err = os.NewError(fmt.Sprintf("serverHandshake: invalid packet Id %#x", packetId))
		return
	}

	return readString16(reader)
}

func ClientReadHandshake(reader io.Reader) (serverId string, err os.Error) {
	var packetId byte
	err = binary.Read(reader, binary.BigEndian, &packetId)
	if err != nil {
		return
	}
	if packetId != packetIdHandshake {
		err = os.NewError(fmt.Sprintf("readHandshake: invalid packet Id %#x", packetId))
		return
	}

	return readString16(reader)
}

func ServerWriteHandshake(writer io.Writer, reply string) (err os.Error) {
	err = binary.Write(writer, binary.BigEndian, byte(packetIdHandshake))
	if err != nil {
		return
	}

	return writeString16(writer, reply)
}

// packetIdChatMessage

func WriteChatMessage(writer io.Writer, message string) (err os.Error) {
	// Check chat message against illegal chars
	if checkChatMessageRegexp.MatchString(message) {
		// Check suffix against color tags eg. "This is a message §0"
		if checkColorsRegexp.MatchString(message) {
			// Found a color tag at the end
			return colorTagEndErr
		} else {
			err = binary.Write(writer, binary.BigEndian, byte(packetIdChatMessage))
			if err != nil {
				return
			}
			err = writeString16(writer, message)
			return
		}
	}
	return illegalCharErr
}

func readChatMessage(reader io.Reader, handler IPacketHandler) (err os.Error) {
	message, err := readString16(reader)
	if err != nil {
		return
	}
	if checkChatMessageRegexp.MatchString(message) {
		// Does not contain illegal chars
		if checkColorsRegexp.MatchString(message) {
			// Contains a color tag at the end
			return colorTagEndErr
		}
		// message is fine
		handler.PacketChatMessage(message)
		return
	}
	return illegalCharErr
}

// packetIdTimeUpdate

func ServerWriteTimeUpdate(writer io.Writer, time Ticks) os.Error {
	var packet = struct {
		PacketId byte
		Time     Ticks
	}{
		packetIdTimeUpdate,
		time,
	}
	return binary.Write(writer, binary.BigEndian, &packet)
}

func readTimeUpdate(reader io.Reader, handler IClientPacketHandler) (err os.Error) {
	var time Ticks

	err = binary.Read(reader, binary.BigEndian, &time)
	if err != nil {
		return
	}

	handler.PacketTimeUpdate(time)
	return
}

// packetIdEntityEquipment

func WriteEntityEquipment(writer io.Writer, entityId EntityId, slot SlotId, itemTypeId ItemTypeId, data ItemData) (err os.Error) {
	var packet = struct {
		PacketId   byte
		EntityId   EntityId
		Slot       SlotId
		ItemTypeId ItemTypeId
		Data       ItemData
	}{
		packetIdEntityEquipment,
		entityId,
		slot,
		itemTypeId,
		data,
	}

	return binary.Write(writer, binary.BigEndian, &packet)
}

func readEntityEquipment(reader io.Reader, handler IClientPacketHandler) (err os.Error) {
	var packet struct {
		EntityId   EntityId
		Slot       SlotId
		ItemTypeId ItemTypeId
		Data       ItemData
	}

	err = binary.Read(reader, binary.BigEndian, &packet)
	if err != nil {
		return
	}

	handler.PacketEntityEquipment(
		packet.EntityId, packet.Slot, packet.ItemTypeId, packet.Data)

	return
}

// packetIdSpawnPosition

func WriteSpawnPosition(writer io.Writer, position *BlockXyz) os.Error {
	var packet = struct {
		PacketId byte
		X        BlockCoord
		Y        int32
		Z        BlockCoord
	}{
		packetIdSpawnPosition,
		position.X,
		int32(position.Y),
		position.Z,
	}
	return binary.Write(writer, binary.BigEndian, &packet)
}

func readSpawnPosition(reader io.Reader, handler IClientPacketHandler) (err os.Error) {
	var packet struct {
		X BlockCoord
		Y int32
		Z BlockCoord
	}

	err = binary.Read(reader, binary.BigEndian, &packet)
	if err != nil {
		return
	}

	handler.PacketSpawnPosition(&BlockXyz{
		packet.X,
		BlockYCoord(packet.Y),
		packet.Z,
	})
	return
}

// packetIdUseEntity

func WriteUseEntity(writer io.Writer, user EntityId, target EntityId, leftClick bool) (err os.Error) {
	var packet = struct {
		PacketId  byte
		User      EntityId
		Target    EntityId
		LeftClick byte
	}{
		packetIdUseEntity,
		user,
		target,
		boolToByte(leftClick),
	}

	return binary.Write(writer, binary.BigEndian, &packet)
}

func readUseEntity(reader io.Reader, handler IPacketHandler) (err os.Error) {
	var packet struct {
		User      EntityId
		Target    EntityId
		LeftClick byte
	}

	err = binary.Read(reader, binary.BigEndian, &packet)
	if err != nil {
		return
	}

	handler.PacketUseEntity(packet.User, packet.Target, byteToBool(packet.LeftClick))

	return
}

// packetIdUpdateHealth

func WriteUpdateHealth(writer io.Writer, health Health, food FoodUnits, foodSaturation float32) (err os.Error) {
	var packet = struct {
		PacketId       byte
		Health         Health
		Food           FoodUnits
		FoodSaturation float32
	}{
		packetIdUpdateHealth,
		health,
		food,
		foodSaturation,
	}

	return binary.Write(writer, binary.BigEndian, &packet)
}

func readUpdateHealth(reader io.Reader, handler IClientPacketHandler) (err os.Error) {
	var packet struct {
		Health         Health
		Food           FoodUnits
		FoodSaturation float32
	}

	err = binary.Read(reader, binary.BigEndian, &packet)
	if err != nil {
		return
	}

	handler.PacketUpdateHealth(packet.Health, packet.Food, packet.FoodSaturation)
	return
}

// packetIdRespawn

func WriteRespawn(writer io.Writer, dimension DimensionId, unknown int8, gameType GameType, worldHeight int16, mapSeed RandomSeed) os.Error {
	var packet = struct {
		PacketId    byte
		Dimension   DimensionId
		Unknown     int8
		GameType    GameType
		WorldHeight int16
		MapSeed     RandomSeed
	}{
		packetIdRespawn,
		dimension,
		unknown,
		gameType,
		worldHeight,
		mapSeed,
	}

	return binary.Write(writer, binary.BigEndian, &packet)
}

func readRespawn(reader io.Reader, handler IPacketHandler) (err os.Error) {
	var packet struct {
		Dimension   DimensionId
		Unknown     int8
		GameType    GameType
		WorldHeight int16
		MapSeed     RandomSeed
	}

	err = binary.Read(reader, binary.BigEndian, &packet)
	if err != nil {
		return
	}

	handler.PacketRespawn(packet.Dimension, packet.Unknown, packet.GameType, packet.WorldHeight, packet.MapSeed)

	return
}

// packetIdPlayer

func WritePlayer(writer io.Writer, onGround bool) (err os.Error) {
	var packet = struct {
		PacketId byte
		OnGround byte
	}{
		packetIdPlayer,
		boolToByte(onGround),
	}

	return binary.Write(writer, binary.BigEndian, &packet)
}

func readPlayer(reader io.Reader, handler IServerPacketHandler) (err os.Error) {
	var onGround byte

	if err = binary.Read(reader, binary.BigEndian, &onGround); err != nil {
		return
	}

	handler.PacketPlayer(byteToBool(onGround))

	return
}

// packetIdPlayerPosition

func WritePlayerPosition(writer io.Writer, position *AbsXyz, stance AbsCoord, onGround bool) os.Error {
	var packet = struct {
		PacketId byte
		X        AbsCoord
		Y        AbsCoord
		Stance   AbsCoord
		Z        AbsCoord
		OnGround byte
	}{
		packetIdPlayerPosition,
		position.X,
		position.Y,
		stance,
		position.Z,
		boolToByte(onGround),
	}
	return binary.Write(writer, binary.BigEndian, &packet)
}

func readPlayerPosition(reader io.Reader, handler IPacketHandler) (err os.Error) {
	var packet struct {
		X        AbsCoord
		Y        AbsCoord
		Stance   AbsCoord
		Z        AbsCoord
		OnGround byte
	}

	if err = binary.Read(reader, binary.BigEndian, &packet); err != nil {
		return
	}

	handler.PacketPlayerPosition(
		&AbsXyz{
			AbsCoord(packet.X),
			AbsCoord(packet.Y),
			AbsCoord(packet.Z),
		},
		packet.Stance,
		byteToBool(packet.OnGround))
	return
}

// packetIdPlayerLook

func WritePlayerLook(writer io.Writer, look *LookDegrees, onGround bool) (err os.Error) {
	var packet = struct {
		PacketId byte
		Yaw      AngleDegrees
		Pitch    AngleDegrees
		OnGround byte
	}{
		packetIdPlayerLook,
		look.Yaw, look.Pitch,
		boolToByte(onGround),
	}

	return binary.Write(writer, binary.BigEndian, &packet)
}

func readPlayerLook(reader io.Reader, handler IPacketHandler) (err os.Error) {
	var packet struct {
		Yaw      AngleDegrees
		Pitch    AngleDegrees
		OnGround byte
	}

	if err = binary.Read(reader, binary.BigEndian, &packet); err != nil {
		return
	}

	handler.PacketPlayerLook(
		&LookDegrees{
			packet.Yaw,
			packet.Pitch,
		},
		byteToBool(packet.OnGround))
	return
}

// packetIdPlayerPositionLook

// packetIdPlayerPositionLook

func writePlayerPositionLookCommon(writer io.Writer, x, y1, y2, z AbsCoord, look *LookDegrees, onGround bool) (err os.Error) {
	var packet = struct {
		PacketId byte
		X        AbsCoord
		Y1       AbsCoord
		Y2       AbsCoord
		Z        AbsCoord
		Yaw      AngleDegrees
		Pitch    AngleDegrees
		OnGround byte
	}{
		packetIdPlayerPositionLook,
		x, y1, y2, z,
		look.Yaw, look.Pitch,
		boolToByte(onGround),
	}

	return binary.Write(writer, binary.BigEndian, &packet)
}

func ClientWritePlayerPositionLook(writer io.Writer, position *AbsXyz, stance AbsCoord, look *LookDegrees, onGround bool) (err os.Error) {
	return writePlayerPositionLookCommon(
		writer,
		position.X, position.Y, stance, position.Z,
		look,
		onGround)
}

func ServerWritePlayerPositionLook(writer io.Writer, position *AbsXyz, stance AbsCoord, look *LookDegrees, onGround bool) (err os.Error) {
	return writePlayerPositionLookCommon(
		writer,
		position.X, stance, position.Y, position.Z,
		look,
		onGround)
}

func clientReadPlayerPositionLook(reader io.Reader, handler IClientPacketHandler) (err os.Error) {
	var packet struct {
		X        AbsCoord
		Stance   AbsCoord
		Y        AbsCoord
		Z        AbsCoord
		Yaw      AngleDegrees
		Pitch    AngleDegrees
		OnGround byte
	}

	err = binary.Read(reader, binary.BigEndian, &packet)
	if err != nil {
		return
	}

	handler.PacketPlayerPosition(
		&AbsXyz{
			packet.X,
			packet.Y,
			packet.Z,
		},
		packet.Stance,
		byteToBool(packet.OnGround))

	handler.PacketPlayerLook(
		&LookDegrees{
			packet.Yaw,
			packet.Pitch,
		},
		byteToBool(packet.OnGround))
	return
}

func serverReadPlayerPositionLook(reader io.Reader, handler IServerPacketHandler) (err os.Error) {
	var packet struct {
		X        AbsCoord
		Y        AbsCoord
		Stance   AbsCoord
		Z        AbsCoord
		Yaw      AngleDegrees
		Pitch    AngleDegrees
		OnGround byte
	}

	err = binary.Read(reader, binary.BigEndian, &packet)
	if err != nil {
		return
	}

	handler.PacketPlayerPosition(
		&AbsXyz{
			packet.X,
			packet.Y,
			packet.Z,
		},
		packet.Stance,
		byteToBool(packet.OnGround))

	handler.PacketPlayerLook(
		&LookDegrees{
			packet.Yaw,
			packet.Pitch,
		},
		byteToBool(packet.OnGround))
	return
}

// packetIdPlayerBlockHit

func WritePlayerBlockHit(writer io.Writer, status DigStatus, blockLoc *BlockXyz, face Face) (err os.Error) {
	var packet = struct {
		PacketId byte
		Status   DigStatus
		X        BlockCoord
		Y        BlockYCoord
		Z        BlockCoord
		Face     Face
	}{
		packetIdPlayerBlockHit,
		status,
		blockLoc.X, blockLoc.Y, blockLoc.Z,
		face,
	}

	return binary.Write(writer, binary.BigEndian, &packet)
}

func readPlayerBlockHit(reader io.Reader, handler IPacketHandler) (err os.Error) {
	var packet struct {
		Status DigStatus
		X      BlockCoord
		Y      BlockYCoord
		Z      BlockCoord
		Face   Face
	}

	err = binary.Read(reader, binary.BigEndian, &packet)
	if err != nil {
		return
	}

	handler.PacketPlayerBlockHit(
		packet.Status,
		&BlockXyz{packet.X, packet.Y, packet.Z},
		packet.Face)
	return
}

// packetIdPlayerBlockInteract

func WritePlayerBlockInteract(writer io.Writer, itemTypeId ItemTypeId, blockLoc *BlockXyz, face Face, amount ItemCount, data ItemData) (err os.Error) {
	var packet = struct {
		PacketId   byte
		X          BlockCoord
		Y          BlockYCoord
		Z          BlockCoord
		Face       Face
		ItemTypeId ItemTypeId
	}{
		packetIdPlayerBlockInteract,
		blockLoc.X, blockLoc.Y, blockLoc.Z,
		face,
		itemTypeId,
	}

	if err = binary.Write(writer, binary.BigEndian, &packet); err != nil {
		return
	}

	if itemTypeId != -1 {
		var packetExtra = struct {
			Amount ItemCount
			Data   ItemData
		}{
			amount,
			data,
		}
		err = binary.Write(writer, binary.BigEndian, &packetExtra)
	}

	return
}

func readPlayerBlockInteract(reader io.Reader, handler IPacketHandler) (err os.Error) {
	var packet struct {
		X          BlockCoord
		Y          BlockYCoord
		Z          BlockCoord
		Face       Face
		ItemTypeId ItemTypeId
	}
	var packetExtra struct {
		Amount ItemCount
		Data   ItemData
	}

	err = binary.Read(reader, binary.BigEndian, &packet)
	if err != nil {
		return
	}

	if packet.ItemTypeId >= 0 {
		err = binary.Read(reader, binary.BigEndian, &packetExtra)
		if err != nil {
			return
		}
	}

	handler.PacketPlayerBlockInteract(
		packet.ItemTypeId,
		&BlockXyz{
			packet.X,
			packet.Y,
			packet.Z,
		},
		packet.Face,
		packetExtra.Amount,
		packetExtra.Data)
	return
}

// packetIdHoldingChange

func WriteHoldingChange(writer io.Writer, slotId SlotId) (err os.Error) {
	var packet = struct {
		PacketId byte
		SlotId   SlotId
	}{
		packetIdHoldingChange,
		slotId,
	}

	return binary.Write(writer, binary.BigEndian, &packet)
}

func readHoldingChange(reader io.Reader, handler IServerPacketHandler) (err os.Error) {
	var slotId SlotId

	if err = binary.Read(reader, binary.BigEndian, &slotId); err != nil {
		return
	}

	handler.PacketHoldingChange(slotId)

	return
}

// packetIdBedUse

func WriteBedUse(writer io.Writer, flag bool, bedLoc *BlockXyz) (err os.Error) {
	var packet = struct {
		PacketId byte
		Flag     byte
		X        BlockCoord
		Y        BlockYCoord
		Z        BlockCoord
	}{
		packetIdBedUse,
		boolToByte(flag),
		bedLoc.X,
		bedLoc.Y,
		bedLoc.Z,
	}

	return binary.Write(writer, binary.BigEndian, &packet)
}

func readBedUse(reader io.Reader, handler IClientPacketHandler) (err os.Error) {
	var packet struct {
		Flag byte
		X    BlockCoord
		Y    BlockYCoord
		Z    BlockCoord
	}

	if err = binary.Read(reader, binary.BigEndian, &packet); err != nil {
		handler.PacketBedUse(
			byteToBool(packet.Flag),
			&BlockXyz{packet.X, packet.Y, packet.Z})
	}

	return
}

// packetIdEntityAnimation

func WriteEntityAnimation(writer io.Writer, entityId EntityId, animation EntityAnimation) (err os.Error) {
	var packet = struct {
		PacketId  byte
		EntityId  EntityId
		Animation EntityAnimation
	}{
		packetIdEntityAnimation,
		entityId,
		animation,
	}

	return binary.Write(writer, binary.BigEndian, &packet)
}

func readEntityAnimation(reader io.Reader, handler IPacketHandler) (err os.Error) {
	var packet struct {
		EntityId  EntityId
		Animation EntityAnimation
	}

	err = binary.Read(reader, binary.BigEndian, &packet)
	if err != nil {
		return
	}

	handler.PacketEntityAnimation(packet.EntityId, packet.Animation)
	return
}

// packetIdEntityAction

func WriteEntityAction(writer io.Writer, entityId EntityId, action EntityAction) (err os.Error) {
	var packet = struct {
		PacketId byte
		EntityId EntityId
		Action   EntityAction
	}{
		packetIdEntityAction,
		entityId,
		action,
	}

	return binary.Write(writer, binary.BigEndian, &packet)
}

func readEntityAction(reader io.Reader, handler IPacketHandler) (err os.Error) {
	var packet struct {
		EntityId EntityId
		Action   EntityAction
	}

	if err = binary.Read(reader, binary.BigEndian, &packet); err != nil {
		return
	}

	handler.PacketEntityAction(packet.EntityId, packet.Action)

	return
}

// packetIdNamedEntitySpawn

func WriteNamedEntitySpawn(writer io.Writer, entityId EntityId, name string, position *AbsIntXyz, look *LookBytes, currentItem ItemTypeId) (err os.Error) {
	var packetStart = struct {
		PacketId byte
		EntityId EntityId
	}{
		packetIdNamedEntitySpawn,
		entityId,
	}

	err = binary.Write(writer, binary.BigEndian, &packetStart)
	if err != nil {
		return
	}

	err = writeString16(writer, name)
	if err != nil {
		return
	}

	var packetFinish = struct {
		X           AbsIntCoord
		Y           AbsIntCoord
		Z           AbsIntCoord
		Yaw         AngleBytes
		Pitch       AngleBytes
		CurrentItem ItemTypeId
	}{
		position.X,
		position.Y,
		position.Z,
		look.Yaw,
		look.Pitch,
		currentItem,
	}

	err = binary.Write(writer, binary.BigEndian, &packetFinish)
	return
}

func readNamedEntitySpawn(reader io.Reader, handler IClientPacketHandler) (err os.Error) {
	var entityId EntityId

	if err = binary.Read(reader, binary.BigEndian, &entityId); err != nil {
		return
	}

	var name string
	if name, err = readString16(reader); err != nil {
		return
	}

	var packetEnd struct {
		X, Y, Z     AbsIntCoord
		Yaw, Pitch  AngleBytes
		CurrentItem ItemTypeId
	}
	if err = binary.Read(reader, binary.BigEndian, &packetEnd); err != nil {
		return
	}

	handler.PacketNamedEntitySpawn(
		entityId,
		name,
		&AbsIntXyz{packetEnd.X, packetEnd.Y, packetEnd.Z},
		&LookBytes{packetEnd.Yaw, packetEnd.Pitch},
		packetEnd.CurrentItem)

	return
}

// packetIdItemSpawn

func WriteItemSpawn(writer io.Writer, entityId EntityId, itemTypeId ItemTypeId, amount ItemCount, data ItemData, position *AbsIntXyz, orientation *OrientationBytes) os.Error {
	var packet = struct {
		PacketId   byte
		EntityId   EntityId
		ItemTypeId ItemTypeId
		Count      ItemCount
		Data       ItemData
		X          AbsIntCoord
		Y          AbsIntCoord
		Z          AbsIntCoord
		Yaw        AngleBytes
		Pitch      AngleBytes
		Roll       AngleBytes
	}{
		packetIdItemSpawn,
		entityId,
		itemTypeId,
		amount,
		data,
		position.X,
		position.Y,
		position.Z,
		orientation.Yaw,
		orientation.Pitch,
		orientation.Roll,
	}

	return binary.Write(writer, binary.BigEndian, &packet)
}

func readItemSpawn(reader io.Reader, handler IClientPacketHandler) (err os.Error) {
	var packet struct {
		EntityId   EntityId
		ItemTypeId ItemTypeId
		Count      ItemCount
		Data       ItemData
		X          AbsIntCoord
		Y          AbsIntCoord
		Z          AbsIntCoord
		Yaw        AngleBytes
		Pitch      AngleBytes
		Roll       AngleBytes
	}

	err = binary.Read(reader, binary.BigEndian, &packet)
	if err != nil {
		return
	}

	handler.PacketItemSpawn(
		packet.EntityId,
		packet.ItemTypeId,
		packet.Count,
		packet.Data,
		&AbsIntXyz{packet.X, packet.Y, packet.Z},
		&OrientationBytes{packet.Yaw, packet.Pitch, packet.Roll})

	return
}

// packetIdItemCollect

func WriteItemCollect(writer io.Writer, collectedItem EntityId, collector EntityId) (err os.Error) {
	var packet = struct {
		PacketId      byte
		CollectedItem EntityId
		Collector     EntityId
	}{
		packetIdItemCollect,
		collectedItem,
		collector,
	}

	return binary.Write(writer, binary.BigEndian, &packet)
}

func readItemCollect(reader io.Reader, handler IClientPacketHandler) (err os.Error) {
	var packet struct {
		CollectedItem EntityId
		Collector     EntityId
	}

	err = binary.Read(reader, binary.BigEndian, &packet)
	if err != nil {
		return
	}

	handler.PacketItemCollect(packet.CollectedItem, packet.Collector)

	return
}

// packetIdObjectSpawn

func WriteObjectSpawn(writer io.Writer, entityId EntityId, objType ObjTypeId, position *AbsIntXyz, objectData *ObjectData) (err os.Error) {
	var packet = struct {
		PacketId byte
		EntityId EntityId
		ObjType  ObjTypeId
		X        AbsIntCoord
		Y        AbsIntCoord
		Z        AbsIntCoord
	}{
		packetIdObjectSpawn,
		entityId,
		objType,
		position.X,
		position.Y,
		position.Z,
	}

	if err = binary.Write(writer, binary.BigEndian, &packet); err != nil {
		return
	}

	if objectData == nil {
		err = binary.Write(writer, binary.BigEndian, int32(0))
	} else {
		err = binary.Write(writer, binary.BigEndian, objectData)
	}

	return
}

func readObjectSpawn(reader io.Reader, handler IClientPacketHandler) (err os.Error) {
	var packet struct {
		EntityId EntityId
		ObjType  ObjTypeId
		X        AbsIntCoord
		Y        AbsIntCoord
		Z        AbsIntCoord
		Field1   int32
	}

	if err = binary.Read(reader, binary.BigEndian, &packet); err != nil {
		return
	}

	var objectData *ObjectData

	if packet.Field1 != 0 {
		objectData = &ObjectData{
			Field1: packet.Field1,
		}

		if err = binary.Read(reader, binary.BigEndian, &objectData.Field2); err != nil {
			return
		}
	}

	handler.PacketObjectSpawn(
		packet.EntityId,
		packet.ObjType,
		&AbsIntXyz{packet.X, packet.Y, packet.Z},
		objectData)

	return
}

// packetIdEntitySpawn

func WriteEntitySpawn(writer io.Writer, entityId EntityId, mobType EntityMobType, position *AbsIntXyz, look *LookBytes, data []EntityMetadata) (err os.Error) {
	var packet = struct {
		PacketId byte
		EntityId EntityId
		MobType  EntityMobType
		X        AbsIntCoord
		Y        AbsIntCoord
		Z        AbsIntCoord
		Yaw      AngleBytes
		Pitch    AngleBytes
	}{
		packetIdEntitySpawn,
		entityId,
		mobType,
		position.X, position.Y, position.Z,
		look.Yaw, look.Pitch,
	}

	if err = binary.Write(writer, binary.BigEndian, &packet); err != nil {
		return
	}

	return writeEntityMetadataField(writer, data)
}

func readEntitySpawn(reader io.Reader, handler IClientPacketHandler) (err os.Error) {
	var packet struct {
		EntityId EntityId
		MobType  EntityMobType
		X        AbsIntCoord
		Y        AbsIntCoord
		Z        AbsIntCoord
		Yaw      AngleBytes
		Pitch    AngleBytes
	}

	err = binary.Read(reader, binary.BigEndian, &packet)
	if err != nil {
		return
	}

	metadata, err := readEntityMetadataField(reader)
	if err != nil {
		return
	}

	handler.PacketEntitySpawn(
		EntityId(packet.EntityId), packet.MobType,
		&AbsIntXyz{packet.X, packet.Y, packet.Z},
		&LookBytes{packet.Yaw, packet.Pitch},
		metadata)

	return err
}

// packetIdPaintingSpawn

func WritePaintingSpawn(writer io.Writer, entityId EntityId, title string, position *BlockXyz, paintingType PaintingTypeId) (err os.Error) {
	if err = binary.Write(writer, binary.BigEndian, &entityId); err != nil {
		return
	}

	if err = writeString16(writer, title); err != nil {
		return
	}

	var packetEnd = struct {
		X, Y, Z      BlockCoord
		PaintingType PaintingTypeId
	}{
		position.X, BlockCoord(position.Y), position.Z,
		paintingType,
	}

	return binary.Write(writer, binary.BigEndian, &packetEnd)
}

func readPaintingSpawn(reader io.Reader, handler IClientPacketHandler) (err os.Error) {
	var entityId EntityId

	if err = binary.Read(reader, binary.BigEndian, &entityId); err != nil {
		return
	}

	title, err := readString16(reader)
	if err != nil {
		return
	}

	var packetEnd struct {
		X, Y, Z      BlockCoord
		PaintingType PaintingTypeId
	}

	err = binary.Read(reader, binary.BigEndian, &packetEnd)
	if err != nil {
		return
	}

	handler.PacketPaintingSpawn(
		entityId,
		title,
		&BlockXyz{packetEnd.X, BlockYCoord(packetEnd.Y), packetEnd.Z},
		packetEnd.PaintingType)

	return
}

// packetIdExperienceOrb

func WriteExperienceOrb(writer io.Writer, entityId EntityId, position AbsIntXyz, count int16) (err os.Error) {
	var packet = struct {
		PacketId byte
		EntityId EntityId
		X, Y, Z  AbsIntCoord
		Count    int16
	}{
		packetIdExperienceOrb,
		entityId,
		position.X, position.Y, position.Z,
		count,
	}

	return binary.Write(writer, binary.BigEndian, &packet)
}

func readExperienceOrb(reader io.Reader, handler IClientPacketHandler) (err os.Error) {
	var packet struct {
		EntityId EntityId
		X, Y, Z  AbsIntCoord
		Count    int16
	}

	if err = binary.Read(reader, binary.BigEndian, &packet); err != nil {
		return
	}

	handler.PacketExperienceOrb(
		packet.EntityId,
		AbsIntXyz{packet.X, packet.Y, packet.Z},
		packet.Count)

	return
}

// packetIdEntityVelocity

func WriteEntityVelocity(writer io.Writer, entityId EntityId, velocity *Velocity) (err os.Error) {
	var packet = struct {
		packetId byte
		EntityId EntityId
		X, Y, Z  VelocityComponent
	}{
		packetIdEntityVelocity,
		entityId,
		velocity.X,
		velocity.Y,
		velocity.Z,
	}

	return binary.Write(writer, binary.BigEndian, &packet)
}

func readEntityVelocity(reader io.Reader, handler IClientPacketHandler) (err os.Error) {
	var packet struct {
		EntityId EntityId
		X, Y, Z  VelocityComponent
	}

	err = binary.Read(reader, binary.BigEndian, &packet)
	if err != nil {
		return
	}

	handler.PacketEntityVelocity(
		packet.EntityId,
		&Velocity{packet.X, packet.Y, packet.Z})

	return
}

// packetIdEntityDestroy

func WriteEntityDestroy(writer io.Writer, entityId EntityId) os.Error {
	var packet = struct {
		PacketId byte
		EntityId EntityId
	}{
		packetIdEntityDestroy,
		entityId,
	}
	return binary.Write(writer, binary.BigEndian, &packet)
}

func readEntityDestroy(reader io.Reader, handler IClientPacketHandler) (err os.Error) {
	var entityId EntityId

	err = binary.Read(reader, binary.BigEndian, &entityId)
	if err != nil {
		return
	}

	handler.PacketEntityDestroy(entityId)

	return
}

// packetIdEntity

func WriteEntity(writer io.Writer, entityId EntityId) (err os.Error) {
	var packet = struct {
		PacketId byte
		EntityId EntityId
	}{
		packetIdEntity,
		entityId,
	}

	return binary.Write(writer, binary.BigEndian, &packet)
}

func readEntity(reader io.Reader, handler IClientPacketHandler) (err os.Error) {
	var entityId EntityId

	err = binary.Read(reader, binary.BigEndian, &entityId)
	if err != nil {
		return
	}

	handler.PacketEntity(entityId)

	return
}

// packetIdEntityRelMove

func WriteEntityRelMove(writer io.Writer, entityId EntityId, movement *RelMove) (err os.Error) {
	var packet = struct {
		PacketId byte
		EntityId EntityId
		X, Y, Z  RelMoveCoord
	}{
		packetIdEntityRelMove,
		entityId,
		movement.X,
		movement.Y,
		movement.Z,
	}

	if err = binary.Write(writer, binary.BigEndian, &packet); err != nil {
		return
	}

	return
}

func readEntityRelMove(reader io.Reader, handler IClientPacketHandler) (err os.Error) {
	var packet struct {
		EntityId EntityId
		X, Y, Z  RelMoveCoord
	}

	err = binary.Read(reader, binary.BigEndian, &packet)
	if err != nil {
		return
	}

	handler.PacketEntityRelMove(
		packet.EntityId,
		&RelMove{packet.X, packet.Y, packet.Z})

	return
}

// packetIdEntityLook

func WriteEntityLook(writer io.Writer, entityId EntityId, look *LookBytes) os.Error {
	var packet = struct {
		PacketId byte
		EntityId EntityId
		Yaw      AngleBytes
		Pitch    AngleBytes
	}{
		packetIdEntityLook,
		entityId,
		look.Yaw,
		look.Pitch,
	}
	return binary.Write(writer, binary.BigEndian, &packet)
}

func readEntityLook(reader io.Reader, handler IClientPacketHandler) (err os.Error) {
	var packet struct {
		EntityId EntityId
		Yaw      AngleBytes
		Pitch    AngleBytes
	}

	err = binary.Read(reader, binary.BigEndian, &packet)
	if err != nil {
		return
	}

	handler.PacketEntityLook(
		packet.EntityId,
		&LookBytes{packet.Yaw, packet.Pitch})

	return
}

// packetIdEntityLookAndRelMove

func WriteEntityLookAndRelMove(writer io.Writer, entityId EntityId, movement *RelMove, look *LookBytes) (err os.Error) {
	var packet = struct {
		PacketId byte
		EntityId EntityId
		X, Y, Z  RelMoveCoord
		Yaw      AngleBytes
		Pitch    AngleBytes
	}{
		packetIdEntityLookAndRelMove,
		entityId,
		movement.X, movement.Y, movement.Z,
		look.Yaw, look.Pitch,
	}

	return binary.Write(writer, binary.BigEndian, &packet)
}

func readEntityLookAndRelMove(reader io.Reader, handler IClientPacketHandler) (err os.Error) {
	var packet struct {
		EntityId EntityId
		X, Y, Z  RelMoveCoord
		Yaw      AngleBytes
		Pitch    AngleBytes
	}

	err = binary.Read(reader, binary.BigEndian, &packet)
	if err != nil {
		return
	}

	handler.PacketEntityRelMove(
		packet.EntityId,
		&RelMove{packet.X, packet.Y, packet.Z})

	handler.PacketEntityLook(
		packet.EntityId,
		&LookBytes{packet.Yaw, packet.Pitch})

	return
}

// packetIdEntityTeleport

func WriteEntityTeleport(writer io.Writer, entityId EntityId, position *AbsIntXyz, look *LookBytes) os.Error {
	var packet = struct {
		PacketId byte
		EntityId EntityId
		X        AbsIntCoord
		Y        AbsIntCoord
		Z        AbsIntCoord
		Yaw      AngleBytes
		Pitch    AngleBytes
	}{
		packetIdEntityTeleport,
		entityId,
		position.X,
		position.Y,
		position.Z,
		look.Yaw,
		look.Pitch,
	}
	return binary.Write(writer, binary.BigEndian, &packet)
}

func readEntityTeleport(reader io.Reader, handler IClientPacketHandler) (err os.Error) {
	var packet struct {
		EntityId EntityId
		X        AbsIntCoord
		Y        AbsIntCoord
		Z        AbsIntCoord
		Yaw      AngleBytes
		Pitch    AngleBytes
	}

	if err = binary.Read(reader, binary.BigEndian, &packet); err != nil {
		return
	}

	handler.PacketEntityTeleport(
		packet.EntityId,
		&AbsIntXyz{
			packet.X,
			packet.Y,
			packet.Z,
		},
		&LookBytes{
			packet.Yaw,
			packet.Pitch,
		})

	return
}

// packetIdEntityStatus

func WriteEntityStatus(writer io.Writer, entityId EntityId, status EntityStatus) (err os.Error) {
	var packet = struct {
		PacketId byte
		EntityId EntityId
		Status   EntityStatus
	}{
		packetIdEntityStatus,
		entityId,
		status,
	}

	return binary.Write(writer, binary.BigEndian, &packet)
}

func readEntityStatus(reader io.Reader, handler IClientPacketHandler) (err os.Error) {
	var packet struct {
		EntityId EntityId
		Status   EntityStatus
	}

	err = binary.Read(reader, binary.BigEndian, &packet)
	if err != nil {
		return
	}

	handler.PacketEntityStatus(packet.EntityId, packet.Status)

	return
}

// packetIdEntityMetadata

func WriteEntityMetadata(writer io.Writer, entityId EntityId, data []EntityMetadata) (err os.Error) {
	var packet = struct {
		PacketId byte
		EntityId EntityId
	}{
		packetIdEntityMetadata,
		entityId,
	}

	if err = binary.Write(writer, binary.BigEndian, &packet); err != nil {
		return
	}

	return writeEntityMetadataField(writer, data)
}

func readEntityMetadata(reader io.Reader, handler IClientPacketHandler) (err os.Error) {
	var entityId EntityId

	if err = binary.Read(reader, binary.BigEndian, &entityId); err != nil {
		return
	}

	metadata, err := readEntityMetadataField(reader)
	if err != nil {
		return
	}

	handler.PacketEntityMetadata(entityId, metadata)

	return
}

// packetIdEntityEffect

func WriteEntityEffect(writer io.Writer, entityId EntityId, effect EntityEffect, value int8, duration int16) (err os.Error) {
	var packet = struct {
		PacketId byte
		EntityId EntityId
		Effect   EntityEffect
		Value    int8
		Duration int16
	}{
		packetIdEntityEffect,
		entityId,
		effect,
		value,
		duration,
	}

	return binary.Write(writer, binary.BigEndian, &packet)
}

func readEntityEffect(reader io.Reader, handler IClientPacketHandler) (err os.Error) {
	var packet struct {
		EntityId EntityId
		Effect   EntityEffect
		Value    int8
		Duration int16
	}

	if err = binary.Read(reader, binary.BigEndian, &packet); err != nil {
		return
	}

	handler.PacketEntityEffect(packet.EntityId, packet.Effect, packet.Value, packet.Duration)

	return
}

// packetIdEntityRemoveEffect

func WriteEntityRemoveEffect(writer io.Writer, entityId EntityId, effect EntityEffect) (err os.Error) {
	var packet = struct {
		PacketId byte
		EntityId EntityId
		Effect   EntityEffect
	}{
		packetIdEntityEffect,
		entityId,
		effect,
	}

	return binary.Write(writer, binary.BigEndian, &packet)
}

func readEntityRemoveEffect(reader io.Reader, handler IClientPacketHandler) (err os.Error) {
	var packet struct {
		EntityId EntityId
		Effect   EntityEffect
	}

	if err = binary.Read(reader, binary.BigEndian, &packet); err != nil {
		return
	}

	handler.PacketEntityRemoveEffect(packet.EntityId, packet.Effect)

	return
}

// packetIdPlayerExperience

func WritePlayerExperience(writer io.Writer, experience, level int8, totalExperience int16) (err os.Error) {
	var packet = struct {
		PacketId        byte
		Experience      int8
		Level           int8
		TotalExperience int16
	}{
		packetIdPlayerExperience,
		experience,
		level,
		totalExperience,
	}

	return binary.Write(writer, binary.BigEndian, &packet)
}

func readPlayerExperience(reader io.Reader, handler IClientPacketHandler) (err os.Error) {
	var packet struct {
		Experience      int8
		Level           int8
		TotalExperience int16
	}

	if err = binary.Read(reader, binary.BigEndian, &packet); err != nil {
		return
	}

	handler.PacketPlayerExperience(packet.Experience, packet.Level, packet.TotalExperience)

	return
}

// packetIdPreChunk

func WritePreChunk(writer io.Writer, chunkLoc *ChunkXz, mode ChunkLoadMode) os.Error {
	var packet = struct {
		PacketId byte
		X        ChunkCoord
		Z        ChunkCoord
		Mode     ChunkLoadMode
	}{
		packetIdPreChunk,
		chunkLoc.X,
		chunkLoc.Z,
		mode,
	}
	return binary.Write(writer, binary.BigEndian, &packet)
}

func readPreChunk(reader io.Reader, handler IClientPacketHandler) (err os.Error) {
	var packet struct {
		X    ChunkCoord
		Z    ChunkCoord
		Mode ChunkLoadMode
	}

	err = binary.Read(reader, binary.BigEndian, &packet)
	if err != nil {
		return
	}

	handler.PacketPreChunk(&ChunkXz{packet.X, packet.Z}, packet.Mode)

	return
}

// packetIdMapChunk

func WriteMapChunk(writer io.Writer, chunkLoc *ChunkXz, blocks, blockData, blockLight, skyLight []byte) (err os.Error) {
	buf := &bytes.Buffer{}
	compressed, err := zlib.NewWriter(buf)
	if err != nil {
		return
	}

	compressed.Write(blocks)
	compressed.Write(blockData)
	compressed.Write(blockLight)
	compressed.Write(skyLight)
	compressed.Close()
	bs := buf.Bytes()

	chunkCornerLoc := chunkLoc.ChunkCornerBlockXY()

	var packet = struct {
		PacketId         byte
		X                BlockCoord
		Y                int16
		Z                BlockCoord
		SizeX            SubChunkSizeCoord
		SizeY            SubChunkSizeCoord
		SizeZ            SubChunkSizeCoord
		CompressedLength int32
	}{
		packetIdMapChunk,
		chunkCornerLoc.X,
		int16(chunkCornerLoc.Y),
		chunkCornerLoc.Z,
		ChunkSizeH - 1,
		ChunkSizeY - 1,
		ChunkSizeH - 1,
		int32(len(bs)),
	}

	err = binary.Write(writer, binary.BigEndian, &packet)
	if err != nil {
		return
	}
	err = binary.Write(writer, binary.BigEndian, bs)
	return
}

func readMapChunk(reader io.Reader, handler IClientPacketHandler) (err os.Error) {
	var packet struct {
		X                BlockCoord
		Y                int16
		Z                BlockCoord
		SizeX            SubChunkSizeCoord
		SizeY            SubChunkSizeCoord
		SizeZ            SubChunkSizeCoord
		CompressedLength int32
	}

	err = binary.Read(reader, binary.BigEndian, &packet)
	if err != nil {
		return
	}

	// TODO extract block data from raw data field, and pass on to handler
	data := make([]byte, packet.CompressedLength)
	_, err = io.ReadFull(reader, data)
	if err != nil {
		return
	}

	handler.PacketMapChunk(
		&BlockXyz{packet.X, BlockYCoord(packet.Y), packet.Z},
		&SubChunkSize{packet.SizeX, packet.SizeY, packet.SizeZ},
		data)
	return
}

// packetIdBlockChangeMulti

func WriteBlockChangeMulti(writer io.Writer, chunkLoc *ChunkXz, blockCoords []SubChunkXyz, blockTypes []BlockId, blockMetaData []byte) (err os.Error) {
	// NOTE that we don't yet check that blockCoords, blockTypes and
	// blockMetaData are of the same length.

	var packet = struct {
		PacketId byte
		ChunkX   ChunkCoord
		ChunkZ   ChunkCoord
		Count    int16
	}{
		packetIdBlockChangeMulti,
		chunkLoc.X, chunkLoc.Z,
		int16(len(blockCoords)),
	}

	if err = binary.Write(writer, binary.BigEndian, &packet); err != nil {
		return
	}

	rawBlockLocs := make([]int16, packet.Count)
	for index, blockCoord := range blockCoords {
		rawBlockCoord := int16(0)
		rawBlockCoord |= int16((blockCoord.X & 0x0f) << 12)
		rawBlockCoord |= int16((blockCoord.Y & 0xff))
		rawBlockCoord |= int16((blockCoord.Z & 0x0f) << 8)
		rawBlockLocs[index] = rawBlockCoord
	}

	binary.Write(writer, binary.BigEndian, rawBlockLocs)

	return
}

func readBlockChangeMulti(reader io.Reader, handler IClientPacketHandler) (err os.Error) {
	var packet struct {
		ChunkX ChunkCoord
		ChunkZ ChunkCoord
		Count  int16
	}

	if err = binary.Read(reader, binary.BigEndian, &packet); err != nil {
		return
	}

	rawBlockLocs := make([]int16, packet.Count)
	blockTypes := make([]BlockId, packet.Count)
	// blockMetadata array appears to represent one block per byte
	blockMetadata := make([]byte, packet.Count)

	err = binary.Read(reader, binary.BigEndian, rawBlockLocs)
	err = binary.Read(reader, binary.BigEndian, blockTypes)
	err = binary.Read(reader, binary.BigEndian, blockMetadata)

	blockLocs := make([]SubChunkXyz, packet.Count)
	for index, rawLoc := range rawBlockLocs {
		blockLocs[index] = SubChunkXyz{
			X: SubChunkCoord(rawLoc >> 12),
			Y: SubChunkCoord(rawLoc & 0xff),
			Z: SubChunkCoord((rawLoc >> 8) & 0x0f),
		}
	}

	handler.PacketBlockChangeMulti(
		&ChunkXz{packet.ChunkX, packet.ChunkZ},
		blockLocs,
		blockTypes,
		blockMetadata)

	return
}

// packetIdBlockChange

func WriteBlockChange(writer io.Writer, blockLoc *BlockXyz, blockType BlockId, blockMetaData byte) (err os.Error) {
	var packet = struct {
		PacketId      byte
		X             BlockCoord
		Y             BlockYCoord
		Z             BlockCoord
		BlockType     BlockId
		BlockMetadata byte
	}{
		packetIdBlockChange,
		blockLoc.X,
		blockLoc.Y,
		blockLoc.Z,
		blockType,
		blockMetaData,
	}
	err = binary.Write(writer, binary.BigEndian, &packet)
	return
}

func readBlockChange(reader io.Reader, handler IClientPacketHandler) (err os.Error) {
	var packet struct {
		X             BlockCoord
		Y             BlockYCoord
		Z             BlockCoord
		BlockType     BlockId
		BlockMetadata byte
	}

	err = binary.Read(reader, binary.BigEndian, &packet)
	if err != nil {
		return
	}

	handler.PacketBlockChange(
		&BlockXyz{packet.X, packet.Y, packet.Z},
		packet.BlockType,
		packet.BlockMetadata)

	return
}

// packetIdNoteBlockPlay

func WriteNoteBlockPlay(writer io.Writer, position *BlockXyz, instrument InstrumentId, pitch NotePitch) (err os.Error) {
	var packet = struct {
		PacketId   byte
		X          BlockCoord
		Y          BlockYCoord
		Z          BlockCoord
		Instrument InstrumentId
		Pitch      NotePitch
	}{
		packetIdNoteBlockPlay,
		position.X, position.Y, position.Z,
		instrument,
		pitch,
	}

	return binary.Write(writer, binary.BigEndian, &packet)
}

func readNoteBlockPlay(reader io.Reader, handler IClientPacketHandler) (err os.Error) {
	var packet struct {
		X          BlockCoord
		Y          BlockYCoord
		Z          BlockCoord
		Instrument InstrumentId
		Pitch      NotePitch
	}

	if err = binary.Read(reader, binary.BigEndian, &packet); err != nil {
		return
	}

	handler.PacketNoteBlockPlay(
		&BlockXyz{packet.X, packet.Y, packet.Z},
		packet.Instrument,
		packet.Pitch)

	return
}

// packetIdExplosion

// TODO introduce better types for ExplosionOffsetXyz and the floats in the
// packet structure when the packet is better understood.

type ExplosionOffsetXyz struct {
	X, Y, Z int8
}

func WriteExplosion(writer io.Writer, position *AbsXyz, power float32, blockOffsets []ExplosionOffsetXyz) (err os.Error) {
	var packet = struct {
		PacketId byte
		// NOTE AbsCoord is just a guess for now
		X, Y, Z AbsCoord
		// NOTE Power isn't known to be a good name for this field
		Power     float32
		NumBlocks int32
	}{
		packetIdExplosion,
		position.X, position.Y, position.Z,
		power,
		int32(len(blockOffsets)),
	}

	if err = binary.Write(writer, binary.BigEndian, &packet); err != nil {
		return
	}

	return binary.Write(writer, binary.BigEndian, blockOffsets)
}

func readExplosion(reader io.Reader, handler IClientPacketHandler) (err os.Error) {
	var packet struct {
		// NOTE AbsCoord is just a guess for now
		X, Y, Z AbsCoord
		// NOTE Power isn't known to be a good name for this field
		Power     float32
		NumBlocks int32
	}

	if err = binary.Read(reader, binary.BigEndian, &packet); err != nil {
		return
	}

	// TODO put sensible size limits on how big arrays read from network could
	// be, both here and in other places
	blockOffsets := make([]ExplosionOffsetXyz, packet.NumBlocks)

	if err = binary.Read(reader, binary.BigEndian, blockOffsets); err != nil {
		return
	}

	handler.PacketExplosion(
		&AbsXyz{packet.X, packet.Y, packet.Z},
		packet.Power,
		blockOffsets)

	return
}

// packetIdSoundEffect

func WriteSoundEffect(writer io.Writer, sound SoundEffect, position BlockXyz, data int32) (err os.Error) {
	var packet = struct {
		PacketId byte
		Sound    SoundEffect
		X        BlockCoord
		Y        BlockYCoord
		Z        BlockCoord
		Data     int32
	}{
		packetIdSoundEffect,
		sound,
		position.X, position.Y, position.Z,
		data,
	}

	return binary.Write(writer, binary.BigEndian, &packet)
}

func readSoundEffect(reader io.Reader, handler IClientPacketHandler) (err os.Error) {
	var packet struct {
		Sound SoundEffect
		X     BlockCoord
		Y     BlockYCoord
		Z     BlockCoord
		Data  int32
	}

	if err = binary.Read(reader, binary.BigEndian, &packet); err != nil {
		return
	}

	handler.PacketSoundEffect(
		packet.Sound,
		BlockXyz{packet.X, packet.Y, packet.Z},
		packet.Data,
	)

	return
}

// packetIdState

func WriteState(writer io.Writer, reason, gameMode byte) (err os.Error) {
	var packet = struct {
		PacketId byte
		Reason   byte
		GameMode byte
	}{
		packetIdState,
		reason,
		gameMode,
	}

	return binary.Write(writer, binary.BigEndian, &packet)
}

func readState(reader io.Reader, handler IClientPacketHandler) (err os.Error) {
	var packet struct {
		Reason   byte
		GameMode byte
	}
	if err = binary.Read(reader, binary.BigEndian, &packet); err != nil {
		return
	}

	handler.PacketState(packet.Reason, packet.GameMode)
	return
}

// packetIdWeather

func WriteWeather(writer io.Writer, entityId EntityId, raining bool, position *AbsIntXyz) (err os.Error) {
	var packet = struct {
		PacketId byte
		EntityId EntityId
		Raining  byte
		X, Y, Z  AbsIntCoord
	}{
		packetIdWeather,
		entityId,
		boolToByte(raining),
		position.X, position.Y, position.Z,
	}

	return binary.Write(writer, binary.BigEndian, &packet)
}

func readWeather(reader io.Reader, handler IClientPacketHandler) (err os.Error) {
	var packet struct {
		EntityId EntityId
		Raining  byte
		X, Y, Z  AbsIntCoord
	}

	err = binary.Read(reader, binary.BigEndian, &packet)
	if err != nil {
		return
	}

	handler.PacketWeather(
		packet.EntityId,
		byteToBool(packet.Raining),
		&AbsIntXyz{packet.X, packet.Y, packet.Z},
	)

	return
}

// packetIdWindowOpen

func WriteWindowOpen(writer io.Writer, windowId WindowId, invTypeId InvTypeId, windowTitle string, numSlots byte) (err os.Error) {
	var packet = struct {
		PacketId  byte
		WindowId  WindowId
		InvTypeId InvTypeId
	}{
		packetIdWindowOpen,
		windowId,
		invTypeId,
	}

	if err = binary.Write(writer, binary.BigEndian, &packet); err != nil {
		return
	}

	if err = writeString16(writer, windowTitle); err != nil {
		return
	}

	return binary.Write(writer, binary.BigEndian, numSlots)
}

func readWindowOpen(reader io.Reader, handler IClientPacketHandler) (err os.Error) {
	var packet struct {
		WindowId  WindowId
		InvTypeId InvTypeId
	}

	if err = binary.Read(reader, binary.BigEndian, &packet); err != nil {
		return
	}

	windowTitle, err := readString16(reader)
	if err != nil {
		return
	}

	var numSlots byte
	if err = binary.Read(reader, binary.BigEndian, &numSlots); err != nil {
		return
	}

	handler.PacketWindowOpen(packet.WindowId, packet.InvTypeId, windowTitle, numSlots)

	return
}

// packetIdWindowClose

func WriteWindowClose(writer io.Writer, windowId WindowId) (err os.Error) {
	var packet = struct {
		PacketId byte
		WindowId WindowId
	}{
		packetIdWindowClose,
		windowId,
	}

	if err = binary.Write(writer, binary.BigEndian, &packet); err != nil {
		return
	}

	return
}

func readWindowClose(reader io.Reader, handler IServerPacketHandler) (err os.Error) {
	var windowId WindowId

	if err = binary.Read(reader, binary.BigEndian, &windowId); err != nil {
		return
	}

	handler.PacketWindowClose(windowId)

	return
}

// packetIdWindowClick

func WriteWindowClick(writer io.Writer, windowId WindowId, slot SlotId, rightClick bool, txId TxId, shiftClick bool, expectedSlot WindowSlot) (err os.Error) {
	var packet = struct {
		PacketId   byte
		WindowId   WindowId
		Slot       SlotId
		RightClick byte
		TxId       TxId
		ShiftClick byte
		ItemTypeId ItemTypeId
	}{
		packetIdWindowClick,
		windowId,
		slot,
		boolToByte(rightClick),
		txId,
		boolToByte(shiftClick),
		expectedSlot.ItemTypeId,
	}

	if err = binary.Write(writer, binary.BigEndian, &packet); err != nil {
		return
	}

	if expectedSlot.ItemTypeId != -1 {
		var packetEnd = struct {
			Amount ItemCount
			Data   ItemData
		}{
			expectedSlot.Count,
			expectedSlot.Data,
		}
		err = binary.Write(writer, binary.BigEndian, &packetEnd)
	}

	return
}

func readWindowClick(reader io.Reader, handler IServerPacketHandler) (err os.Error) {
	var packetStart struct {
		WindowId   WindowId
		Slot       SlotId
		RightClick byte
		TxId       TxId
		ShiftClick byte
		ItemTypeId ItemTypeId
	}

	err = binary.Read(reader, binary.BigEndian, &packetStart)
	if err != nil {
		return
	}

	var packetEnd struct {
		Amount ItemCount
		Data   ItemData
	}

	if packetStart.ItemTypeId != -1 {
		err = binary.Read(reader, binary.BigEndian, &packetEnd)
		if err != nil {
			return
		}
	}

	expectedSlot := &WindowSlot{
		packetStart.ItemTypeId,
		packetEnd.Amount,
		packetEnd.Data,
	}

	handler.PacketWindowClick(
		packetStart.WindowId,
		packetStart.Slot,
		byteToBool(packetStart.RightClick),
		packetStart.TxId,
		byteToBool(packetStart.ShiftClick),
		expectedSlot)

	return
}

// packetIdWindowSetSlot

func WriteWindowSetSlot(writer io.Writer, windowId WindowId, slot SlotId, itemTypeId ItemTypeId, amount ItemCount, data ItemData) (err os.Error) {
	var packet = struct {
		PacketId byte
		WindowId WindowId
		Slot     SlotId
	}{
		packetIdWindowSetSlot,
		windowId,
		slot,
	}

	if err = binary.Write(writer, binary.BigEndian, &packet); err != nil {
		return
	}

	if itemTypeId > 0 {
		var packetEnd = struct {
			ItemTypeId ItemTypeId
			Amount     ItemCount
			Data       ItemData
		}{
			itemTypeId,
			amount,
			data,
		}
		err = binary.Write(writer, binary.BigEndian, &packetEnd)
	} else {
		err = binary.Write(writer, binary.BigEndian, ItemTypeId(-1))
	}

	return err
}

func readWindowSetSlot(reader io.Reader, handler IClientPacketHandler) (err os.Error) {
	var packetStart struct {
		WindowId   WindowId
		Slot       SlotId
		ItemTypeId ItemTypeId
	}

	err = binary.Read(reader, binary.BigEndian, &packetStart)
	if err != nil {
		return
	}

	var packetEnd struct {
		Amount ItemCount
		Data   ItemData
	}

	if packetStart.ItemTypeId != -1 {
		err = binary.Read(reader, binary.BigEndian, &packetEnd)
		if err != nil {
			return
		}
	} else {
		// We use zero as the null item internally.
		packetStart.ItemTypeId = 0
	}

	handler.PacketWindowSetSlot(
		packetStart.WindowId,
		packetStart.Slot,
		packetStart.ItemTypeId,
		packetEnd.Amount,
		packetEnd.Data)

	return
}

// packetIdWindowItems

func WriteWindowItems(writer io.Writer, windowId WindowId, items []WindowSlot) (err os.Error) {
	var packet = struct {
		PacketId byte
		WindowId WindowId
		Count    int16
	}{
		packetIdWindowItems,
		windowId,
		int16(len(items)),
	}

	if err = binary.Write(writer, binary.BigEndian, &packet); err != nil {
		return
	}

	for i := range items {
		slot := &items[i]
		if slot.ItemTypeId > 0 {
			err = binary.Write(writer, binary.BigEndian, slot)
		} else {
			err = binary.Write(writer, binary.BigEndian, ItemTypeId(-1))
		}

		if err != nil {
			return
		}
	}

	return
}

func readWindowItems(reader io.Reader, handler IClientPacketHandler) (err os.Error) {
	var packetStart struct {
		WindowId WindowId
		Count    int16
	}

	err = binary.Read(reader, binary.BigEndian, &packetStart)
	if err != nil {
		return
	}

	var itemTypeId ItemTypeId

	items := make([]WindowSlot, 0, packetStart.Count)

	var itemInfo struct {
		Count ItemCount
		Data  ItemData
	}

	for i := int16(0); i < packetStart.Count; i++ {
		err = binary.Read(reader, binary.BigEndian, &itemTypeId)
		if err != nil {
			return
		}

		if itemTypeId > 0 {
			err = binary.Read(reader, binary.BigEndian, &itemInfo)
			if err != nil {
				return
			}
		} else if itemTypeId == 0 {
			err = os.NewError("Invalid item ID 0 in window")
			return
		} else {
			// We use zero as the null item internally.
			itemTypeId = 0
		}

		items = append(items, WindowSlot{
			ItemTypeId: itemTypeId,
			Count:      itemInfo.Count,
			Data:       itemInfo.Data,
		})
	}

	handler.PacketWindowItems(
		packetStart.WindowId,
		items)

	return
}

// packetIdWindowProgressBar

func WriteWindowProgressBar(writer io.Writer, windowId WindowId, prgBarId PrgBarId, value PrgBarValue) os.Error {
	var packet = struct {
		PacketId byte
		WindowId WindowId
		PrgBarId PrgBarId
		Value    PrgBarValue
	}{
		packetIdWindowProgressBar,
		windowId,
		prgBarId,
		value,
	}

	return binary.Write(writer, binary.BigEndian, &packet)
}

func readWindowProgressBar(reader io.Reader, handler IClientPacketHandler) (err os.Error) {
	var packet struct {
		WindowId WindowId
		PrgBarId PrgBarId
		Value    PrgBarValue
	}

	if err = binary.Read(reader, binary.BigEndian, &packet); err != nil {
		return
	}

	handler.PacketWindowProgressBar(packet.WindowId, packet.PrgBarId, packet.Value)

	return
}

// packetIdWindowTransaction

func WriteWindowTransaction(writer io.Writer, windowId WindowId, txId TxId, accepted bool) (err os.Error) {
	var packet = struct {
		PacketId byte
		WindowId WindowId
		TxId     TxId
		Accepted byte
	}{
		packetIdWindowTransaction,
		windowId,
		txId,
		boolToByte(accepted),
	}

	return binary.Write(writer, binary.BigEndian, &packet)
}

func readWindowTransaction(reader io.Reader, handler IPacketHandler) (err os.Error) {
	var packet struct {
		WindowId WindowId
		TxId     TxId
		Accepted byte
	}

	if err = binary.Read(reader, binary.BigEndian, &packet); err != nil {
		return
	}

	handler.PacketWindowTransaction(packet.WindowId, packet.TxId, byteToBool(packet.Accepted))

	return
}

// packetIdQuickbarSlotUpdate

func WriteQuickbarSlotUpdate(writer io.Writer, slot SlotId, itemId ItemTypeId, count ItemCount, data ItemData) (err os.Error) {
	var packet = struct {
		PacketId byte
		Slot     SlotId
		ItemId   ItemTypeId
		Count    ItemCount
		Data     ItemData
	}{
		packetIdQuickbarSlotUpdate,
		slot,
		itemId,
		count,
		data,
	}

	return binary.Write(writer, binary.BigEndian, &packet)
}

func readQuickbarSlotUpdate(reader io.Reader, handler IClientPacketHandler) (err os.Error) {
	var packet struct {
		Slot   SlotId
		ItemId ItemTypeId
		Count  ItemCount
		Data   ItemData
	}

	if err = binary.Read(reader, binary.BigEndian, &packet); err != nil {
		return
	}

	handler.PacketQuickbarSlotUpdate(packet.Slot, packet.ItemId, packet.Count, packet.Data)

	return
}

// packetIdSignUpdate

func WriteSignUpdate(writer io.Writer, position *BlockXyz, lines [4]string) (err os.Error) {
	var packet = struct {
		PacketId byte
		X        BlockCoord
		Y        BlockYCoord
		Z        BlockCoord
	}{
		packetIdSignUpdate,
		position.X, position.Y, position.Z,
	}

	if err = binary.Write(writer, binary.BigEndian, &packet); err != nil {
		return
	}

	for _, line := range lines {
		if err = writeString16(writer, line); err != nil {
			return
		}
	}

	return
}

func readSignUpdate(reader io.Reader, handler IPacketHandler) (err os.Error) {
	var packet struct {
		X BlockCoord
		Y BlockYCoord
		Z BlockCoord
	}

	if err = binary.Read(reader, binary.BigEndian, &packet); err != nil {
		return
	}

	var lines [4]string

	for i := 0; i < len(lines); i++ {
		if lines[i], err = readString16(reader); err != nil {
			return
		}
	}

	handler.PacketSignUpdate(
		&BlockXyz{packet.X, packet.Y, packet.Z},
		lines)

	return
}

// packetIdItemData

func WriteItemData(writer io.Writer, itemTypeId ItemTypeId, itemDataId ItemData, data []byte) (err os.Error) {
	var packet = struct {
		PacketId   byte
		ItemTypeId ItemTypeId
		ItemDataId ItemData
		DataLength byte
	}{
		packetIdItemData,
		itemTypeId,
		itemDataId,
		byte(len(data)),
	}

	if err = binary.Write(writer, binary.BigEndian, &packet); err != nil {
		return
	}

	_, err = writer.Write(data)

	return
}

func readItemData(reader io.Reader, handler IClientPacketHandler) (err os.Error) {
	var packet struct {
		ItemTypeId ItemTypeId
		ItemDataId ItemData
		DataLength byte
	}

	if err = binary.Read(reader, binary.BigEndian, &packet); err != nil {
		return
	}

	data := make([]byte, packet.DataLength)
	if _, err = io.ReadFull(reader, data); err != nil {
		return
	}

	handler.PacketItemData(packet.ItemTypeId, packet.ItemDataId, data)

	return
}

// packetIdIncrementStatistic

func WriteIncrementStatistic(writer io.Writer, statisticId StatisticId, delta int8) (err os.Error) {
	var packet = struct {
		PacketId    byte
		StatisticId StatisticId
		Delta       int8
	}{
		packetIdIncrementStatistic,
		statisticId,
		delta,
	}
	return binary.Write(writer, binary.BigEndian, &packet)
}

func readIncrementStatistic(reader io.Reader, handler IClientPacketHandler) (err os.Error) {
	var packet struct {
		StatisticId StatisticId
		Delta       int8
	}

	err = binary.Read(reader, binary.BigEndian, &packet)
	if err != nil {
		return
	}

	handler.PacketIncrementStatistic(packet.StatisticId, packet.Delta)

	return
}

// packetIdUserListItem

func WriteUserListItem(writer io.Writer, username string, online bool, pingMs int16) (err os.Error) {
	if err = binary.Write(writer, binary.BigEndian, byte(packetIdUserListItem)); err != nil {
		return
	}

	if err = writeString16(writer, username); err != nil {
		return
	}

	var packetEnd = struct {
		Online byte
		PingMs int16
	}{
		boolToByte(online),
		pingMs,
	}

	return binary.Write(writer, binary.BigEndian, &packetEnd)
}

func readUserListItem(reader io.Reader, handler IClientPacketHandler) (err os.Error) {
	username, err := readString16(reader)
	if err != nil {
		return
	}

	var packetEnd struct {
		Online byte
		PingMs int16
	}
	if err = binary.Read(reader, binary.BigEndian, &packetEnd); err != nil {
		return
	}

	handler.PacketUserListItem(username, byteToBool(packetEnd.Online), packetEnd.PingMs)

	return
}

// packetIdServerListPing

func WriteServerListPing(writer io.Writer) (err os.Error) {
	return binary.Write(writer, binary.BigEndian, byte(packetIdServerListPing))
}

func readServerListPing(reader io.Reader, handler IServerPacketHandler) (err os.Error) {
	handler.PacketServerListPing()
	return
}

// packetIdDisconnect

func WriteDisconnect(writer io.Writer, reason string) (err os.Error) {
	buf := &bytes.Buffer{}
	binary.Write(buf, binary.BigEndian, byte(packetIdDisconnect))
	writeString16(buf, reason)
	_, err = writer.Write(buf.Bytes())
	return
}

func readDisconnect(reader io.Reader, handler IPacketHandler) (err os.Error) {
	reason, err := readString16(reader)
	if err != nil {
		return
	}

	handler.PacketDisconnect(reason)
	return
}

// End of packet reader/writer functions


type commonPacketHandler func(io.Reader, IPacketHandler) os.Error
type serverPacketHandler func(io.Reader, IServerPacketHandler) os.Error
type clientPacketHandler func(io.Reader, IClientPacketHandler) os.Error

type commonPacketReaderMap map[byte]commonPacketHandler
type serverPacketReaderMap map[byte]serverPacketHandler
type clientPacketReaderMap map[byte]clientPacketHandler

// Common packet mapping
var commonReadFns = commonPacketReaderMap{
	packetIdKeepAlive:           readKeepAlive,
	packetIdChatMessage:         readChatMessage,
	packetIdEntityAction:        readEntityAction,
	packetIdUseEntity:           readUseEntity,
	packetIdRespawn:             readRespawn,
	packetIdPlayerPosition:      readPlayerPosition,
	packetIdPlayerLook:          readPlayerLook,
	packetIdPlayerBlockHit:      readPlayerBlockHit,
	packetIdPlayerBlockInteract: readPlayerBlockInteract,
	packetIdEntityAnimation:     readEntityAnimation,
	packetIdWindowTransaction:   readWindowTransaction,
	packetIdSignUpdate:          readSignUpdate,
	packetIdDisconnect:          readDisconnect,
}

// Client->server specific packet mapping
var serverReadFns = serverPacketReaderMap{
	packetIdPlayer:             readPlayer,
	packetIdPlayerPositionLook: serverReadPlayerPositionLook,
	packetIdWindowClick:        readWindowClick,
	packetIdHoldingChange:      readHoldingChange,
	packetIdWindowClose:        readWindowClose,
}

// Server->client specific packet mapping
var clientReadFns = clientPacketReaderMap{
	packetIdLogin:                clientReadLogin,
	packetIdTimeUpdate:           readTimeUpdate,
	packetIdEntityEquipment:      readEntityEquipment,
	packetIdSpawnPosition:        readSpawnPosition,
	packetIdUpdateHealth:         readUpdateHealth,
	packetIdPlayerPositionLook:   clientReadPlayerPositionLook,
	packetIdBedUse:               readBedUse,
	packetIdNamedEntitySpawn:     readNamedEntitySpawn,
	packetIdItemSpawn:            readItemSpawn,
	packetIdItemCollect:          readItemCollect,
	packetIdObjectSpawn:          readObjectSpawn,
	packetIdEntitySpawn:          readEntitySpawn,
	packetIdPaintingSpawn:        readPaintingSpawn,
	packetIdExperienceOrb:        readExperienceOrb,
	packetIdEntityVelocity:       readEntityVelocity,
	packetIdEntityDestroy:        readEntityDestroy,
	packetIdEntity:               readEntity,
	packetIdEntityRelMove:        readEntityRelMove,
	packetIdEntityLook:           readEntityLook,
	packetIdEntityLookAndRelMove: readEntityLookAndRelMove,
	packetIdEntityTeleport:       readEntityTeleport,
	packetIdEntityStatus:         readEntityStatus,
	packetIdEntityMetadata:       readEntityMetadata,
	packetIdEntityEffect:         readEntityEffect,
	packetIdEntityRemoveEffect:   readEntityRemoveEffect,
	packetIdPlayerExperience:     readPlayerExperience,
	packetIdPreChunk:             readPreChunk,
	packetIdMapChunk:             readMapChunk,
	packetIdBlockChangeMulti:     readBlockChangeMulti,
	packetIdBlockChange:          readBlockChange,
	packetIdNoteBlockPlay:        readNoteBlockPlay,
	packetIdExplosion:            readExplosion,
	packetIdSoundEffect:          readSoundEffect,
	packetIdState:                readState,
	packetIdWeather:              readWeather,
	packetIdWindowOpen:           readWindowOpen,
	packetIdWindowSetSlot:        readWindowSetSlot,
	packetIdWindowItems:          readWindowItems,
	packetIdWindowProgressBar:    readWindowProgressBar,
	packetIdQuickbarSlotUpdate:   readQuickbarSlotUpdate,
	packetIdItemData:             readItemData,
	packetIdIncrementStatistic:   readIncrementStatistic,
}

// A server should call this to receive a single packet from a client. It will
// block until a packet was successfully handled, or there was an error.
func ServerReadPacket(reader io.Reader, handler IServerPacketHandler) os.Error {
	var packetId byte

	if err := binary.Read(reader, binary.BigEndian, &packetId); err != nil {
		return err
	}

	if commonFn, ok := commonReadFns[packetId]; ok {
		return commonFn(reader, handler)
	}

	if serverFn, ok := serverReadFns[packetId]; ok {
		return serverFn(reader, handler)
	}

	return os.NewError(fmt.Sprintf("unhandled packet type %#x", packetId))
}

// A client should call this to receive a single packet from a client. It will
// block until a packet was successfully handled, or there was an error.
func ClientReadPacket(reader io.Reader, handler IClientPacketHandler) os.Error {
	var packetId byte

	if err := binary.Read(reader, binary.BigEndian, &packetId); err != nil {
		return err
	}

	if commonFn, ok := commonReadFns[packetId]; ok {
		return commonFn(reader, handler)
	}

	if clientFn, ok := clientReadFns[packetId]; ok {
		return clientFn(reader, handler)
	}

	return os.NewError(fmt.Sprintf("unhandled packet type %#x", packetId))
}
