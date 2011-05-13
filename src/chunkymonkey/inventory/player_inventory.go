package inventory

import (
	"io"
	"os"

	"chunkymonkey/recipe"
	"chunkymonkey/slot"
	. "chunkymonkey/types"
)

const (
	playerInvCraftWidth  = 2
	playerInvCraftHeight = 2
	playerInvCraftNum    = 1 + playerInvCraftWidth + playerInvCraftHeight
	playerInvCraftStart  = 0
	playerInvCraftEnd    = playerInvCraftStart + playerInvCraftNum

	playerInvArmorNum   = 4
	playerInvArmorStart = playerInvCraftEnd
	playerInvArmorEnd   = playerInvArmorStart + playerInvArmorNum

	playerInvMainNum   = 3 * 9
	playerInvMainStart = playerInvArmorEnd
	playerInvMainEnd   = playerInvMainStart + playerInvMainNum

	playerInvHoldingNum   = 9
	playerInvHoldingStart = playerInvMainEnd
	playerInvHoldingEnd   = playerInvHoldingStart + playerInvHoldingNum

	playerInvSize = playerInvCraftNum + playerInvArmorNum + playerInvMainNum + playerInvHoldingNum
)

type PlayerInventory struct {
	Window
	entityId     EntityId
	crafting     CraftingInventory
	armor        Inventory
	main         Inventory
	holding      Inventory
	holdingIndex SlotId
}

// Init initializes PlayerInventory.
// entityId - The EntityId of the player who holds the inventory.
func (w *PlayerInventory) Init(entityId EntityId, viewer IWindowViewer, recipes *recipe.RecipeSet) {
	w.entityId = entityId

	w.crafting.Init(playerInvCraftWidth, playerInvCraftHeight, nil, recipes)
	w.armor.Init(playerInvArmorNum, nil)
	w.main.Init(playerInvMainNum, nil)
	w.holding.Init(playerInvHoldingNum, nil)
	w.Window.Init(
		WindowIdInventory,
		// Note that we have no known value for invTypeId - but it's only used
		// in WriteWindowOpen which isn't used for PlayerInventory.
		-1,
		viewer,
		"Inventory",
		&w.crafting.Inventory,
		&w.armor,
		&w.main,
		&w.holding,
	)
	w.holdingIndex = 0
}

// NewWindow creates a new window for the player that shares its player
// inventory sections with `w`. Returns nil for unrecognized inventory types.
// TODO implement more inventory types.
func (w *PlayerInventory) NewWindow(invTypeId InvTypeId, windowId WindowId, inventory interface{}) IWindow {
	switch invTypeId {
	case InvTypeIdWorkbench:
		if crafting, ok := inventory.(*WorkbenchInventory); ok && crafting != nil {
			return NewWorkbenchWindow(
				w.entityId, w.viewer,
				windowId,
				crafting, &w.main, &w.holding)
		}
	default:
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
func (w *PlayerInventory) HeldItem() (slot *slot.Slot, slotId SlotId) {
	slotId = w.holdingIndex
	slot = &w.holding.slots[w.holdingIndex]
	return
}

// TakeOneHeldItem takes one item from the stack of items the player is holding
// and puts it in `into`. It does nothing if the player is holding no items, or
// if `into` cannot take any items of that type.
func (w *PlayerInventory) TakeOneHeldItem(into *slot.Slot) {
	slot := &w.holding.slots[w.holdingIndex]
	if into.AddOne(slot) {
		w.holding.slotUpdate(slot, w.holdingIndex)
	}
}

// Writes packets for other players to see the equipped items.
func (w *PlayerInventory) SendFullEquipmentUpdate(writer io.Writer) (err os.Error) {
	slot, _ := w.HeldItem()
	err = slot.SendEquipmentUpdate(writer, w.entityId, 0)
	if err != nil {
		return
	}

	for i := range w.armor.slots {
		err = w.armor.slots[i].SendEquipmentUpdate(writer, w.entityId, SlotId(i+1))
		if err != nil {
			return
		}
	}
	return
}

// PutItem attempts to put the item stack into the player's inventory. The item
// will be modified as a result.
func (w *PlayerInventory) PutItem(item *slot.Slot) {
	w.holding.PutItem(item)
	w.main.PutItem(item)
	return
}

func (w *PlayerInventory) Click(slotId SlotId, cursor *slot.Slot, rightClick bool, shiftClick bool) (accepted bool) {
	switch {
	case slotId < 0:
		return false
	case slotId < playerInvCraftEnd:
		accepted = w.crafting.Click(
			slotId-playerInvCraftStart,
			cursor, rightClick, shiftClick)
	case slotId < playerInvArmorEnd:
		// TODO - handle armor
		return false
	case slotId < playerInvMainEnd:
		accepted = w.main.StandardClick(
			slotId-playerInvMainStart,
			cursor, rightClick, shiftClick)
	case slotId < playerInvHoldingEnd:
		accepted = w.holding.StandardClick(
			slotId-playerInvHoldingStart,
			cursor, rightClick, shiftClick)
	}
	return
}
