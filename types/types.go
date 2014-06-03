package types

import (
	"math"
)

// Defines the basic types such as ID types, and world units.

type Ticks int64

const (
	TicksPerDay         = 24000
	TicksPerSecond      = 20
	NanosecondsInSecond = 1e9
)

// 1 "TickTime" is the duration of a server "tick". This value is intended for
// use in sub-tick physics calculations.
type TickTime float64

type RandomSeed int64

type GameDifficulty int8

const (
	GameDifficultyPeaceful = 0
	GameDifficultyEasy     = 1
	GameDifficultyNormal   = 2
	GameDifficultyHard     = 3
)

// Which 'world'?
type DimensionId int8

const (
	DimensionNether = DimensionId(-1)
	DimensionNormal = DimensionId(0)
)

// GameType indicates the server play mode.
type GameType byte

const (
	GameTypeSurvival = GameType(0)
	GameTypeCreative = GameType(1)
)

// Player/mob health.
type Health int16

// Player food level.
type FoodUnits int16

// Item-related types

// Item type ID
type ItemTypeId int16

// ToBlockId returns the ItemTypeId as a BlockId, or 0, ok=false if it's not a
// valid BlockId.
func (id ItemTypeId) ToBlockId() (blockId BlockId, ok bool) {
	if id >= BlockIdMin && id <= BlockIdMax {
		return BlockId(id), true
	}
	return 0, false
}

// Item metadata. The meaning of this varies depending upon the item type. In
// the case of tools/armor it indicates "uses" or "damage".
type ItemData int16

// How many items are in a stack/slot etc.
type ItemCount int8

// Entity-related types

type EntityId int32

func (e EntityId) GetEntityId() EntityId {
	return e
}

func (e *EntityId) SetEntityId(entityId EntityId) {
	*e = entityId
}

// The type of mob
type EntityMobType byte

const (
	MobTypeIdCreeper      = EntityMobType(50)
	MobTypeIdSkeleton     = EntityMobType(51)
	MobTypeIdSpider       = EntityMobType(52)
	MobTypeIdGiantZombie  = EntityMobType(53)
	MobTypeIdZombie       = EntityMobType(54)
	MobTypeIdSlime        = EntityMobType(55)
	MobTypeIdGhast        = EntityMobType(56)
	MobTypeIdZombiePigman = EntityMobType(57)
	MobTypeIdPig          = EntityMobType(90)
	MobTypeIdSheep        = EntityMobType(91)
	MobTypeIdCow          = EntityMobType(92)
	MobTypeIdHen          = EntityMobType(93)
	MobTypeIdSquid        = EntityMobType(94)
	MobTypeIdWolf         = EntityMobType(95)
)

var MobTypeByName = map[string]EntityMobType{
	"Creeper":      MobTypeIdCreeper,
	"Skeleton":     MobTypeIdSkeleton,
	"Spider":       MobTypeIdSpider,
	"GiantZombie":  MobTypeIdGiantZombie,
	"Zombie":       MobTypeIdZombie,
	"Slime":        MobTypeIdSlime,
	"Ghast":        MobTypeIdGhast,
	"ZombiePigman": MobTypeIdZombiePigman,
	"Pig":          MobTypeIdPig,
	"Sheep":        MobTypeIdSheep,
	"Cow":          MobTypeIdCow,
	"Hen":          MobTypeIdHen,
	"Squid":        MobTypeIdSquid,
	"Wolf":         MobTypeIdWolf,
}

// MobNameByType reverses the MobTypeByName mapping, initialized in init().
var MobNameByType = map[EntityMobType]string{}

type EntityStatus byte

type EntityAnimation byte

const (
	EntityAnimationNone     = EntityAnimation(0)
	EntityAnimationSwingArm = EntityAnimation(1)
	EntityAnimationDamage   = EntityAnimation(2)
	EntityAnimationUnknown1 = EntityAnimation(102)
	EntityAnimationCrouch   = EntityAnimation(104)
	EntityAnimationUncrouch = EntityAnimation(105)
)

