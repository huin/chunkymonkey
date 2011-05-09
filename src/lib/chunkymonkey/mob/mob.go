package mob

import (
	"expvar"
	"io"
	"os"

	"chunkymonkey/entity"
	"chunkymonkey/proto"
	"chunkymonkey/types"
)

var (
	expVarMobSpawnCount *expvar.Int
)

func init() {
	expVarMobSpawnCount = expvar.NewInt("mob-spawn-count")
}

func newMob(id types.EntityMobType) Mob {
	m := Mob{}
	m.mobType = id
	m.look = types.LookDegrees{200, 0}
	m.metadata = map[byte]byte{
		0:  byte(0),
		16: byte(0),
	}
	return m
}

// When using an object of type Mob or a sub-type, the caller must:
// - set the entity ID using, for example, game.AddEntity(Mob.GetEntity()).
// - set a valid position with Mob.SetPosition().
type Mob struct {
	entity.Entity
	mobType  types.EntityMobType
	position types.AbsXyz
	look     types.LookDegrees
	metadata map[byte]byte
}

func (mob *Mob) SetPosition(pos types.AbsXyz) {
	mob.position = pos
}

func (mob *Mob) SetLook(look types.LookDegrees) {
	mob.look = look
}

func (mob *Mob) GetEntityId() types.EntityId {
	return mob.EntityId
}

func (mob *Mob) GetEntity() *entity.Entity {
	return &mob.Entity
}

func (mob *Mob) SetBurning(burn bool) {
	if burn {
		mob.metadata[0] |= 0x01
	} else {
		mob.metadata[0] ^= 0x01
	}
}

func (mob *Mob) FormatMetadata() []proto.EntityMetadata {
	x := []proto.EntityMetadata{}
	for k, v := range mob.metadata {
		x = append(x, proto.EntityMetadata{0, k, v})
	}
	return x
}

func (mob *Mob) SendSpawn(writer io.Writer) (err os.Error) {
	err = proto.WriteEntitySpawn(
		writer,
		mob.Entity.EntityId,
		mob.mobType,
		mob.position.ToAbsIntXyz(),
		mob.look.ToLookBytes(),
		mob.FormatMetadata())
	if err != nil {
		expVarMobSpawnCount.Add(1)
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

func NewCreeper() (c *Creeper) {
	c = &Creeper{newMob(CreeperType.Id)}
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

func NewSkeleton() (s *Skeleton) {
	return &Skeleton{newMob(SkeletonType.Id)}
}


type Spider struct {
	Mob
}

func NewSpider() (s *Spider) {
	return &Spider{newMob(SpiderType.Id)}
}


// Passive mobs.

type Pig struct {
	Mob
}

func NewPig() (p *Pig) {
	return &Pig{newMob(PigType.Id)}
}

type Sheep struct {
	Mob
}

func NewSheep() (s *Sheep) {
	return &Sheep{newMob(SheepType.Id)}
}

type Cow struct {
	Mob
}

func NewCow() (c *Cow) {
	return &Cow{newMob(CowType.Id)}
}

type Hen struct {
	Mob
}

func NewHen() (h *Hen) {
	return &Hen{newMob(HenType.Id)}
}

type Squid struct {
	Mob
}

func NewSquid() (s *Squid) {
	return &Squid{newMob(SquidType.Id)}
}

type Wolf struct {
	Mob
}

func NewWolf() (w *Wolf) {
	w = &Wolf{newMob(WolfType.Id)}
	// TODO(nictuku): String with an optional owner's username.
	w.Mob.metadata[17] = 0
	w.Mob.metadata[16] = 0
	w.Mob.metadata[18] = 0
	return w
}
