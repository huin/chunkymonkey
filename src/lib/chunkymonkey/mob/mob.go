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

type Mob struct {
	entity.Entity
	mobType  types.EntityMobType
	position types.AbsXyz
	look     types.LookDegrees
	metadata map[byte]byte
}

// If we don't care about locking these resources, we could expose the fields instead?
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

func (mob *Mob) SetBurning() {
	mob.metadata[0] = 0x01
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

// ======================= CREEPER ======================
var (
	creeperNormal   = byte(0)
	creeperBlueAura = byte(1)
)

type Creeper struct {
	Mob
}

func NewCreeper() (c *Creeper) {
	m := Mob{}
	c = &Creeper{m}

	c.Mob.mobType = CreeperType.Id
	c.Mob.look = types.LookDegrees{200, 0}
	c.Mob.metadata = map[byte]byte{}
	return c
}

func (c *Creeper) SetNormalStatus() {
	c.Mob.metadata[17] = creeperNormal
}

func (c *Creeper) CreeperSetBlueAura() {
	c.Mob.metadata[17] = creeperBlueAura
}