// EntityEffect is an effect that can be placed on a player (and other things?).
type EntityEffect byte

const (
	EntityEffectMoveFaster     = EntityEffect(1)
	EntityEffectMoveSlower     = EntityEffect(2)
	EntityEffectDigFaster      = EntityEffect(3)
	EntityEffectDigSlower      = EntityEffect(4)
	EntityEffectDamageBoost    = EntityEffect(5)
	EntityEffectHeal           = EntityEffect(6)
	EntityEffectHarm           = EntityEffect(7)
	EntityEffectJump           = EntityEffect(8)
	EntityEffectConfusion      = EntityEffect(9)
	EntityEffectRegeneration   = EntityEffect(10)
	EntityEffectResistance     = EntityEffect(11)
	EntityEffectFireResistance = EntityEffect(12)
	EntityEffectWaterBreathing = EntityEffect(13)
	EntityEffectInvisibility   = EntityEffect(14)
	EntityEffectBlindness      = EntityEffect(15)
	EntityEffectNightVision    = EntityEffect(16)
	EntityEffectSick           = EntityEffect(17)
	EntityEffectWeakness       = EntityEffect(18)
	EntityEffectPoison         = EntityEffect(19)
)

type EntityAction byte

const (
	EntityActionCrouch   = EntityAction(1)
	EntityActionUncrouch = EntityAction(2)
)

type ObjTypeId int8

const (
	ObjTypeIdBoat           = ObjTypeId(1)
	ObjTypeIdMinecart       = ObjTypeId(10)
	ObjTypeIdStorageCart    = ObjTypeId(11)
	ObjTypeIdPoweredCart    = ObjTypeId(12)
	ObjTypeIdActivatedTnt   = ObjTypeId(50)
	ObjTypeIdArrow          = ObjTypeId(60)
	ObjTypeIdThrownSnowball = ObjTypeId(61)
	ObjTypeIdThrownEgg      = ObjTypeId(62)
	ObjTypeIdFallingSand    = ObjTypeId(70)
	ObjTypeIdFallingGravel  = ObjTypeId(71)
	ObjTypeIdFishingFloat   = ObjTypeId(90)
)

var ObjTypeByName = map[string]ObjTypeId{
	"Boat":           ObjTypeIdBoat,
	"Minecart":       ObjTypeIdMinecart,
	"StorageCart":    ObjTypeIdStorageCart,
	"PoweredCart":    ObjTypeIdPoweredCart,
	"ActivatedTnt":   ObjTypeIdActivatedTnt,
	"Arrow":          ObjTypeIdArrow,
	"ThrownSnowball": ObjTypeIdThrownSnowball,
	"ThrownEgg":      ObjTypeIdThrownEgg,
	"FallingSand":    ObjTypeIdFallingSand,
	"FallingGravel":  ObjTypeIdFallingGravel,
	"FishingFloat":   ObjTypeIdFishingFloat,
}

// ObjNameByType reverses ObjTypeByName (initialized in init()).
var ObjNameByType = map[ObjTypeId]string{}

type PaintingTypeId int32

type InstrumentId byte

const (
	InstrumentIdDoubleBass = InstrumentId(1)
	InstrumentIdSnareDrum  = InstrumentId(2)
	InstrumentIdSticks     = InstrumentId(3)
	InstrumentIdBassDrum   = InstrumentId(4)
	InstrumentIdHarp       = InstrumentId(5)
)

type NotePitch int8

const (
	NotePitchMin = NotePitch(0)
	NotePitchMax = NotePitch(24)
)

// SoundEffect is an enumeration of sound effects.
type SoundEffect int32

