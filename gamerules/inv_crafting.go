package gamerules

import (
	. "chunkymonkey/types"
)

const (
	playerInvCraftWidth     = 2
	playerInvCraftHeight    = 2
	workbenchInvCraftWidth  = 3
	workbenchInvCraftHeight = 3
)

// Inventory with extended function to perform crafting. It assumes that slot 0
// is the output, and that the remaining slots are inputs.
type CraftingInventory struct {
	Inventory
	width, height int
	recipes       RecipeSetMatcher
}

func (inv *CraftingInventory) init(width, height int) {
	inv.Inventory.Init(1 + width*height)
	inv.width = width
	inv.height = height
	inv.recipes.Init(Recipes)
}

// InitWorkbenchInventory initializes inv as a 2x2 player crafting inventory.
func (inv *CraftingInventory) InitPlayerCraftingInventory() {
	inv.init(
		playerInvCraftWidth,
		playerInvCraftHeight,
	)
}

// NewWorkbenchInventory creates a 3x3 workbench crafting inventory.
func NewWorkbenchInventory() *CraftingInventory {
	inv := new(CraftingInventory)
	inv.init(
		workbenchInvCraftWidth,
		workbenchInvCraftHeight,
	)
	return inv
}

// Click handles window clicks from a user with special handling for crafting.
func (inv *CraftingInventory) Click(click *Click) (txState TxState) {
	if click.SlotId == 0 {
		// Player may only *take* the *whole* stack from the output slot.
		txState = inv.Inventory.TakeOnlyClick(click)
	} else {
		// Player may interact with the input slots like any other slot.
		txState = inv.Inventory.Click(click)
	}

	if txState == TxStateRejected {
		return
	}

	if click.SlotId == 0 {
		// Player took items from the output slot. Subtract 1 count from each
		// non-empty input slot.
		for i := 1; i < len(inv.slots); i++ {
			inv.slots[i].Decrement()
			inv.slotUpdate(&inv.slots[i], SlotId(i))
		}
	}

	// Match recipe and set output slot.
	inv.slots[0] = inv.recipes.Match(inv.width, inv.height, inv.slots[1:])
	inv.slotUpdate(&inv.slots[0], 0)

	return
}

// TakeAllItems empties the inventory, and returns all items that were inside
// it inside a slice of Slots.
func (inv *CraftingInventory) TakeAllItems() (items []Slot) {
	// The output slot gets emptied, the only items that are to be taken are
	// those in the input slots.
	output := &inv.slots[0]
	output.Clear()
	inv.slotUpdate(output, 0)

	return inv.Inventory.TakeAllItems()
}
