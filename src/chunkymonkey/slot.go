package slot

import (
	"io"
	"os"

	"chunkymonkey/itemtype"
	"chunkymonkey/proto"
	. "chunkymonkey/types"
)

// Represents an inventory slot, e.g in a player's inventory, their cursor, a
// chest.
type Slot struct {
	// ItemType can be nil, specifically for empty slots.
	ItemType *itemtype.ItemType
	Count    ItemCount
	Data     ItemData
}

func (s *Slot) Init() {
	s.ItemType = nil
	s.Count = 0
	s.Data = 0
}

func (s *Slot) GetItemTypeId() (itemTypeId ItemTypeId) {
	if s.ItemType != nil {
		itemTypeId = s.ItemType.Id
	} else {
		itemTypeId = ItemTypeIdNull
	}
	return
}

func (s *Slot) GetAttr() (ItemTypeId, ItemCount, ItemData) {
	return s.GetItemTypeId(), s.Count, s.Data
}

func (s *Slot) SendUpdate(writer io.Writer, windowId WindowId, slotId SlotId) os.Error {
	return proto.WriteWindowSetSlot(writer, windowId, slotId, s.GetItemTypeId(), s.Count, s.Data)
}

func (s *Slot) SendEquipmentUpdate(writer io.Writer, entityId EntityId, slotId SlotId) os.Error {
	return proto.WriteEntityEquipment(writer, entityId, slotId, s.GetItemTypeId(), s.Data)
}

func (s *Slot) setCount(count ItemCount) {
	s.Count = count
	if s.Count == 0 {
		s.ItemType = nil
		s.Data = 0
	}
}

// Adds as many items from the passed slot to the destination (subject) slot as
// possible, depending on stacking allowances and item types etc.
// Returns true if slots changed as a result.
func (s *Slot) Add(src *Slot) (changed bool) {
	// NOTE: This code assumes that 2*ItemType.MaxStack will not overflow the
	// ItemCount type.
	if src.ItemType == nil {
		return
	}

	maxStack := src.ItemType.MaxStack

	if s.ItemType != nil {
		if s.ItemType != src.ItemType {
			return
		}
		if s.Data != src.Data {
			return
		}
	}

	if s.Count >= maxStack {
		return
	}

	s.ItemType = src.ItemType

	toTransfer := src.Count
	if s.Count+toTransfer > maxStack {
		toTransfer = maxStack - s.Count
	}
	if toTransfer != 0 {
		changed = true

		s.Data = src.Data

		s.setCount(s.Count + toTransfer)
		src.setCount(src.Count - toTransfer)
	}
	return
}

// AddWhole is similar to Add, but with the exception that if not all the items
// can be transferred, then none are transferred at all.
// Returns true if slots changed as a result.
func (s *Slot) AddWhole(src *Slot) (changed bool) {
	// NOTE: This code assumes that 2*ItemType.MaxStack will not overflow the
	// ItemCount type.
	if src.ItemType == nil {
		return
	}

	maxStack := src.ItemType.MaxStack

	if src.Count+s.Count > maxStack {
		return
	}

	return s.Add(src)
}

// Swaps the contents of the slots.
// Returns true if slots changed as a result.
func (s *Slot) Swap(src *Slot) (changed bool) {
	if s.ItemType != src.ItemType {
		tmp := src.ItemType
		src.ItemType = s.ItemType
		s.ItemType = tmp
		changed = true
	}

	if s.Count != src.Count {
		s.Count ^= src.Count
		src.Count ^= s.Count
		s.Count ^= src.Count
		changed = true
	}

	if s.Data != src.Data {
		s.Data ^= src.Data
		src.Data ^= s.Data
		s.Data ^= src.Data
		changed = true
	}

	return
}

// Splits the contents of the subject slot (s) into half, half remaining in s,
// and half moving to src (odd amounts put the spare item into the src slot).
// If src is not empty, then this does nothing.
// Returns true if slots changed as a result.
func (s *Slot) Split(src *Slot) (changed bool) {
	if s.Count == 0 || src.Count != 0 {
		return
	}

	changed = true
	src.ItemType = s.ItemType
	src.Data = s.Data

	count := s.Count >> 1
	odd := s.Count & 1

	src.setCount(count + odd)
	s.setCount(count)

	return
}

// Takes one item count from src and adds it to the subject s. It does nothing
// if the items in the slots are not compatible.
// Returns true if slots changed as a result.
func (s *Slot) AddOne(src *Slot) (changed bool) {
	if src.ItemType == nil {
		return
	}
	maxStack := src.ItemType.MaxStack

	if s.ItemType != src.ItemType && s.ItemType != nil {
		return
	}
	if src.Data != s.Data {
		return
	}

	if s.Count >= maxStack {
		return
	}

	changed = true
	s.setCount(s.Count + 1)
	s.ItemType = src.ItemType
	s.Data = src.Data
	src.setCount(src.Count - 1)

	return
}

// Decrement destroys one item count from the subject slot.
func (s *Slot) Decrement() (changed bool) {
	if s.Count <= 0 {
		return
	}

	s.setCount(s.Count - 1)
	changed = true
	return
}
