package inventory

import (
	"sync"

	"chunkymonkey/proto"
	"chunkymonkey/slot"
	. "chunkymonkey/types"
)

type IInventorySubscriber interface {
	SlotUpdate(slot *slot.Slot, slotId SlotId)
}

type Inventory struct {
	lock           sync.Mutex
	slots          []slot.Slot
	subscribers    map[IInventorySubscriber]bool
	onUnsubscribed func()
}

// Init initializes the inventory. onUnsubscribed is called in a new goroutine
// when the number of subscribers to the inventory reaches zero (but is not
// called initially).
func (inv *Inventory) Init(size int, onUnsubscribed func()) {
	inv.slots = make([]slot.Slot, size)
	for i := range inv.slots {
		inv.slots[i].Init()
	}
	inv.subscribers = make(map[IInventorySubscriber]bool)
	inv.onUnsubscribed = onUnsubscribed
}

func (inv *Inventory) AddSubscriber(subscriber IInventorySubscriber) {
	inv.lock.Lock()
	defer inv.lock.Unlock()
	inv.subscribers[subscriber] = true
}

func (inv *Inventory) RemoveSubscriber(subscriber IInventorySubscriber) {
	inv.lock.Lock()
	defer inv.lock.Unlock()
	inv.subscribers[subscriber] = false, false
	if len(inv.subscribers) == 0 && inv.onUnsubscribed != nil {
		go inv.onUnsubscribed()
	}
}

// StandardClick takes the default actions upon a click event from a player.
func (inv *Inventory) StandardClick(slotId SlotId, cursor *slot.Slot, rightClick bool, shiftClick bool) (accepted bool) {
	inv.lock.Lock()
	defer inv.lock.Unlock()

	if slotId < 0 || int(slotId) > len(inv.slots) {
		return false
	}

	clickedSlot := &inv.slots[slotId]

	// Apply the change.
	if cursor.Count == 0 {
		if rightClick {
			accepted = clickedSlot.Split(cursor)
		} else {
			accepted = clickedSlot.Swap(cursor)
		}
	} else {
		if rightClick {
			accepted = clickedSlot.AddOne(cursor)
		} else {
			accepted = clickedSlot.Add(cursor)
		}

		if !accepted {
			accepted = clickedSlot.Swap(cursor)
		}
	}

	// We send slot updates in case we have custom max counts that differ from
	// the client's idea.
	inv.slotUpdate(clickedSlot, slotId)

	return
}

// TakeOnlyClick only allows items to be taken from the slot, and it only
// allows the *whole* stack to be taken, otherwise no items are taken at all.
func (inv *Inventory) TakeOnlyClick(slotId SlotId, cursor *slot.Slot, rightClick, shiftClick bool) (accepted bool) {
	inv.lock.Lock()
	defer inv.lock.Unlock()

	if slotId < 0 || int(slotId) > len(inv.slots) {
		return false
	}

	clickedSlot := &inv.slots[slotId]

	// Apply the change.
	accepted = cursor.AddWhole(clickedSlot)

	// We send slot updates in case we have custom max counts that differ from
	// the client's idea.
	inv.slotUpdate(clickedSlot, slotId)

	return
}

// PutItem attempts to put the given item into the inventory.
func (inv *Inventory) PutItem(item *slot.Slot) {
	inv.lock.Lock()
	defer inv.lock.Unlock()
	// TODO optimize this algorithm, maybe by maintaining a map of non-full
	// slots containing an item of various item type IDs. Additionally, it
	// should prefer to put stackable items into stacks of the same type,
	// rather than in empty slots.
	for slotIndex := range inv.slots {
		if item.Count <= 0 {
			break
		}
		slot := &inv.slots[slotIndex]
		if slot.ItemType == nil || slot.ItemType == item.ItemType {
			if slot.Add(item) {
				inv.slotUpdate(slot, SlotId(slotIndex))
			}
		}
	}
	return
}

// WriteProtoSlots stores into the slots parameter the proto version of the
// item data in the inventory.
// Precondition: len(slots) == len(inv.slots)
func (inv *Inventory) writeProtoSlots(slots []proto.WindowSlot) {
	inv.lock.Lock()
	defer inv.lock.Unlock()
	for i := range inv.slots {
		src := &inv.slots[i]
		itemTypeId := ItemTypeIdNull
		if src.ItemType != nil {
			itemTypeId = src.ItemType.Id
		}
		slots[i] = proto.WindowSlot{
			ItemTypeId: itemTypeId,
			Count:      src.Count,
			Data:       src.Data,
		}
	}
}

// Send message about the slot change to the relevant places.
func (inv *Inventory) slotUpdate(slot *slot.Slot, slotId SlotId) {
	for subscriber := range inv.subscribers {
		subscriber.SlotUpdate(slot, slotId)
	}
}