const (
	SoundEffectClick2     = SoundEffect(1000)
	SoundEffectClick1     = SoundEffect(1001)
	SoundEffectBowFire    = SoundEffect(1002)
	SoundEffectDoor       = SoundEffect(1003)
	SoundEffectExtinguish = SoundEffect(1004)
	SoundEffectRecordPlay = SoundEffect(1005)
	SoundEffectSmoke      = SoundEffect(2000)
	SoundEffectBlockBreak = SoundEffect(2001)
)

// Block-related types

type BlockId byte

const (
	BlockIdMin = 0
	BlockIdAir = BlockId(0)
	BlockIdMax = 255
)

// Block face (0-5)
type Face int8

// Used when a block face is not appropriate to the situation, but block
// location data passed (such as using an item not on a block).
const (
	FaceNull     = Face(-1)
	FaceMinValid = 0
	FaceBottom   = 0
	FaceTop      = 1
	FaceEast     = 2
	FaceWest     = 3
	FaceNorth    = 4
	FaceSouth    = 5
	FaceMaxValid = 5
)

func (f Face) Dxyz() (dx BlockCoord, dy BlockYCoord, dz BlockCoord) {
	switch f {
	case FaceBottom:
		dy = -1
	case FaceTop:
		dy = 1
	case FaceEast:
		dz = -1
	case FaceWest:
		dz = 1
	case FaceNorth:
		dx = -1
	case FaceSouth:
		dx = 1
	}
	return
}

// Action-related types and constants

type DigStatus byte

const (
	DigStarted    = DigStatus(0)
	DigBlockBroke = DigStatus(2)
	DigDropItem   = DigStatus(4)
)

const (
	// MaxInteractDistance is the maximum distance at which a player can interact
	// with something, such as dig.
	MaxInteractDistance = AbsCoord(6)
)

// Window/inventory-related types and constants

// ID specifying which slotted window, such as inventory
type WindowId int8

const (
	WindowIdCursor    = WindowId(-1)
	WindowIdInventory = WindowId(0)
	WindowIdFreeMin   = WindowId(1)
	WindowIdFreeMax   = WindowId(127)
)

type InvTypeId int8

const (
	InvTypeIdChest     = InvTypeId(0)
	InvTypeIdWorkbench = InvTypeId(1)
	InvTypeIdFurnace   = InvTypeId(2)
	InvTypeIdDispenser = InvTypeId(3)
)

// ID of the slow in inventory or other item-slotted window element
type SlotId int16

const (
	SlotIdCursor = SlotId(-1)
	SlotIdNull   = SlotId(999) // Clicked outside window.
)

type PrgBarId int16

const (
	PrgBarIdFurnaceProgress = PrgBarId(0)
	PrgBarIdFurnaceFire     = PrgBarId(1)
)

type PrgBarValue int16

// ID specifying a player statistic.
type StatisticId int32

// Transaction ID.
type TxId int16

// Transaction state.
type TxState byte

const (
	TxStateAccepted = TxState(iota)
	TxStateRejected
	TxStateDeferred
)

// Movement-related types and constants

// VelocityComponent in VelocityComponentBlocksPerTick
type VelocityComponent int16

const (
	VelocityComponentMax = 28800
	VelocityComponentMin = -28800

	AbsToIntVelocityComponent = TicksPerSecond * 32000 / 100
)

type Velocity struct {
	X, Y, Z VelocityComponent
}

// AbsVelocityCoord is measured in blocks per tick.
type AbsVelocityCoord AbsCoord

func (v AbsVelocityCoord) ToVelocityComponent() VelocityComponent {
	scaledV := v * AbsToIntVelocityComponent
	if scaledV > VelocityComponentMax {
		return VelocityComponentMax
	} else if scaledV < VelocityComponentMin {
		return VelocityComponentMin
	}
	return VelocityComponent(scaledV)
}

type AbsVelocity struct {
	X, Y, Z AbsVelocityCoord
}

func (v *AbsVelocity) ToVelocity() *Velocity {
	return &Velocity{
		v.X.ToVelocityComponent(),
		v.Y.ToVelocityComponent(),
		v.Z.ToVelocityComponent(),
	}
}

