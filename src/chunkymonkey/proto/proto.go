package proto

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"regexp"
	"unicode/utf8"

	. "chunkymonkey/types"
)

const (
	// Currently only this protocol version is supported.
	protocolVersion = 17

	maxUcs2Char  = 0xffff
	ucs2ReplChar = 0xfffd

	// Packet type IDs
	PacketIdKeepAlive            = 0x00
	PacketIdLogin                = 0x01
	PacketIdHandshake            = 0x02
	PacketIdChatMessage          = 0x03
	PacketIdTimeUpdate           = 0x04
	PacketIdEntityEquipment      = 0x05
	PacketIdSpawnPosition        = 0x06
	PacketIdUseEntity            = 0x07
	PacketIdUpdateHealth         = 0x08
	PacketIdRespawn              = 0x09
	PacketIdPlayer               = 0x0a
	PacketIdPlayerPosition       = 0x0b
	PacketIdPlayerLook           = 0x0c
	PacketIdPlayerPositionLook   = 0x0d
	PacketIdPlayerBlockHit       = 0x0e
	PacketIdPlayerBlockInteract  = 0x0f
	PacketIdHoldingChange        = 0x10
	PacketIdBedUse               = 0x11
	PacketIdEntityAnimation      = 0x12
	PacketIdEntityAction         = 0x13
	PacketIdNamedEntitySpawn     = 0x14
	PacketIdItemSpawn            = 0x15
	PacketIdItemCollect          = 0x16
	PacketIdObjectSpawn          = 0x17
	PacketIdEntitySpawn          = 0x18
	PacketIdPaintingSpawn        = 0x19
	PacketIdExperienceOrb        = 0x1a
	PacketIdEntityVelocity       = 0x1c
	PacketIdEntityDestroy        = 0x1d
	PacketIdEntity               = 0x1e
	PacketIdEntityRelMove        = 0x1f
	PacketIdEntityLook           = 0x20
	PacketIdEntityLookAndRelMove = 0x21
	PacketIdEntityTeleport       = 0x22
	PacketIdEntityStatus         = 0x26
	PacketIdEntityMetadata       = 0x28
	PacketIdEntityEffect         = 0x29
	PacketIdEntityRemoveEffect   = 0x2a
	PacketIdPlayerExperience     = 0x2b
	PacketIdPreChunk             = 0x32
	PacketIdMapChunk             = 0x33
	PacketIdBlockChangeMulti     = 0x34
	PacketIdBlockChange          = 0x35
	PacketIdNoteBlockPlay        = 0x36
	PacketIdExplosion            = 0x3c
	PacketIdSoundEffect          = 0x3d
	PacketIdState                = 0x46
	PacketIdWeather              = 0x47
	PacketIdWindowOpen           = 0x64
	PacketIdWindowClose          = 0x65
	PacketIdWindowClick          = 0x66
	PacketIdWindowSetSlot        = 0x67
	PacketIdWindowItems          = 0x68
	PacketIdWindowProgressBar    = 0x69
	PacketIdWindowTransaction    = 0x6a
	PacketIdQuickbarSlotUpdate   = 0x6b
	PacketIdSignUpdate           = 0x82
	PacketIdItemData             = 0x83
	PacketIdIncrementStatistic   = 0xc8
	PacketIdUserListItem         = 0xc9
	PacketIdServerListPing       = 0xfe
	PacketIdDisconnect           = 0xff
)

type UnexpectedPacketIdError byte

func (err UnexpectedPacketIdError) String() string {
	return fmt.Sprintf("unexpected packet ID: 0x%02x", byte(err))
}

type UnknownPacketIdError byte

func (err UnknownPacketIdError) String() string {
	return fmt.Sprintf("unknown packet ID: 0x%02x", byte(err))
}

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
	PacketServerLogin(username string)
	PacketServerHandshake(username string)
	PacketPlayer(onGround bool)
	PacketHoldingChange(slotId SlotId)
	PacketWindowClose(windowId WindowId)
	PacketWindowClick(windowId WindowId, slot SlotId, rightClick bool, txId TxId, shiftClick bool, expectedSlot *WindowSlot)
	PacketServerListPing()
}

