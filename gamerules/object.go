// Defines non-block movable objects such as arrows in flight, boats and
// minecarts.

package gamerules

import (
	"errors"
	"io"

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

func (object *Object) UnmarshalNbt(tag *nbt.Compound) (err error) {
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

func (object *Object) MarshalNbt(tag *nbt.Compound) (err error) {
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

func (object *Object) SendSpawn(writer io.Writer) (err error) {
	// TODO: Send non-nil ObjectData (is there any?)
	err = proto.WriteObjectSpawn(writer, object.EntityId, object.ObjTypeId, &object.PointObject.LastSentPosition, nil)
	if err != nil {
		return
	}

	err = proto.WriteEntityVelocity(writer, object.EntityId, &object.PointObject.LastSentVelocity)
	return
}

func (object *Object) SendUpdate(writer io.Writer) (err error) {
	if err = proto.WriteEntity(writer, object.EntityId); err != nil {
		return
	}

	// TODO: Should this be the Rotation information?
	err = object.PointObject.SendUpdate(writer, object.EntityId, &LookBytes{0, 0})

	return
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