// Relative movement, using same units as AbsIntCoord, but in byte form so
// constrained
type RelMoveCoord int8

type RelMove struct {
	X, Y, Z RelMoveCoord
}

// Angle-related types and constants

const (
	DegreesToBytes = 256.0 / 360.0
)

// An angle, where there are 256 units in a circle.
type AngleBytes byte

// An angle in degrees
type AngleDegrees float32

func (d *AngleDegrees) ToAngleBytes() AngleBytes {
	norm := math.Mod(float64(*d), 360)
	if norm < 0 {
		norm = 360 + norm
	}
	return AngleBytes(norm * DegreesToBytes)
}

type LookDegrees struct {
	// Pitch is -ve when looking above the horizontal, and +ve below
	Yaw, Pitch AngleDegrees
}

func (l *LookDegrees) ToLookBytes() *LookBytes {
	return &LookBytes{
		l.Yaw.ToAngleBytes(),
		l.Pitch.ToAngleBytes(),
	}
}

type LookBytes struct {
	Yaw, Pitch AngleBytes
}

type OrientationDegrees struct {
	Yaw, Pitch, Roll AngleDegrees
}

type OrientationBytes struct {
	Yaw, Pitch, Roll AngleBytes
}

// Location-related types and constants

const (
	ChunkHShift = 4
	ChunkYShift = 7
	// Chunk coordinates can be converted to block coordinates
	ChunkSizeH = 1 << ChunkHShift
	ChunkSizeY = 1 << ChunkYShift
	ChunkHMask = ChunkSizeH - 1
	ChunkYMask = ChunkSizeY - 1

	// The area within which a client receives updates.
	ChunkRadius = 10
	// The radius in which all chunks must be sent before completing a client's
	// login process.
	MinChunkRadius = 2

	// Sometimes it is useful to convert block coordinates to pixels
	PixelShift     = 5
	PixelsPerBlock = 1 << PixelShift

	PixelsPerChunkShift = (ChunkHShift + PixelShift)
	PixelsPerChunk      = 1 << PixelsPerChunkShift

	// Millipixels are used in velocity values
	MilliPixelsPerPixel = 1000
	MilliPixelsPerBlock = PixelsPerBlock * MilliPixelsPerPixel
)

// Specifies exact world distance in blocks (floating point)
type AbsCoord float64

type AbsXyz struct {
	X, Y, Z AbsCoord
}

// Convert an (x, z) absolute coordinate pair to chunk coordinates
func (p *AbsXyz) ToChunkXz() ChunkXz {
	return ChunkXz{
		X: ChunkCoord(math.Floor(float64(p.X / ChunkSizeH))),
		Z: ChunkCoord(math.Floor(float64(p.Z / ChunkSizeH))),
	}
}

func (p *AbsXyz) ApplyVelocity(dt TickTime, v *AbsVelocity) {
	p.X += AbsCoord(float64(v.X) * float64(dt))
	p.Y += AbsCoord(float64(v.Y) * float64(dt))
	p.Z += AbsCoord(float64(v.Z) * float64(dt))
}

func (p *AbsXyz) ToAbsIntXyz() *AbsIntXyz {
	return &AbsIntXyz{
		AbsIntCoord(p.X * PixelsPerBlock),
		AbsIntCoord(p.Y * PixelsPerBlock),
		AbsIntCoord(p.Z * PixelsPerBlock),
	}
}

func (p *AbsXyz) ToBlockXyz() *BlockXyz {
	return &BlockXyz{
		BlockCoord(math.Floor(float64(p.X))),
		BlockYCoord(math.Floor(float64(p.Y))),
		BlockCoord(math.Floor(float64(p.Z))),
	}
}

