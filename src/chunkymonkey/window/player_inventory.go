package window

import (
	"errors"
	"fmt"
	"io"

	"chunkymonkey/gamerules"
	. "chunkymonkey/types"
	"nbt"
)

const (
	playerInvArmorNum   = 4
	playerInvMainNum    = 3 * 9
	playerInvHoldingNum = 9
)

type PlayerInventory struct {
	Window
	entityId     EntityId
	crafting     gamerules.CraftingInventory
	armor        gamerules.Inventory
	main         gamerules.Inventory
	holding      gamerules.Inventory
	holdingIndex SlotId
}

// Init initializes PlayerInventory.
// entityId - The EntityId of the player who holds the inventory.
func (w *PlayerInventory) Init(entityId EntityId, viewer IWindowViewer) {
	w.entityId = entityId

	w.crafting.InitPlayerCraftingInventory()
	w.armor.Init(playerInvArmorNum)
	w.main.Init(playerInvMainNum)
	w.holding.Init(playerInvHoldingNum)
	w.Window.Init(
		WindowIdInventory,
		// Note that we have no known value for invTypeId - but it's only used
		// in WriteWindowOpen which isn't used for PlayerInventory.
		-1,
		viewer,
		"Inventory",
		&w.crafting,
		// TODO Create and use special inventory type for armor slots only.
		&w.armor,
		&w.main,
		&w.holding,
	)
	w.holdingIndex = 0
}

// Resubscribe should be called when another window has potentially been
// subscribed to the player's inventory, and subsequently closed.
func (w *PlayerInventory) Resubscribe() {
	for i := range w.Window.views {
		w.Window.views[i].Resubscribe()
	}
}

// NewWindow creates a new window for the player that shares its player
// inventory sections with `w`. Returns nil for unrecognized inventory types.
// TODO implement more inventory types.
func (w *PlayerInventory) NewWindow(invTypeId InvTypeId, windowId WindowId, inv IInventory) IWindow {
	switch invTypeId {
	case InvTypeIdWorkbench:
		return NewWindow(
			windowId, invTypeId, w.viewer, "Crafting",
			inv, &w.main, &w.holding)
	case InvTypeIdChest:
		return NewWindow(
			windowId, invTypeId, w.viewer, "Chest",
			inv, &w.main, &w.holding)
	case InvTypeIdFurnace:
		return NewWindow(
			windowId, invTypeId, w.viewer, "Furnace",
			inv, &w.main, &w.holding)
	}
	return nil
}

// SetHolding chooses the held item (0-8). Out of range values have no effect.
func (w *PlayerInventory) SetHolding(holding SlotId) {
	if holding >= 0 && holding < SlotId(playerInvHoldingNum) {
		w.holdingIndex = holding
	}
}

// HeldItem returns the slot that is the current "held" item.
// TODO need any changes to the held item slot to create notifications to
// players.
func (w *PlayerInventory) HeldItem() (slot gamerules.Slot, slotId SlotId) {
	return w.holding.Slot(w.holdingIndex), w.holdingIndex
}

// TakeOneHeldItem takes one item from the stack of items the player is holding
// and puts it in `into`. It does nothing if the player is holding no items, or
// if `into` cannot take any items of that type.
func (w *PlayerInventory) TakeOneHeldItem(into *gamerules.Slot) {
	w.holding.TakeOneItem(w.holdingIndex, into)
}

// Writes packets for other players to see the equipped items.
func (w *PlayerInventory) SendFullEquipmentUpdate(writer io.Writer) (err error) {
	slot, _ := w.HeldItem()
	err = slot.SendEquipmentUpdate(writer, w.entityId, 0)
	if err != nil {
		return
	}

	numArmor := w.armor.NumSlots()
	for i := SlotId(0); i < numArmor; i++ {
		slot := w.armor.Slot(i)
		err = slot.SendEquipmentUpdate(writer, w.entityId, SlotId(i+1))
		if err != nil {
			return
		}
	}
	return
}

