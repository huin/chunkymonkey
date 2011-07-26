package mob

import (
	"expvar"
	"io"
	"os"

	"chunkymonkey/physics"
	"chunkymonkey/proto"
	. "chunkymonkey/types"
)

var (
	expVarMobSpawnCount *expvar.Int
)

func init() {
	expVarMobSpawnCount = expvar.NewInt("mob-spawn-count")
}

// When using an object of type Mob or a sub-type, the caller must set an
// EntityId, most likely obtained from the EntityManager.
type Mob struct {
	EntityId
	physics.PointObject
	mobType EntityMobType
	look    LookDegrees
	// TODO(nictuku): Move to a more structured form.
	metadata map[byte]byte
	// TODO: Change to an AABB object when we have that.
}

func (mob *Mob) Init(id EntityMobType, position *AbsXyz, velocity *AbsVelocity, look *LookDegrees) {
	mob.PointObject.Init(position, velocity)
	mob.mobType = id
	mob.look = *look
	mob.metadata = map[byte]byte{
		0:  byte(0),
		16: byte(0),
	}

	expVarMobSpawnCount.Add(1)
}

func (mob *Mob) SetLook(look LookDegrees) {
	mob.look = look
}

func (mob *Mob) SetBurning(burn bool) {
	if burn {
		mob.metadata[0] |= 0x01
	} else {
		mob.metadata[0] ^= 0x01
	}
}

func (mob *Mob) Tick(blockQuerier physics.IBlockQuerier) (leftBlock bool) {
	// TODO: Spontaneous mob movement.
	return mob.PointObject.Tick(blockQuerier)
}

func (mob *Mob) FormatMetadata() []proto.EntityMetadata {
	x := make([]proto.EntityMetadata, len(mob.metadata))
	i := 0
	for k, v := range mob.metadata {
		x[i] = proto.EntityMetadata{0, k, v}
		i++
	}
	return x
}

func (mob *Mob) SendUpdate(writer io.Writer) (err os.Error) {
	if err = proto.WriteEntity(writer, mob.EntityId); err != nil {
		return
	}

	err = mob.PointObject.SendUpdate(writer, mob.EntityId, mob.look.ToLookBytes())

	return
}

func (mob *Mob) SendSpawn(writer io.Writer) (err os.Error) {
	err = proto.WriteEntitySpawn(
		writer,
		mob.EntityId,
		mob.mobType,
		&mob.PointObject.LastSentPosition,
		mob.look.ToLookBytes(),
		mob.FormatMetadata())
	if err != nil {
		return
	}
	err = proto.WriteEntityVelocity(
		writer,
		mob.EntityId,
		&mob.PointObject.LastSentVelocity)
	if err != nil {
		return
	}

	return
}

// Evil mobs.

type Creeper struct {
	Mob
}

var (
	creeperNormal   = byte(0)
	creeperBlueAura = byte(1)
)

func NewCreeper(position *AbsXyz, velocity *AbsVelocity, look *LookDegrees) (c *Creeper) {
	c = new(Creeper)
	c.Mob.Init(CreeperType.Id, position, velocity, look)
	c.Mob.metadata[17] = creeperNormal
	c.Mob.metadata[16] = byte(255)
	return c
}

func (c *Creeper) SetNormalStatus() {
	c.Mob.metadata[17] = creeperNormal
}

func (c *Creeper) CreeperSetBlueAura() {
	c.Mob.metadata[17] = creeperBlueAura
}

type Skeleton struct {
	Mob
}

func NewSkeleton(position *AbsXyz, velocity *AbsVelocity, look *LookDegrees) (s *Skeleton) {
	s = new(Skeleton)
	s.Mob.Init(SkeletonType.Id, position, velocity, look)
	return
}

type Spider struct {
	Mob
}

func NewSpider(position *AbsXyz, velocity *AbsVelocity, look *LookDegrees) (s *Spider) {
	s = new(Spider)
	s.Mob.Init(SpiderType.Id, position, velocity, look)
	return
}

type Zombie struct {
	Mob
}

func NewZombie(position *AbsXyz, velocity *AbsVelocity, look *LookDegrees) (s *Zombie) {
	s = new(Zombie)
	s.Mob.Init(ZombieType.Id, position, velocity, look)
	return
}

// Passive mobs.

type Pig struct {
	Mob
}

func NewPig(position *AbsXyz, velocity *AbsVelocity, look *LookDegrees) (p *Pig) {
	p = new(Pig)
	p.Mob.Init(PigType.Id, position, velocity, look)
	return
}

type Sheep struct {
	Mob
}

func NewSheep(position *AbsXyz, velocity *AbsVelocity, look *LookDegrees) (s *Sheep) {
	s = new(Sheep)
	s.Mob.Init(SheepType.Id, position, velocity, look)
	return
}

type Cow struct {
	Mob
}

func NewCow(position *AbsXyz, velocity *AbsVelocity, look *LookDegrees) (c *Cow) {
	c = new(Cow)
	c.Mob.Init(CowType.Id, position, velocity, look)
	return
}

type Hen struct {
	Mob
}

func NewHen(position *AbsXyz, velocity *AbsVelocity, look *LookDegrees) (h *Hen) {
	h = new(Hen)
	h.Mob.Init(HenType.Id, position, velocity, look)
	return
}

type Squid struct {
	Mob
}

func NewSquid(position *AbsXyz, velocity *AbsVelocity, look *LookDegrees) (s *Squid) {
	s = new(Squid)
	s.Mob.Init(SquidType.Id, position, velocity, look)
	return
}

type Wolf struct {
	Mob
}

func NewWolf(position *AbsXyz, velocity *AbsVelocity, look *LookDegrees) (w *Wolf) {
	w = new(Wolf)
	w.Mob.Init(WolfType.Id, position, velocity, look)
	// TODO(nictuku): String with an optional owner's username.
	w.Mob.metadata[17] = 0
	w.Mob.metadata[16] = 0
	w.Mob.metadata[18] = 0
	return w
}

// byteArray implements the sort.Interface for a slice of bytes.
// TODO: Move to a more appropriate place.
type byteArray []byte

func (p byteArray) Len() int           { return len(p) }
func (p byteArray) Less(i, j int) bool { return p[i] < p[j] }
func (p byteArray) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