func (p *AbsXyz) ToShardXz() ShardXz {
	return ShardXz{
		X: ShardCoord(math.Floor(float64(p.X / (ChunkSizeH * ShardSize)))),
		Z: ShardCoord(math.Floor(float64(p.Z / (ChunkSizeH * ShardSize)))),
	}
}

func (p *AbsXyz) IsWithinDistanceOf(other *AbsXyz, maxDistance AbsCoord) bool {
	dx := p.X - other.X
	dy := p.Y - other.Y
	dz := p.Z - other.Z
	return (dx*dx + dy*dy + dz*dz) <= maxDistance*maxDistance
}

// Specifies approximate world distance in pixels (absolute / PixelsPerBlock)
type AbsIntCoord int32

type AbsIntXyz struct {
	X, Y, Z AbsIntCoord
}

func (p *AbsIntXyz) ToBlockXyz() *BlockXyz {
	return &BlockXyz{
		BlockCoord(p.X / PixelsPerBlock),
		BlockYCoord(p.Y / PixelsPerBlock),
		BlockCoord(p.Z / PixelsPerBlock),
	}
}

// Convert (x, z) absolute integer coordinates to chunk coordinates
func (abs *AbsIntXyz) ToChunkXz() *ChunkXz {
	return &ChunkXz{
		ChunkCoord(abs.X >> PixelsPerChunkShift),
		ChunkCoord(abs.Z >> PixelsPerChunkShift),
	}
}

func (abs *AbsIntXyz) IAdd(dx, dy, dz AbsIntCoord) {
	abs.X += dx
	abs.Y += dy
	abs.Z += dz
}

// Shard types and data.

const (
	// Each shard is ShardSize * ShardSize chunks square.
	ShardSize = 16
)

type ShardCoord int32

type ShardXz struct {
	X, Z ShardCoord
}

func (loc *ShardXz) ToChunkXz() ChunkXz {
	return ChunkXz{
		X: ChunkCoord(loc.X * ShardSize),
		Z: ChunkCoord(loc.Z * ShardSize),
	}
}

// Converts a ShardXz location into a key suitable for using in a hash.
func (loc *ShardXz) Key() uint64 {
	return uint64(loc.X)<<32 | uint64(uint32(loc.Z))
}

func (loc *ShardXz) Equals(other *ShardXz) bool {
	return loc.X == other.X && loc.Z == other.Z
}

// Coordinate of a chunk in the world (block / 16).
type ChunkCoord int32

func (c ChunkCoord) Abs() ChunkCoord {
	if c < 0 {
		return -c
	}
	return c
}

func (c ChunkCoord) ToShardCoord() (s ShardCoord) {
	s = ShardCoord(c / ShardSize)
	if c%ShardSize < 0 {
		s--
	}
	return
}

// ChunkXz represents the position of a chunk within the world.
type ChunkXz struct {
	X, Z ChunkCoord
}

func (chunkLoc ChunkXz) Equals(rhs ChunkXz) bool {
	return chunkLoc.X == rhs.X && chunkLoc.Z == rhs.Z
}

// Returns the world BlockXyz position of the (0, 0, 0) block in the chunk
func (chunkLoc *ChunkXz) ChunkCornerBlockXY() *BlockXyz {
	return &BlockXyz{
		BlockCoord(chunkLoc.X) * ChunkSizeH,
		0,
		BlockCoord(chunkLoc.Z) * ChunkSizeH,
	}
}

// Convert a position within a chunk to a block position within the world
func (chunkLoc *ChunkXz) ToBlockXyz(subLoc *SubChunkXyz) *BlockXyz {
	return &BlockXyz{
		BlockCoord(chunkLoc.X)*ChunkSizeH + BlockCoord(subLoc.X),
		BlockYCoord(subLoc.Y),
		BlockCoord(chunkLoc.Z)*ChunkSizeH + BlockCoord(subLoc.Z),
	}
}

// Converts a chunk location into a key suitable for using in a hash.
func (chunkLoc *ChunkXz) ChunkKey() uint64 {
	return uint64(chunkLoc.X)<<32 | uint64(uint32(chunkLoc.Z))
}

