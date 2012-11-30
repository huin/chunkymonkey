// Defines non-block movable objects such as arrows in flight, boats and
// minecarts.

package gamerules

import (
	"errors"

	"chunkymonkey/physics"
	"chunkymonkey/proto"
	. "chunkymonkey/types"
	"nbt"
)

// TODO Object sub-types?

type Object struct {
	EntityId
	ObjTypeId
	physics.PointObject
	orientation OrientationBytes
}

func NewObject(objType ObjTypeId) (object *Object) {
	object = &Object{
		// TODO: proper orientation
		orientation: OrientationBytes{0, 0, 0},
	}
	object.ObjTypeId = objType
	return
}

func (object *Object) UnmarshalNbt(tag nbt.Compound) (err error) {
	if err = object.PointObject.UnmarshalNbt(tag); err != nil {
		return
	}

	var typeName string
	if entityObjectId, ok := tag.Lookup("id").(*nbt.String); !ok {
		return errors.New("missing object type id")
	} else {
		typeName = entityObjectId.Value
	}

	var ok bool
	if object.ObjTypeId, ok = ObjTypeByName[typeName]; !ok {
		return errors.New("unknown object type id")
	}

	// TODO load orientation

	return
}

func (object *Object) MarshalNbt(tag nbt.Compound) (err error) {
	objTypeName, ok := ObjNameByType[object.ObjTypeId]
	if !ok {
		return errors.New("unknown object type")
	}
	if err = object.PointObject.MarshalNbt(tag); err != nil {
		return
	}
	tag.Set("id", &nbt.String{objTypeName})
	// TODO unknown fields
	return
}

func (object *Object) SpawnPackets(pkts []proto.IPacket) []proto.IPacket {
	return append(pkts,
		&proto.PacketObjectSpawn{
			EntityId: object.EntityId,
			ObjType:  object.ObjTypeId,
			Position: object.PointObject.LastSentPosition,
		},
		&proto.PacketEntityVelocity{
			EntityId: object.EntityId,
			Velocity: object.PointObject.LastSentVelocity,
		},
	)
}

func (object *Object) UpdatePackets(pkts []proto.IPacket) []proto.IPacket {
	pkts = append(pkts, &proto.PacketEntity{object.EntityId})

	// TODO: Should this be the Rotation information?
	pkts = object.PointObject.UpdatePackets(pkts, object.EntityId, LookBytes{})

	return pkts
}

func NewBoat() INonPlayerEntity {
	return NewObject(ObjTypeIdBoat)
}

func NewMinecart() INonPlayerEntity {
	return NewObject(ObjTypeIdMinecart)
}

func NewStorageCart() INonPlayerEntity {
	return NewObject(ObjTypeIdStorageCart)
}

func NewPoweredCart() INonPlayerEntity {
	return NewObject(ObjTypeIdPoweredCart)
}

func NewEnderCrystal() INonPlayerEntity {
	return NewObject(ObjTypeIdEnderCrystal)
}

func NewActivatedTnt() INonPlayerEntity {
	return NewObject(ObjTypeIdActivatedTnt)
}

func NewArrow() INonPlayerEntity {
	return NewObject(ObjTypeIdArrow)
}

func NewThrownSnowball() INonPlayerEntity {
	return NewObject(ObjTypeIdThrownSnowball)
}

func NewThrownEgg() INonPlayerEntity {
	return NewObject(ObjTypeIdThrownEgg)
}

func NewFallingSand() INonPlayerEntity {
	return NewObject(ObjTypeIdFallingSand)
}

func NewFallingGravel() INonPlayerEntity {
	return NewObject(ObjTypeIdFallingGravel)
}

func NewFishingFloat() INonPlayerEntity {
	return NewObject(ObjTypeIdFishingFloat)
}
