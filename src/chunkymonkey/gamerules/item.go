package gamerules

import (
	"errors"

	"chunkymonkey/physics"
	"chunkymonkey/proto"
	. "chunkymonkey/types"
	"nbt"
)

type Item struct {
	EntityId
	Slot
	physics.PointObject
	orientation    OrientationBytes
	PickupImmunity Ticks
}

func NewBlankItem() INonPlayerEntity {
	return new(Item)
}

func NewItem(itemTypeId ItemTypeId, count ItemCount, data ItemData, position AbsXyz, velocity AbsVelocity, pickupImmunity Ticks) (item *Item) {
	item = &Item{
		Slot: Slot{
			ItemTypeId: itemTypeId,
			Count:      count,
			Data:       data,
		},
		PickupImmunity: pickupImmunity,
	}
	item.PointObject.Init(position, velocity)
	return
}

func (item *Item) UnmarshalNbt(tag nbt.Compound) (err error) {
	if err = item.PointObject.UnmarshalNbt(tag); err != nil {
		return
	}

	itemInfo, ok := tag.Lookup("Item").(nbt.Compound)
	if !ok {
		return errors.New("bad item data")
	}

	// Grab the basic item data
	id, idOk := itemInfo.Lookup("id").(*nbt.Short)
	count, countOk := itemInfo.Lookup("Count").(*nbt.Byte)
	data, dataOk := itemInfo.Lookup("Damage").(*nbt.Short)
	if !idOk || !countOk || !dataOk {
		return errors.New("bad item data")
	}

	item.Slot = Slot{
		ItemTypeId: ItemTypeId(id.Value),
		Count:      ItemCount(count.Value),
		Data:       ItemData(data.Value),
	}

	return nil
}

func (item *Item) MarshalNbt(tag nbt.Compound) (err error) {
	if err = item.PointObject.MarshalNbt(tag); err != nil {
		return
	}
	tag.Set("id", &nbt.String{"Item"})
	tag.Set("Item", nbt.Compound{
		"id":     &nbt.Short{int16(item.ItemTypeId)},
		"Count":  &nbt.Byte{int8(item.Count)},
		"Damage": &nbt.Short{int16(item.Data)},
	})
	return nil
}

func (item *Item) GetSlot() *Slot {
	return &item.Slot
}

func (item *Item) SpawnPackets(pkts []proto.IPacket) []proto.IPacket {
	return append(pkts,
		&proto.PacketItemSpawn{
			EntityId:    item.EntityId,
			ItemTypeId:  item.ItemTypeId,
			Count:       item.Slot.Count,
			Data:        item.Slot.Data,
			Position:    item.PointObject.LastSentPosition,
			Orientation: item.orientation,
		},
		&proto.PacketEntityVelocity{
			EntityId: item.EntityId,
			Velocity: item.PointObject.LastSentVelocity,
		},
	)
}

func (item *Item) UpdatePackets(pkts []proto.IPacket) []proto.IPacket {
	pkts = append(pkts, &proto.PacketEntity{
		EntityId: item.EntityId,
	})

	pkts = item.PointObject.UpdatePackets(pkts, item.EntityId, LookBytes{})

	return pkts
}