// Clients to the protocol must implement this interface to receive packets
type IClientPacketHandler interface {
	IPacketHandler
	PacketClientLogin(entityId EntityId, mapSeed RandomSeed, serverMode int32, dimension DimensionId, unknown int8, worldHeight, maxPlayers byte)
	PacketClientHandshake(serverId string)
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

func readString16(reader io.Reader) (s string, err error) {
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

func writeString16(writer io.Writer, s string) (err error) {
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

func writeEntityMetadataField(writer io.Writer, data []EntityMetadata) (err error) {
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
func readEntityMetadataField(reader io.Reader) (data []EntityMetadata, err error) {
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

// PacketIdKeepAlive

func WriteKeepAlive(writer io.Writer, id int32) error {
	var packet = struct {
		PacketId byte
		Id       int32
	}{
		PacketIdKeepAlive,
		id,
	}
	return binary.Write(writer, binary.BigEndian, &packet)
}

func readKeepAlive(reader io.Reader, handler IPacketHandler) (err error) {
	var id int32
	if err = binary.Read(reader, binary.BigEndian, &id); err != nil {
		return
	}
	handler.PacketKeepAlive(id)
	return
}

// PacketIdLogin

func commonWriteLogin(writer io.Writer, versionOrEntityId int32, str string, mapSeed RandomSeed, serverMode int32, dimension DimensionId, difficulty GameDifficulty, worldHeight, maxPlayers byte) (err error) {
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
		Difficulty  GameDifficulty
		WorldHeight byte
		MaxPlayers  byte
	}{
		mapSeed,
		serverMode,
		dimension,
		difficulty,
		worldHeight,
		maxPlayers,
	}
	return binary.Write(writer, binary.BigEndian, &packetEnd)
}

func ServerWriteLogin(writer io.Writer, entityId EntityId, mapSeed RandomSeed, serverMode int32, dimension DimensionId, difficulty GameDifficulty, worldHeight, maxPlayers byte) (err error) {
	if err = binary.Write(writer, binary.BigEndian, byte(PacketIdLogin)); err != nil {
		return
	}

	return commonWriteLogin(writer, int32(entityId), "", mapSeed, serverMode, dimension, difficulty, worldHeight, maxPlayers)
}

func ClientWriteLogin(writer io.Writer, username, password string) (err error) {
	if err = binary.Write(writer, binary.BigEndian, byte(PacketIdLogin)); err != nil {
		return
	}

	return commonWriteLogin(writer, protocolVersion, username, 0, 0, 0, 0, 0, 0)
}

func commonReadLogin(reader io.Reader) (versionOrEntityId int32, str string, mapSeed RandomSeed, serverMode int32, dimension DimensionId, unknown int8, worldHeight, maxPlayers byte, err error) {
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

func serverReadLogin(reader io.Reader, handler IServerPacketHandler) (err error) {
	version, username, _, _, _, _, _, _, err := commonReadLogin(reader)
	if err != nil {
		return
	}

	if version != protocolVersion {
		err = fmt.Errorf("serverLogin: unsupported protocol version %#x", version)
		return
	}

	handler.PacketServerLogin(username)

	return
}

func clientReadLogin(reader io.Reader, handler IClientPacketHandler) (err error) {
	entityId, _, mapSeed, serverMode, dimension, unknown, worldHeight, maxPlayers, err := commonReadLogin(reader)
	if err != nil {
		return
	}

	handler.PacketClientLogin(EntityId(entityId), mapSeed, serverMode, dimension, unknown, worldHeight, maxPlayers)

	return
}

// PacketIdHandshake

func ServerWriteHandshake(writer io.Writer, reply string) (err error) {
	if err = binary.Write(writer, binary.BigEndian, byte(PacketIdHandshake)); err != nil {
		return
	}

	return writeString16(writer, reply)
}

func serverReadHandshake(reader io.Reader, handler IServerPacketHandler) (err error) {
	var username string
	if username, err = readString16(reader); err != nil {
		return
	}

	handler.PacketServerHandshake(username)

	return
}

func clientReadHandshake(reader io.Reader, handler IClientPacketHandler) (err error) {
	var serverId string
	if serverId, err = readString16(reader); err != nil {
		return
	}

	handler.PacketClientHandshake(serverId)

	return
}

// PacketIdChatMessage

func WriteChatMessage(writer io.Writer, message string) (err error) {
	// Check chat message against illegal chars
	if checkChatMessageRegexp.MatchString(message) {
		// Check suffix against color tags eg. "This is a message §0"
		if checkColorsRegexp.MatchString(message) {
			// Found a color tag at the end
			return colorTagEndErr
		} else {
			err = binary.Write(writer, binary.BigEndian, byte(PacketIdChatMessage))
			if err != nil {
				return
			}
			err = writeString16(writer, message)
			return
		}
	}
	return illegalCharErr
}

func readChatMessage(reader io.Reader, handler IPacketHandler) (err error) {
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

// PacketIdTimeUpdate

func ServerWriteTimeUpdate(writer io.Writer, time Ticks) error {
	var packet = struct {
		PacketId byte
		Time     Ticks
	}{
		PacketIdTimeUpdate,
		time,
	}
	return binary.Write(writer, binary.BigEndian, &packet)
}

func readTimeUpdate(reader io.Reader, handler IClientPacketHandler) (err error) {
	var time Ticks

	err = binary.Read(reader, binary.BigEndian, &time)
	if err != nil {
		return
	}

	handler.PacketTimeUpdate(time)
	return
}

// PacketIdEntityEquipment

func WriteEntityEquipment(writer io.Writer, entityId EntityId, slot SlotId, itemTypeId ItemTypeId, data ItemData) (err error) {
	var packet = struct {
		PacketId   byte
		EntityId   EntityId
		Slot       SlotId
		ItemTypeId ItemTypeId
		Data       ItemData
	}{
		PacketIdEntityEquipment,
		entityId,
		slot,
		itemTypeId,
		data,
	}

	return binary.Write(writer, binary.BigEndian, &packet)
}

func readEntityEquipment(reader io.Reader, handler IClientPacketHandler) (err error) {
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

// PacketIdSpawnPosition

func WriteSpawnPosition(writer io.Writer, position *BlockXyz) error {
	var packet = struct {
		PacketId byte
		X        BlockCoord
		Y        int32
		Z        BlockCoord
	}{
		PacketIdSpawnPosition,
		position.X,
		int32(position.Y),
		position.Z,
	}
	return binary.Write(writer, binary.BigEndian, &packet)
}

func readSpawnPosition(reader io.Reader, handler IClientPacketHandler) (err error) {
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

// PacketIdUseEntity

func WriteUseEntity(writer io.Writer, user EntityId, target EntityId, leftClick bool) (err error) {
	var packet = struct {
		PacketId  byte
		User      EntityId
		Target    EntityId
		LeftClick byte
	}{
		PacketIdUseEntity,
		user,
		target,
		boolToByte(leftClick),
	}

	return binary.Write(writer, binary.BigEndian, &packet)
}

func readUseEntity(reader io.Reader, handler IPacketHandler) (err error) {
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

// PacketIdUpdateHealth

func WriteUpdateHealth(writer io.Writer, health Health, food FoodUnits, foodSaturation float32) (err error) {
	var packet = struct {
		PacketId       byte
		Health         Health
		Food           FoodUnits
		FoodSaturation float32
	}{
		PacketIdUpdateHealth,
		health,
		food,
		foodSaturation,
	}

	return binary.Write(writer, binary.BigEndian, &packet)
}

func readUpdateHealth(reader io.Reader, handler IClientPacketHandler) (err error) {
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

// PacketIdRespawn

func WriteRespawn(writer io.Writer, dimension DimensionId, unknown int8, gameType GameType, worldHeight int16, mapSeed RandomSeed) error {
	var packet = struct {
		PacketId    byte
		Dimension   DimensionId
		Unknown     int8
		GameType    GameType
		WorldHeight int16
		MapSeed     RandomSeed
	}{
		PacketIdRespawn,
		dimension,
		unknown,
		gameType,
		worldHeight,
		mapSeed,
	}

	return binary.Write(writer, binary.BigEndian, &packet)
}

func readRespawn(reader io.Reader, handler IPacketHandler) (err error) {
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

// PacketIdPlayer

func WritePlayer(writer io.Writer, onGround bool) (err error) {
	var packet = struct {
		PacketId byte
		OnGround byte
	}{
		PacketIdPlayer,
		boolToByte(onGround),
	}

	return binary.Write(writer, binary.BigEndian, &packet)
}

func readPlayer(reader io.Reader, handler IServerPacketHandler) (err error) {
	var onGround byte

	if err = binary.Read(reader, binary.BigEndian, &onGround); err != nil {
		return
	}

	handler.PacketPlayer(byteToBool(onGround))

	return
}

// PacketIdPlayerPosition

func WritePlayerPosition(writer io.Writer, position *AbsXyz, stance AbsCoord, onGround bool) error {
	var packet = struct {
		PacketId byte
		X        AbsCoord
		Y        AbsCoord
		Stance   AbsCoord
		Z        AbsCoord
		OnGround byte
	}{
		PacketIdPlayerPosition,
		position.X,
		position.Y,
		stance,
		position.Z,
		boolToByte(onGround),
	}
	return binary.Write(writer, binary.BigEndian, &packet)
}

func readPlayerPosition(reader io.Reader, handler IPacketHandler) (err error) {
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

// PacketIdPlayerLook

func WritePlayerLook(writer io.Writer, look *LookDegrees, onGround bool) (err error) {
	var packet = struct {
		PacketId byte
		Yaw      AngleDegrees
		Pitch    AngleDegrees
		OnGround byte
	}{
		PacketIdPlayerLook,
		look.Yaw, look.Pitch,
		boolToByte(onGround),
	}

	return binary.Write(writer, binary.BigEndian, &packet)
}

func readPlayerLook(reader io.Reader, handler IPacketHandler) (err error) {
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

// PacketIdPlayerPositionLook

func writePlayerPositionLookCommon(writer io.Writer, x, y1, y2, z AbsCoord, look *LookDegrees, onGround bool) (err error) {
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
		PacketIdPlayerPositionLook,
		x, y1, y2, z,
		look.Yaw, look.Pitch,
		boolToByte(onGround),
	}

	return binary.Write(writer, binary.BigEndian, &packet)
}

func ClientWritePlayerPositionLook(writer io.Writer, position *AbsXyz, stance AbsCoord, look *LookDegrees, onGround bool) (err error) {
	return writePlayerPositionLookCommon(
		writer,
		position.X, position.Y, stance, position.Z,
		look,
		onGround)
}

func ServerWritePlayerPositionLook(writer io.Writer, position *AbsXyz, stance AbsCoord, look *LookDegrees, onGround bool) (err error) {
	return writePlayerPositionLookCommon(
		writer,
		position.X, stance, position.Y, position.Z,
		look,
		onGround)
}

func commonReadPlayerPositionLook(reader io.Reader) (x, y1, y2, z AbsCoord, look *LookDegrees, onGround bool, err error) {
	var packet struct {
		X        AbsCoord
		Y1       AbsCoord
		Y2       AbsCoord
		Z        AbsCoord
		Yaw      AngleDegrees
		Pitch    AngleDegrees
		OnGround byte
	}

	if err = binary.Read(reader, binary.BigEndian, &packet); err != nil {
		return
	}

	return packet.X, packet.Y1, packet.Y2, packet.Z, &LookDegrees{packet.Yaw, packet.Pitch}, byteToBool(packet.OnGround), nil
}

func clientReadPlayerPositionLook(reader io.Reader, handler IClientPacketHandler) (err error) {
	x, y1, y2, z, look, onGround, err := commonReadPlayerPositionLook(reader)
	if err != nil {
		return
	}

	handler.PacketPlayerPosition(&AbsXyz{x, y2, z}, y1, onGround)
	handler.PacketPlayerLook(look, onGround)

	return
}

func serverReadPlayerPositionLook(reader io.Reader, handler IServerPacketHandler) (err error) {
	x, y1, y2, z, look, onGround, err := commonReadPlayerPositionLook(reader)
	if err != nil {
		return
	}

	handler.PacketPlayerPosition(&AbsXyz{x, y1, z}, y2, onGround)
	handler.PacketPlayerLook(look, onGround)

	return
}

// PacketIdPlayerBlockHit

func WritePlayerBlockHit(writer io.Writer, status DigStatus, blockLoc *BlockXyz, face Face) (err error) {
	var packet = struct {
		PacketId byte
		Status   DigStatus
		X        BlockCoord
		Y        BlockYCoord
		Z        BlockCoord
		Face     Face
	}{
		PacketIdPlayerBlockHit,
		status,
		blockLoc.X, blockLoc.Y, blockLoc.Z,
		face,
	}

	return binary.Write(writer, binary.BigEndian, &packet)
}

func readPlayerBlockHit(reader io.Reader, handler IPacketHandler) (err error) {
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

// PacketIdPlayerBlockInteract

func WritePlayerBlockInteract(writer io.Writer, itemTypeId ItemTypeId, blockLoc *BlockXyz, face Face, amount ItemCount, data ItemData) (err error) {
	var packet = struct {
		PacketId   byte
		X          BlockCoord
		Y          BlockYCoord
		Z          BlockCoord
		Face       Face
		ItemTypeId ItemTypeId
	}{
		PacketIdPlayerBlockInteract,
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

func readPlayerBlockInteract(reader io.Reader, handler IPacketHandler) (err error) {
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

// PacketIdHoldingChange

func WriteHoldingChange(writer io.Writer, slotId SlotId) (err error) {
	var packet = struct {
		PacketId byte
		SlotId   SlotId
	}{
		PacketIdHoldingChange,
		slotId,
	}

	return binary.Write(writer, binary.BigEndian, &packet)
}

func readHoldingChange(reader io.Reader, handler IServerPacketHandler) (err error) {
	var slotId SlotId

	if err = binary.Read(reader, binary.BigEndian, &slotId); err != nil {
		return
	}

	handler.PacketHoldingChange(slotId)

	return
}

// PacketIdBedUse

func WriteBedUse(writer io.Writer, flag bool, bedLoc *BlockXyz) (err error) {
	var packet = struct {
		PacketId byte
		Flag     byte
		X        BlockCoord
		Y        BlockYCoord
		Z        BlockCoord
	}{
		PacketIdBedUse,
		boolToByte(flag),
		bedLoc.X,
		bedLoc.Y,
		bedLoc.Z,
	}

	return binary.Write(writer, binary.BigEndian, &packet)
}

func readBedUse(reader io.Reader, handler IClientPacketHandler) (err error) {
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

// PacketIdEntityAnimation

func WriteEntityAnimation(writer io.Writer, entityId EntityId, animation EntityAnimation) (err error) {
	var packet = struct {
		PacketId  byte
		EntityId  EntityId
		Animation EntityAnimation
	}{
		PacketIdEntityAnimation,
		entityId,
		animation,
	}

	return binary.Write(writer, binary.BigEndian, &packet)
}

func readEntityAnimation(reader io.Reader, handler IPacketHandler) (err error) {
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

// PacketIdEntityAction

func WriteEntityAction(writer io.Writer, entityId EntityId, action EntityAction) (err error) {
	var packet = struct {
		PacketId byte
		EntityId EntityId
		Action   EntityAction
	}{
		PacketIdEntityAction,
		entityId,
		action,
	}

	return binary.Write(writer, binary.BigEndian, &packet)
}

func readEntityAction(reader io.Reader, handler IPacketHandler) (err error) {
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

// PacketIdNamedEntitySpawn

func WriteNamedEntitySpawn(writer io.Writer, entityId EntityId, name string, position *AbsIntXyz, look *LookBytes, currentItem ItemTypeId) (err error) {
	var packetStart = struct {
		PacketId byte
		EntityId EntityId
	}{
		PacketIdNamedEntitySpawn,
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

func readNamedEntitySpawn(reader io.Reader, handler IClientPacketHandler) (err error) {
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

// PacketIdItemSpawn

func WriteItemSpawn(writer io.Writer, entityId EntityId, itemTypeId ItemTypeId, amount ItemCount, data ItemData, position *AbsIntXyz, orientation *OrientationBytes) error {
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
		PacketIdItemSpawn,
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

func readItemSpawn(reader io.Reader, handler IClientPacketHandler) (err error) {
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

// PacketIdItemCollect

func WriteItemCollect(writer io.Writer, collectedItem EntityId, collector EntityId) (err error) {
	var packet = struct {
		PacketId      byte
		CollectedItem EntityId
		Collector     EntityId
	}{
		PacketIdItemCollect,
		collectedItem,
		collector,
	}

	return binary.Write(writer, binary.BigEndian, &packet)
}

func readItemCollect(reader io.Reader, handler IClientPacketHandler) (err error) {
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

// PacketIdObjectSpawn

func WriteObjectSpawn(writer io.Writer, entityId EntityId, objType ObjTypeId, position *AbsIntXyz, objectData *ObjectData) (err error) {
	var packet = struct {
		PacketId byte
		EntityId EntityId
		ObjType  ObjTypeId
		X        AbsIntCoord
		Y        AbsIntCoord
		Z        AbsIntCoord
	}{
		PacketIdObjectSpawn,
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

func readObjectSpawn(reader io.Reader, handler IClientPacketHandler) (err error) {
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

// PacketIdEntitySpawn

func WriteEntitySpawn(writer io.Writer, entityId EntityId, mobType EntityMobType, position *AbsIntXyz, look *LookBytes, data []EntityMetadata) (err error) {
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
		PacketIdEntitySpawn,
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

func readEntitySpawn(reader io.Reader, handler IClientPacketHandler) (err error) {
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

// PacketIdPaintingSpawn

func WritePaintingSpawn(writer io.Writer, entityId EntityId, title string, position *BlockXyz, paintingType PaintingTypeId) (err error) {
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

func readPaintingSpawn(reader io.Reader, handler IClientPacketHandler) (err error) {
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

// PacketIdExperienceOrb

func WriteExperienceOrb(writer io.Writer, entityId EntityId, position AbsIntXyz, count int16) (err error) {
	var packet = struct {
		PacketId byte
		EntityId EntityId
		X, Y, Z  AbsIntCoord
		Count    int16
	}{
		PacketIdExperienceOrb,
		entityId,
		position.X, position.Y, position.Z,
		count,
	}

	return binary.Write(writer, binary.BigEndian, &packet)
}

func readExperienceOrb(reader io.Reader, handler IClientPacketHandler) (err error) {
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

// PacketIdEntityVelocity

func WriteEntityVelocity(writer io.Writer, entityId EntityId, velocity *Velocity) (err error) {
	var packet = struct {
		packetId byte
		EntityId EntityId
		X, Y, Z  VelocityComponent
	}{
		PacketIdEntityVelocity,
		entityId,
		velocity.X,
		velocity.Y,
		velocity.Z,
	}

	return binary.Write(writer, binary.BigEndian, &packet)
}

func readEntityVelocity(reader io.Reader, handler IClientPacketHandler) (err error) {
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

// PacketIdEntityDestroy

func WriteEntityDestroy(writer io.Writer, entityId EntityId) error {
	var packet = struct {
		PacketId byte
		EntityId EntityId
	}{
		PacketIdEntityDestroy,
		entityId,
	}
	return binary.Write(writer, binary.BigEndian, &packet)
}

func readEntityDestroy(reader io.Reader, handler IClientPacketHandler) (err error) {
	var entityId EntityId

	err = binary.Read(reader, binary.BigEndian, &entityId)
	if err != nil {
		return
	}

	handler.PacketEntityDestroy(entityId)

	return
}

// PacketIdEntity

func WriteEntity(writer io.Writer, entityId EntityId) (err error) {
	var packet = struct {
		PacketId byte
		EntityId EntityId
	}{
		PacketIdEntity,
		entityId,
	}

	return binary.Write(writer, binary.BigEndian, &packet)
}

func readEntity(reader io.Reader, handler IClientPacketHandler) (err error) {
	var entityId EntityId

	err = binary.Read(reader, binary.BigEndian, &entityId)
	if err != nil {
		return
	}

	handler.PacketEntity(entityId)

	return
}

// PacketIdEntityRelMove

func WriteEntityRelMove(writer io.Writer, entityId EntityId, movement *RelMove) (err error) {
	var packet = struct {
		PacketId byte
		EntityId EntityId
		X, Y, Z  RelMoveCoord
	}{
		PacketIdEntityRelMove,
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

func readEntityRelMove(reader io.Reader, handler IClientPacketHandler) (err error) {
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

// PacketIdEntityLook

func WriteEntityLook(writer io.Writer, entityId EntityId, look *LookBytes) error {
	var packet = struct {
		PacketId byte
		EntityId EntityId
		Yaw      AngleBytes
		Pitch    AngleBytes
	}{
		PacketIdEntityLook,
		entityId,
		look.Yaw,
		look.Pitch,
	}
	return binary.Write(writer, binary.BigEndian, &packet)
}

func readEntityLook(reader io.Reader, handler IClientPacketHandler) (err error) {
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

// PacketIdEntityLookAndRelMove

func WriteEntityLookAndRelMove(writer io.Writer, entityId EntityId, movement *RelMove, look *LookBytes) (err error) {
	var packet = struct {
		PacketId byte
		EntityId EntityId
		X, Y, Z  RelMoveCoord
		Yaw      AngleBytes
		Pitch    AngleBytes
	}{
		PacketIdEntityLookAndRelMove,
		entityId,
		movement.X, movement.Y, movement.Z,
		look.Yaw, look.Pitch,
	}

	return binary.Write(writer, binary.BigEndian, &packet)
}

func readEntityLookAndRelMove(reader io.Reader, handler IClientPacketHandler) (err error) {
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

// PacketIdEntityTeleport

func WriteEntityTeleport(writer io.Writer, entityId EntityId, position *AbsIntXyz, look *LookBytes) error {
	var packet = struct {
		PacketId byte
		EntityId EntityId
		X        AbsIntCoord
		Y        AbsIntCoord
		Z        AbsIntCoord
		Yaw      AngleBytes
		Pitch    AngleBytes
	}{
		PacketIdEntityTeleport,
		entityId,
		position.X,
		position.Y,
		position.Z,
		look.Yaw,
		look.Pitch,
	}
	return binary.Write(writer, binary.BigEndian, &packet)
}

func readEntityTeleport(reader io.Reader, handler IClientPacketHandler) (err error) {
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

// PacketIdEntityStatus

func WriteEntityStatus(writer io.Writer, entityId EntityId, status EntityStatus) (err error) {
	var packet = struct {
		PacketId byte
		EntityId EntityId
		Status   EntityStatus
	}{
		PacketIdEntityStatus,
		entityId,
		status,
	}

	return binary.Write(writer, binary.BigEndian, &packet)
}

func readEntityStatus(reader io.Reader, handler IClientPacketHandler) (err error) {
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

// PacketIdEntityMetadata

func WriteEntityMetadata(writer io.Writer, entityId EntityId, data []EntityMetadata) (err error) {
	var packet = struct {
		PacketId byte
		EntityId EntityId
	}{
		PacketIdEntityMetadata,
		entityId,
	}

	if err = binary.Write(writer, binary.BigEndian, &packet); err != nil {
		return
	}

	return writeEntityMetadataField(writer, data)
}

func readEntityMetadata(reader io.Reader, handler IClientPacketHandler) (err error) {
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

// PacketIdEntityEffect

func WriteEntityEffect(writer io.Writer, entityId EntityId, effect EntityEffect, value int8, duration int16) (err error) {
	var packet = struct {
		PacketId byte
		EntityId EntityId
		Effect   EntityEffect
		Value    int8
		Duration int16
	}{
		PacketIdEntityEffect,
		entityId,
		effect,
		value,
		duration,
	}

	return binary.Write(writer, binary.BigEndian, &packet)
}

func readEntityEffect(reader io.Reader, handler IClientPacketHandler) (err error) {
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

// PacketIdEntityRemoveEffect

func WriteEntityRemoveEffect(writer io.Writer, entityId EntityId, effect EntityEffect) (err error) {
	var packet = struct {
		PacketId byte
		EntityId EntityId
		Effect   EntityEffect
	}{
		PacketIdEntityEffect,
		entityId,
		effect,
	}

	return binary.Write(writer, binary.BigEndian, &packet)
}

func readEntityRemoveEffect(reader io.Reader, handler IClientPacketHandler) (err error) {
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

// PacketIdPlayerExperience

func WritePlayerExperience(writer io.Writer, experience, level int8, totalExperience int16) (err error) {
	var packet = struct {
		PacketId        byte
		Experience      int8
		Level           int8
		TotalExperience int16
	}{
		PacketIdPlayerExperience,
		experience,
		level,
		totalExperience,
	}

	return binary.Write(writer, binary.BigEndian, &packet)
}

func readPlayerExperience(reader io.Reader, handler IClientPacketHandler) (err error) {
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

// PacketIdPreChunk

func WritePreChunk(writer io.Writer, chunkLoc *ChunkXz, mode ChunkLoadMode) error {
	var packet = struct {
		PacketId byte
		X        ChunkCoord
		Z        ChunkCoord
		Mode     ChunkLoadMode
	}{
		PacketIdPreChunk,
		chunkLoc.X,
		chunkLoc.Z,
		mode,
	}
	return binary.Write(writer, binary.BigEndian, &packet)
}

func readPreChunk(reader io.Reader, handler IClientPacketHandler) (err error) {
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

// PacketIdMapChunk

func WriteMapChunk(writer io.Writer, chunkLoc *ChunkXz, blocks, blockData, blockLight, skyLight []byte) (err error) {
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
		PacketIdMapChunk,
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

func readMapChunk(reader io.Reader, handler IClientPacketHandler) (err error) {
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

// PacketIdBlockChangeMulti

func WriteBlockChangeMulti(writer io.Writer, chunkLoc *ChunkXz, blockCoords []SubChunkXyz, blockTypes []BlockId, blockMetaData []byte) (err error) {
	// NOTE that we don't yet check that blockCoords, blockTypes and
	// blockMetaData are of the same length.

	var packet = struct {
		PacketId byte
		ChunkX   ChunkCoord
		ChunkZ   ChunkCoord
		Count    int16
	}{
		PacketIdBlockChangeMulti,
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

func readBlockChangeMulti(reader io.Reader, handler IClientPacketHandler) (err error) {
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

// PacketIdBlockChange

func WriteBlockChange(writer io.Writer, blockLoc *BlockXyz, blockType BlockId, blockMetaData byte) (err error) {
	var packet = struct {
		PacketId      byte
		X             BlockCoord
		Y             BlockYCoord
		Z             BlockCoord
		BlockType     BlockId
		BlockMetadata byte
	}{
		PacketIdBlockChange,
		blockLoc.X,
		blockLoc.Y,
		blockLoc.Z,
		blockType,
		blockMetaData,
	}
	err = binary.Write(writer, binary.BigEndian, &packet)
	return
}

func readBlockChange(reader io.Reader, handler IClientPacketHandler) (err error) {
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

// PacketIdNoteBlockPlay

func WriteNoteBlockPlay(writer io.Writer, position *BlockXyz, instrument InstrumentId, pitch NotePitch) (err error) {
	var packet = struct {
		PacketId   byte
		X          BlockCoord
		Y          BlockYCoord
		Z          BlockCoord
		Instrument InstrumentId
		Pitch      NotePitch
	}{
		PacketIdNoteBlockPlay,
		position.X, position.Y, position.Z,
		instrument,
		pitch,
	}

	return binary.Write(writer, binary.BigEndian, &packet)
}

func readNoteBlockPlay(reader io.Reader, handler IClientPacketHandler) (err error) {
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

// PacketIdExplosion

// TODO introduce better types for ExplosionOffsetXyz and the floats in the
// packet structure when the packet is better understood.

type ExplosionOffsetXyz struct {
	X, Y, Z int8
}

func WriteExplosion(writer io.Writer, position *AbsXyz, power float32, blockOffsets []ExplosionOffsetXyz) (err error) {
	var packet = struct {
		PacketId byte
		// NOTE AbsCoord is just a guess for now
		X, Y, Z AbsCoord
		// NOTE Power isn't known to be a good name for this field
		Power     float32
		NumBlocks int32
	}{
		PacketIdExplosion,
		position.X, position.Y, position.Z,
		power,
		int32(len(blockOffsets)),
	}

	if err = binary.Write(writer, binary.BigEndian, &packet); err != nil {
		return
	}

	return binary.Write(writer, binary.BigEndian, blockOffsets)
}

func readExplosion(reader io.Reader, handler IClientPacketHandler) (err error) {
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

// PacketIdSoundEffect

func WriteSoundEffect(writer io.Writer, sound SoundEffect, position BlockXyz, data int32) (err error) {
	var packet = struct {
		PacketId byte
		Sound    SoundEffect
		X        BlockCoord
		Y        BlockYCoord
		Z        BlockCoord
		Data     int32
	}{
		PacketIdSoundEffect,
		sound,
		position.X, position.Y, position.Z,
		data,
	}

	return binary.Write(writer, binary.BigEndian, &packet)
}

func readSoundEffect(reader io.Reader, handler IClientPacketHandler) (err error) {
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

// PacketIdState

func WriteState(writer io.Writer, reason, gameMode byte) (err error) {
	var packet = struct {
		PacketId byte
		Reason   byte
		GameMode byte
	}{
		PacketIdState,
		reason,
		gameMode,
	}

	return binary.Write(writer, binary.BigEndian, &packet)
}

func readState(reader io.Reader, handler IClientPacketHandler) (err error) {
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

// PacketIdWeather

func WriteWeather(writer io.Writer, entityId EntityId, raining bool, position *AbsIntXyz) (err error) {
	var packet = struct {
		PacketId byte
		EntityId EntityId
		Raining  byte
		X, Y, Z  AbsIntCoord
	}{
		PacketIdWeather,
		entityId,
		boolToByte(raining),
		position.X, position.Y, position.Z,
	}

	return binary.Write(writer, binary.BigEndian, &packet)
}

func readWeather(reader io.Reader, handler IClientPacketHandler) (err error) {
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

// PacketIdWindowOpen

func WriteWindowOpen(writer io.Writer, windowId WindowId, invTypeId InvTypeId, windowTitle string, numSlots byte) (err error) {
	var packet = struct {
		PacketId  byte
		WindowId  WindowId
		InvTypeId InvTypeId
	}{
		PacketIdWindowOpen,
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

func readWindowOpen(reader io.Reader, handler IClientPacketHandler) (err error) {
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

// PacketIdWindowClose

func WriteWindowClose(writer io.Writer, windowId WindowId) (err error) {
	var packet = struct {
		PacketId byte
		WindowId WindowId
	}{
		PacketIdWindowClose,
		windowId,
	}

	if err = binary.Write(writer, binary.BigEndian, &packet); err != nil {
		return
	}

	return
}

func readWindowClose(reader io.Reader, handler IServerPacketHandler) (err error) {
	var windowId WindowId

	if err = binary.Read(reader, binary.BigEndian, &windowId); err != nil {
		return
	}

	handler.PacketWindowClose(windowId)

	return
}

// PacketIdWindowClick

func WriteWindowClick(writer io.Writer, windowId WindowId, slot SlotId, rightClick bool, txId TxId, shiftClick bool, expectedSlot WindowSlot) (err error) {
	var packet = struct {
		PacketId   byte
		WindowId   WindowId
		Slot       SlotId
		RightClick byte
		TxId       TxId
		ShiftClick byte
		ItemTypeId ItemTypeId
	}{
		PacketIdWindowClick,
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

func readWindowClick(reader io.Reader, handler IServerPacketHandler) (err error) {
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

// PacketIdWindowSetSlot

func WriteWindowSetSlot(writer io.Writer, windowId WindowId, slot SlotId, itemTypeId ItemTypeId, amount ItemCount, data ItemData) (err error) {
	var packet = struct {
		PacketId byte
		WindowId WindowId
		Slot     SlotId
	}{
		PacketIdWindowSetSlot,
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

func readWindowSetSlot(reader io.Reader, handler IClientPacketHandler) (err error) {
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

// PacketIdWindowItems

func WriteWindowItems(writer io.Writer, windowId WindowId, items []WindowSlot) (err error) {
	var packet = struct {
		PacketId byte
		WindowId WindowId
		Count    int16
	}{
		PacketIdWindowItems,
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

func readWindowItems(reader io.Reader, handler IClientPacketHandler) (err error) {
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

// PacketIdWindowProgressBar

func WriteWindowProgressBar(writer io.Writer, windowId WindowId, prgBarId PrgBarId, value PrgBarValue) error {
	var packet = struct {
		PacketId byte
		WindowId WindowId
		PrgBarId PrgBarId
		Value    PrgBarValue
	}{
		PacketIdWindowProgressBar,
		windowId,
		prgBarId,
		value,
	}

	return binary.Write(writer, binary.BigEndian, &packet)
}

func readWindowProgressBar(reader io.Reader, handler IClientPacketHandler) (err error) {
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

// PacketIdWindowTransaction

func WriteWindowTransaction(writer io.Writer, windowId WindowId, txId TxId, accepted bool) (err error) {
	var packet = struct {
		PacketId byte
		WindowId WindowId
		TxId     TxId
		Accepted byte
	}{
		PacketIdWindowTransaction,
		windowId,
		txId,
		boolToByte(accepted),
	}

	return binary.Write(writer, binary.BigEndian, &packet)
}

func readWindowTransaction(reader io.Reader, handler IPacketHandler) (err error) {
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

// PacketIdQuickbarSlotUpdate

func WriteQuickbarSlotUpdate(writer io.Writer, slot SlotId, itemId ItemTypeId, count ItemCount, data ItemData) (err error) {
	var packet = struct {
		PacketId byte
		Slot     SlotId
		ItemId   ItemTypeId
		Count    ItemCount
		Data     ItemData
	}{
		PacketIdQuickbarSlotUpdate,
		slot,
		itemId,
		count,
		data,
	}

	return binary.Write(writer, binary.BigEndian, &packet)
}

func readQuickbarSlotUpdate(reader io.Reader, handler IClientPacketHandler) (err error) {
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

// PacketIdSignUpdate

func WriteSignUpdate(writer io.Writer, position *BlockXyz, lines [4]string) (err error) {
	var packet = struct {
		PacketId byte
		X        BlockCoord
		Y        BlockYCoord
		Z        BlockCoord
	}{
		PacketIdSignUpdate,
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

func readSignUpdate(reader io.Reader, handler IPacketHandler) (err error) {
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

// PacketIdItemData

func WriteItemData(writer io.Writer, itemTypeId ItemTypeId, itemDataId ItemData, data []byte) (err error) {
	var packet = struct {
		PacketId   byte
		ItemTypeId ItemTypeId
		ItemDataId ItemData
		DataLength byte
	}{
		PacketIdItemData,
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

func readItemData(reader io.Reader, handler IClientPacketHandler) (err error) {
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

// PacketIdIncrementStatistic

func WriteIncrementStatistic(writer io.Writer, statisticId StatisticId, delta int8) (err error) {
	var packet = struct {
		PacketId    byte
		StatisticId StatisticId
		Delta       int8
	}{
		PacketIdIncrementStatistic,
		statisticId,
		delta,
	}
	return binary.Write(writer, binary.BigEndian, &packet)
}

func readIncrementStatistic(reader io.Reader, handler IClientPacketHandler) (err error) {
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

// PacketIdUserListItem

func WriteUserListItem(writer io.Writer, username string, online bool, pingMs int16) (err error) {
	if err = binary.Write(writer, binary.BigEndian, byte(PacketIdUserListItem)); err != nil {
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

func readUserListItem(reader io.Reader, handler IClientPacketHandler) (err error) {
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

// PacketIdServerListPing

func WriteServerListPing(writer io.Writer) (err error) {
	return binary.Write(writer, binary.BigEndian, byte(PacketIdServerListPing))
}

func readServerListPing(reader io.Reader, handler IServerPacketHandler) (err error) {
	handler.PacketServerListPing()
	return
}

// PacketIdDisconnect

func WriteDisconnect(writer io.Writer, reason string) (err error) {
	buf := &bytes.Buffer{}
	binary.Write(buf, binary.BigEndian, byte(PacketIdDisconnect))
	writeString16(buf, reason)
	_, err = writer.Write(buf.Bytes())
	return
}

func readDisconnect(reader io.Reader, handler IPacketHandler) (err error) {
	reason, err := readString16(reader)
	if err != nil {
		return
	}

	handler.PacketDisconnect(reason)
	return
}

// End of packet reader/writer functions

type commonPacketHandler func(io.Reader, IPacketHandler) error
type serverPacketHandler func(io.Reader, IServerPacketHandler) error
type clientPacketHandler func(io.Reader, IClientPacketHandler) error

type commonPacketReaderMap map[byte]commonPacketHandler
type serverPacketReaderMap map[byte]serverPacketHandler
type clientPacketReaderMap map[byte]clientPacketHandler

// Common packet mapping
var commonReadFns = commonPacketReaderMap{
	PacketIdKeepAlive:           readKeepAlive,
	PacketIdChatMessage:         readChatMessage,
	PacketIdEntityAction:        readEntityAction,
	PacketIdUseEntity:           readUseEntity,
	PacketIdRespawn:             readRespawn,
	PacketIdPlayerPosition:      readPlayerPosition,
	PacketIdPlayerLook:          readPlayerLook,
	PacketIdPlayerBlockHit:      readPlayerBlockHit,
	PacketIdPlayerBlockInteract: readPlayerBlockInteract,
	PacketIdEntityAnimation:     readEntityAnimation,
	PacketIdWindowTransaction:   readWindowTransaction,
	PacketIdSignUpdate:          readSignUpdate,
	PacketIdDisconnect:          readDisconnect,
}

// Client->server specific packet mapping
var serverReadFns = serverPacketReaderMap{
	PacketIdLogin:              serverReadLogin,
	PacketIdHandshake:          serverReadHandshake,
	PacketIdPlayer:             readPlayer,
	PacketIdPlayerPositionLook: serverReadPlayerPositionLook,
	PacketIdWindowClick:        readWindowClick,
	PacketIdHoldingChange:      readHoldingChange,
	PacketIdWindowClose:        readWindowClose,
	PacketIdServerListPing:     readServerListPing,
}

// Server->client specific packet mapping
var clientReadFns = clientPacketReaderMap{
	PacketIdLogin:                clientReadLogin,
	PacketIdHandshake:            clientReadHandshake,
	PacketIdTimeUpdate:           readTimeUpdate,
	PacketIdEntityEquipment:      readEntityEquipment,
	PacketIdSpawnPosition:        readSpawnPosition,
	PacketIdUpdateHealth:         readUpdateHealth,
	PacketIdPlayerPositionLook:   clientReadPlayerPositionLook,
	PacketIdBedUse:               readBedUse,
	PacketIdNamedEntitySpawn:     readNamedEntitySpawn,
	PacketIdItemSpawn:            readItemSpawn,
	PacketIdItemCollect:          readItemCollect,
	PacketIdObjectSpawn:          readObjectSpawn,
	PacketIdEntitySpawn:          readEntitySpawn,
	PacketIdPaintingSpawn:        readPaintingSpawn,
	PacketIdExperienceOrb:        readExperienceOrb,
	PacketIdEntityVelocity:       readEntityVelocity,
	PacketIdEntityDestroy:        readEntityDestroy,
	PacketIdEntity:               readEntity,
	PacketIdEntityRelMove:        readEntityRelMove,
	PacketIdEntityLook:           readEntityLook,
	PacketIdEntityLookAndRelMove: readEntityLookAndRelMove,
	PacketIdEntityTeleport:       readEntityTeleport,
	PacketIdEntityStatus:         readEntityStatus,
	PacketIdEntityMetadata:       readEntityMetadata,
	PacketIdEntityEffect:         readEntityEffect,
	PacketIdEntityRemoveEffect:   readEntityRemoveEffect,
	PacketIdPlayerExperience:     readPlayerExperience,
	PacketIdPreChunk:             readPreChunk,
	PacketIdMapChunk:             readMapChunk,
	PacketIdBlockChangeMulti:     readBlockChangeMulti,
	PacketIdBlockChange:          readBlockChange,
	PacketIdNoteBlockPlay:        readNoteBlockPlay,
	PacketIdExplosion:            readExplosion,
	PacketIdSoundEffect:          readSoundEffect,
	PacketIdState:                readState,
	PacketIdWeather:              readWeather,
	PacketIdWindowOpen:           readWindowOpen,
	PacketIdWindowSetSlot:        readWindowSetSlot,
	PacketIdWindowItems:          readWindowItems,
	PacketIdWindowProgressBar:    readWindowProgressBar,
	PacketIdQuickbarSlotUpdate:   readQuickbarSlotUpdate,
	PacketIdItemData:             readItemData,
	PacketIdIncrementStatistic:   readIncrementStatistic,
}

func readPacketId(reader io.Reader) (packetId byte, err error) {
	err = binary.Read(reader, binary.BigEndian, &packetId)
	return
}

func serverHandlePacket(reader io.Reader, handler IServerPacketHandler, packetId byte) error {
	if commonFn, ok := commonReadFns[packetId]; ok {
		return commonFn(reader, handler)
	}

	if serverFn, ok := serverReadFns[packetId]; ok {
		return serverFn(reader, handler)
	}

	return UnknownPacketIdError(packetId)
}

func ServerReadPacketExpect(reader io.Reader, handler IServerPacketHandler, expectedIds []byte) (err error) {
	var packetId byte
	if packetId, err = readPacketId(reader); err != nil {
		return
	}

	for _, expectedId := range expectedIds {
		if expectedId == packetId {
			return serverHandlePacket(reader, handler, packetId)
		}
	}

	return UnexpectedPacketIdError(packetId)
}

// A server should call this to receive a single packet from a client. It will
// block until a packet was successfully handled, or there was an error.
func ServerReadPacket(reader io.Reader, handler IServerPacketHandler) (err error) {
	var packetId byte
	if packetId, err = readPacketId(reader); err != nil {
		return
	}

	return serverHandlePacket(reader, handler, packetId)
}

func clientHandlePacket(reader io.Reader, handler IClientPacketHandler, packetId byte) error {
	if commonFn, ok := commonReadFns[packetId]; ok {
		return commonFn(reader, handler)
	}

	if clientFn, ok := clientReadFns[packetId]; ok {
		return clientFn(reader, handler)
	}

	return UnknownPacketIdError(packetId)
}

// A client should call this to receive a single packet from a client. It will
// block until a packet was successfully handled, or there was an error.
func ClientReadPacket(reader io.Reader, handler IClientPacketHandler) (err error) {
	var packetId byte
	if packetId, err = readPacketId(reader); err != nil {
		return
	}

	return clientHandlePacket(reader, handler, packetId)
}

func ClientReadPacketExpect(reader io.Reader, handler IClientPacketHandler, expectedIds []byte) (err error) {
	var packetId byte
	if packetId, err = readPacketId(reader); err != nil {
		return
	}

	for _, expectedId := range expectedIds {
		if expectedId == packetId {
			return clientHandlePacket(reader, handler, packetId)
		}
	}

	return UnexpectedPacketIdError(packetId)
}
