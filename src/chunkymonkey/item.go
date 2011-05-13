package item

import (
	"io"
	"os"

	"chunkymonkey/entity"
	"chunkymonkey/itemtype"
	"chunkymonkey/physics"
	"chunkymonkey/proto"
	"chunkymonkey/slot"
	. "chunkymonkey/types"
)

type Item struct {
	entity.Entity
	slot.Slot
	physObj     physics.PointObject
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
	item.physObj.Init(position, velocity)
	return
}

func (item *Item) GetEntity() *entity.Entity {
	return &item.Entity
}

func (item *Item) GetSlot() *slot.Slot {
	return &item.Slot
}

func (item *Item) GetItemTypeId() ItemTypeId {
	return item.ItemType.Id
}

func (item *Item) GetCount() ItemCount {
	return item.Count
}

func (item *Item) SetCount(count ItemCount) {
	item.Count = count
}

func (item *Item) GetPosition() *AbsXyz {
	return &item.physObj.Position
}

func (item *Item) SendSpawn(writer io.Writer) (err os.Error) {
	// TODO pass uses value instead of 0
	err = proto.WriteItemSpawn(
		writer, item.EntityId, item.ItemType.Id, item.Count, 0,
		&item.physObj.LastSentPosition, &item.orientation)
	if err != nil {
		return
	}

	err = proto.WriteEntityVelocity(writer, item.EntityId, &item.physObj.LastSentVelocity)
	if err != nil {
		return
	}

	return
}

func (item *Item) SendUpdate(writer io.Writer) (err os.Error) {
	if err = proto.WriteEntity(writer, item.Entity.EntityId); err != nil {
		return
	}

	err = item.physObj.SendUpdate(writer, item.Entity.EntityId, &LookBytes{0, 0})

	return
}

func (item *Item) Tick(blockQuery physics.BlockQueryFn) (leftBlock bool) {
	return item.physObj.Tick(blockQuery)
}