// PutItem attempts to put the item stack into the player's inventory. The item
// will be modified as a result.
func (w *PlayerInventory) PutItem(item *gamerules.Slot) {
	w.holding.PutItem(item)
	w.main.PutItem(item)
	return
}

// CanTakeItem returns true if it can take at least one item from the passed
// Slot.
func (w *PlayerInventory) CanTakeItem(item *gamerules.Slot) bool {
	return w.holding.CanTakeItem(item) || w.main.CanTakeItem(item)
}

func (w *PlayerInventory) UnmarshalNbt(tag nbt.ITag) (err error) {
	if tag == nil {
		return
	}

	list, ok := tag.(*nbt.List)
	if !ok {
		return errors.New("bad inventory - not a list")
	}

	for _, slotTagITag := range list.Value {
		slotTag, ok := slotTagITag.(*nbt.Compound)
		if !ok {
			return errors.New("non-compound found for slot in player inventory")
		}
		var slotIdTag *nbt.Byte
		if slotIdTag, ok = slotTag.Lookup("Slot").(*nbt.Byte); !ok {
			return errors.New("slot ID not a byte")
		}
		slotId := SlotId(slotIdTag.Value)
		// The mapping order in NBT differs from that used in the window protocol.
		// 0-8 = holding
		// 9-35 = main inventory
		// 100-103 = armor slots (in order: feet, legs, torso, head)
		// Crafting slots appear not to be present on the official server, as the
		// items are ejected into the world when the client disconnects.
		var inv gamerules.IInventory
		var invSlotId SlotId
		switch {
		case 0 <= slotId && slotId < playerInvHoldingNum:
			inv = &w.holding
			invSlotId = slotId
		case playerInvHoldingNum <= slotId && slotId < (playerInvHoldingNum+playerInvMainNum):
			inv = &w.main
			invSlotId = slotId - playerInvHoldingNum
		case 100 <= slotId && slotId <= 103:
			inv = &w.armor
			invSlotId = 103 - slotId
		default:
			return fmt.Errorf("inventory slot %d out of range", slotId)
		}
		if err = inv.SlotUnmarshalNbt(slotTag, invSlotId); err != nil {
			return
		}
	}

	return
}

func (w *PlayerInventory) MarshalNbt(tag *nbt.Compound) (err error) {
	slots := make([]nbt.ITag, 0, 0)

	// Add the holding inventory
	for i := 0; i < int(w.holding.NumSlots()); i++ {
		slot := w.holding.Slot(SlotId(i))
		if !slot.IsEmpty() {
			slotTag := nbt.NewCompound()
			slotTag.Set("Slot", &nbt.Byte{int8(i)})
			if err = slot.MarshalNbt(slotTag); err != nil {
				return
			}
			slots = append(slots, slotTag)
		}
	}

	// Add the main inventory
	for i := 0; i < int(w.main.NumSlots()); i++ {
		slot := w.main.Slot(SlotId(i))
		if !slot.IsEmpty() {
			slotTag := nbt.NewCompound()
			slotTag.Set("Slot", &nbt.Byte{int8(i + playerInvHoldingNum)})
			if err = slot.MarshalNbt(slotTag); err != nil {
				return
			}
			slots = append(slots, slotTag)
		}
	}

	// Add the armor inventory
	for i := 0; i < int(w.armor.NumSlots()); i++ {
		slot := w.armor.Slot(SlotId(i))
		if !slot.IsEmpty() {
			slotTag := nbt.NewCompound()
			slotTag.Set("Slot", &nbt.Byte{int8(i + 100)})
			if err = slot.MarshalNbt(slotTag); err != nil {
				return
			}
			slots = append(slots, slotTag)
		}
	}

	tag.Set("Inventory", &nbt.List{nbt.TagCompound, slots})

	return nil
}