// ToShardXz returns the location of the shard that the chunk is within.
func (chunkLoc *ChunkXz) ToShardXz() ShardXz {
	return ShardXz{
		X: chunkLoc.X.ToShardCoord(),
		Z: chunkLoc.Z.ToShardCoord(),
	}
}

// Size of a sub-chunk
type SubChunkSizeCoord byte

// Coordinate of a block within a chunk
type SubChunkCoord byte

type SubChunkSize struct {
	X, Y, Z SubChunkSizeCoord
}

// SubChunkXyz represents the position of a block within a chunk.
type SubChunkXyz struct {
	X, Y, Z SubChunkCoord
}

// BlockIndex returns the relevant index for a block with a given position
// within a chunk. If subLoc represents an invalid position, then ok=False is
// returned.
func (subLoc *SubChunkXyz) BlockIndex() (index BlockIndex, ok bool) {
	ok = (subLoc.X|subLoc.Z)&^ChunkHMask == 0 && subLoc.Y&^ChunkYMask == 0
	index = ((BlockIndex(subLoc.X) << (ChunkHShift + ChunkYShift)) |
		BlockIndex(subLoc.Y) |
		(BlockIndex(subLoc.Z) << ChunkYShift))

	return
}

type BlockIndex uint32

func (bi BlockIndex) ToSubChunkXyz() (subLoc SubChunkXyz) {
	subLoc.Y = SubChunkCoord(bi & ChunkYMask)
	bi >>= ChunkYShift
	subLoc.Z = SubChunkCoord(bi & ChunkHMask)
	bi >>= ChunkHShift
	subLoc.X = SubChunkCoord(bi & ChunkHMask)
	return
}

func (bi BlockIndex) BlockId(blocks []byte) BlockId {
	return BlockId(blocks[bi])
}

func (bi BlockIndex) BlockData(blockData []byte) byte {
	shift := (bi & 1) << 2
	index := bi >> 1
	return (blockData[index] >> shift) & 0xf
}

func (bi BlockIndex) SetBlockId(blocks []byte, id BlockId) {
	blocks[bi] = byte(id)
}

// SetBlockData is used to set block metadata inside an array of bytes, where
// each byte contains packed nibbles.
func (bi BlockIndex) SetBlockData(blockData []byte, data byte) {
	index := bi >> 1

	combinedData := blockData[index]

	shift := (bi & 1) << 2
	mask := byte(0x0f) << shift
	combinedData = ((data << shift) & mask) | (combinedData & ^mask)

	blockData[index] = combinedData
}

// Coordinate of a block within the world
type BlockCoord int32
type BlockYCoord int8

func (b BlockCoord) ToChunkLocalCoord() (c ChunkCoord, s SubChunkCoord) {
	return ChunkCoord(b >> ChunkHShift), SubChunkCoord(b & ChunkHMask)
}

// BlockXyz represents the position of a block within the world.
type BlockXyz struct {
	X BlockCoord
	Y BlockYCoord
	Z BlockCoord
}

const (
	MaxXCoord = math.MaxInt32
	MinXCoord = math.MinInt32
	MaxYCoord = math.MaxInt8
	MinYCoord = 0
	MaxZCoord = math.MaxInt32
	MinZCoord = math.MinInt32
)

func (b *BlockXyz) Equals(rhs BlockXyz) bool {
	return b.X == rhs.X && b.Y == rhs.Y && b.Z == rhs.Z
}

// Test if a block location is not appropriate to the situation, but block
// location data passed (such as using an item not on a block).
func (b *BlockXyz) IsNull() bool {
	return b.Y == -1 && b.X == -1 && b.Z == -1
}

// Test if a block location is the 0 block. This is used in certain situations
// such as PacketPlayerBlockHit, when the player is throwing an item rather
// than hitting an item in a chunk.
func (b *BlockXyz) IsZero() bool {
	return b.Y == 0 && b.X == 0 && b.Z == 0
}

