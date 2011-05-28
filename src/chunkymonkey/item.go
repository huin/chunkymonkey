package item

import (
	"io"
	"os"

	"chunkymonkey/itemtype"
	"chunkymonkey/physics"
	"chunkymonkey/proto"
	"chunkymonkey/slot"
	. "chunkymonkey/types"
)

type Item struct {
	EntityId
	slot.Slot
	physics.PointObject
	orientation OrientationBytes
}

func NewItem(itemType *itemtype.ItemType, count ItemCount, data ItemData, position *AbsXyz, velocity *AbsVelocity) (item *Item) {
	item = &Item{
		// TODO proper orientation
		orientation: OrientationBytes{0, 0, 0},
	}
	item.Slot.ItemType = itemType
	item.Slot.Count = count
	item.Slot.Data = data
	item.PointObject.Init(position, velocity)
	return
}

func (item *Item) GetSlot() *slot.Slot {
	return &item.Slot
}

func (item *Item) GetItemTypeId() ItemTypeId {
	return item.ItemType.Id
}

func (item *Item) SendSpawn(writer io.Writer) (err os.Error) {
	// TODO pass uses value instead of 0
	err = proto.WriteItemSpawn(
		writer, item.EntityId, item.ItemType.Id, item.Slot.Count, 0,
		&item.PointObject.LastSentPosition, &item.orientation)
	if err != nil {
		return
	}

	err = proto.WriteEntityVelocity(writer, item.EntityId, &item.PointObject.LastSentVelocity)
	if err != nil {
		return
	}

	return
}

func (item *Item) SendUpdate(writer io.Writer) (err os.Error) {
	if err = proto.WriteEntity(writer, item.EntityId); err != nil {
		return
	}

	err = item.PointObject.SendUpdate(writer, item.EntityId, &LookBytes{0, 0})

	return
}
