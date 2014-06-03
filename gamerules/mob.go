package gamerules

import (
	"errors"
	"expvar"
	"io"

	"chunkymonkey/nbtutil"
	"chunkymonkey/physics"
	"chunkymonkey/proto"
	. "chunkymonkey/types"
	"nbt"
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

func (mob *Mob) Init(id EntityMobType) {
	mob.mobType = id
	mob.metadata = map[byte]byte{
		0:  byte(0),
		16: byte(0),
	}

	expVarMobSpawnCount.Add(1)
}

func (mob *Mob) UnmarshalNbt(tag *nbt.Compound) (err error) {
	if err = mob.PointObject.UnmarshalNbt(tag); err != nil {
		return
	}

	if mob.look, err = nbtutil.ReadLookDegrees(tag, "Rotation"); err != nil {
		return
	}

	// TODO
	_ = tag.Lookup("Air").(*nbt.Short).Value
	_ = tag.Lookup("AttackTime").(*nbt.Short).Value
	_ = tag.Lookup("DeathTime").(*nbt.Short).Value
	_ = tag.Lookup("FallDistance").(*nbt.Float).Value
	_ = tag.Lookup("Fire").(*nbt.Short).Value
	_ = tag.Lookup("Health").(*nbt.Short).Value
	_ = tag.Lookup("HurtTime").(*nbt.Short).Value

	return nil
}

func (mob *Mob) MarshalNbt(tag *nbt.Compound) (err error) {
	mobTypeName, ok := MobNameByType[mob.mobType]
	if !ok {
		return errors.New("unknown mob type")
	}
	if err = mob.PointObject.MarshalNbt(tag); err != nil {
		return
	}
	tag.Set("id", &nbt.String{mobTypeName})
	tag.Set("Rotation", &nbt.List{nbt.TagFloat, []nbt.ITag{
		&nbt.Float{float32(mob.look.Yaw)},
		&nbt.Float{float32(mob.look.Pitch)},
	}})
	// TODO
	tag.Set("Air", &nbt.Short{0})
	tag.Set("AttackTime", &nbt.Short{0})
	tag.Set("DeathTime", &nbt.Short{0})
	tag.Set("FallDistance", &nbt.Float{0})
	tag.Set("Fire", &nbt.Short{0})
	tag.Set("Health", &nbt.Short{0})
	tag.Set("HurtTime", &nbt.Short{0})
	return nil
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

func (mob *Mob) SendUpdate(writer io.Writer) (err error) {
	if err = proto.WriteEntity(writer, mob.EntityId); err != nil {
		return
	}

	err = mob.PointObject.SendUpdate(writer, mob.EntityId, mob.look.ToLookBytes())

	return
}

func (mob *Mob) SendSpawn(writer io.Writer) (err error) {
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

func NewCreeper() INonPlayerEntity {
	c := new(Creeper)
	c.Mob.Init(CreeperType.Id)
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

func NewSkeleton() INonPlayerEntity {
	s := new(Skeleton)
	s.Mob.Init(SkeletonType.Id)
	return s
}

type Spider struct {
	Mob
}

func NewSpider() INonPlayerEntity {
	s := new(Spider)
	s.Mob.Init(SpiderType.Id)
	return s
}

type Zombie struct {
	Mob
}

func NewZombie() INonPlayerEntity {
	z := new(Zombie)
	z.Mob.Init(ZombieType.Id)
	return z
}

// Passive mobs.

type Pig struct {
	Mob
}

func NewPig() INonPlayerEntity {
	p := new(Pig)
	p.Mob.Init(PigType.Id)
	return p
}

type Sheep struct {
	Mob
}

func NewSheep() INonPlayerEntity {
	s := new(Sheep)
	s.Mob.Init(SheepType.Id)
	return s
}

type Cow struct {
	Mob
}

func NewCow() INonPlayerEntity {
	c := new(Cow)
	c.Mob.Init(CowType.Id)
	return c
}

type Hen struct {
	Mob
}

func NewHen() INonPlayerEntity {
	h := new(Hen)
	h.Mob.Init(HenType.Id)
	return h
}

type Squid struct {
	Mob
}

func NewSquid() INonPlayerEntity {
	s := new(Squid)
	s.Mob.Init(SquidType.Id)
	return s
}

type Wolf struct {
	Mob
}

func NewWolf() INonPlayerEntity {
	w := new(Wolf)
	w.Mob.Init(WolfType.Id)
	// TODO(nictuku): String with an optional owner's username.
	w.Mob.metadata[17] = 0
	w.Mob.metadata[16] = 0
	w.Mob.metadata[18] = 0
	return w
}