// Translate one block location to another by dx, dy, dz, checking for
// overflow. If overflow occurs, return nil. There may be a more elegant
// solution to check this, here we go for simplicity and clarity. This
// function assumes we cannot have a negative Y coordinate.
func (b *BlockXyz) AddXyz(dx BlockCoord, dy BlockYCoord, dz BlockCoord) (newb *BlockXyz) {
	// Check for overflow
	sumx := b.X + dx
	if b.X >= 0 && sumx < dx {
		return nil
	} else if b.X < 0 && sumx > dx {
		return nil
	}

	sumy := b.Y + dy
	if b.Y >= 0 && sumy < dy {
		return nil
	} else if b.Y < 0 && sumy > dy {
		return nil
	}

	sumz := b.Z + dz
	if b.Z >= 0 && sumz < dz {
		return nil
	} else if b.Z < 0 && sumz > dz {
		return nil
	}

	if sumx > MaxXCoord || sumx < MinXCoord {
		return nil
	}
	if sumy > MaxYCoord || sumy < MinYCoord {
		return nil
	}
	if sumz > MaxZCoord || sumz < MinZCoord {
		return nil
	}

	return &BlockXyz{sumx, sumy, sumz}
}

// Convert an (x, y, z) block coordinate to chunk coordinates.
func (blockLoc *BlockXyz) ToChunkXz() (chunkLoc *ChunkXz) {
	chunkX, _ := blockLoc.X.ToChunkLocalCoord()
	chunkZ, _ := blockLoc.Z.ToChunkLocalCoord()

	chunkLoc = &ChunkXz{ChunkCoord(chunkX), ChunkCoord(chunkZ)}
	return
}

// Convert an (x, y, z) block coordinate to chunk coordinates and the
// coordinates of the block within the chunk
func (blockLoc *BlockXyz) ToChunkLocal() (chunkLoc *ChunkXz, subLoc *SubChunkXyz) {
	chunkX, subX := blockLoc.X.ToChunkLocalCoord()
	chunkZ, subZ := blockLoc.Z.ToChunkLocalCoord()

	chunkLoc = &ChunkXz{ChunkCoord(chunkX), ChunkCoord(chunkZ)}
	subLoc = &SubChunkXyz{SubChunkCoord(subX), SubChunkCoord(blockLoc.Y), SubChunkCoord(subZ)}
	return
}

func (blockLoc *BlockXyz) ToAbsIntXyz() *AbsIntXyz {
	return &AbsIntXyz{
		AbsIntCoord(blockLoc.X) * PixelsPerBlock,
		AbsIntCoord(blockLoc.Y) * PixelsPerBlock,
		AbsIntCoord(blockLoc.Z) * PixelsPerBlock,
	}
}

func (blockLoc *BlockXyz) ToAbsXyz() *AbsXyz {
	return &AbsXyz{
		AbsCoord(blockLoc.X),
		AbsCoord(blockLoc.Y),
		AbsCoord(blockLoc.Z),
	}
}

func (blockLoc *BlockXyz) MidPointToAbsXyz() AbsXyz {
	return AbsXyz{
		AbsCoord(blockLoc.X) + 0.5,
		AbsCoord(blockLoc.Y) + 0.5,
		AbsCoord(blockLoc.Z) + 0.5,
	}
}

// Misc. types and constants

type ChunkLoadMode byte

const (
	// Client should unload the chunk
	ChunkUnload = ChunkLoadMode(0)

	// Client should initialise the chunk
	ChunkInit = ChunkLoadMode(1)
)

func init() {
	// Create reverse mappings.
	for name, type_ := range ObjTypeByName {
		ObjNameByType[type_] = name
	}
	for name, type_ := range MobTypeByName {
		MobNameByType[type_] = name
	}
}
