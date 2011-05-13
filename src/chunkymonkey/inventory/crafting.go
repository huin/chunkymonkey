package inventory

import (
	"chunkymonkey/recipe"
	"chunkymonkey/slot"
	. "chunkymonkey/types"
)

const (
	workbenchInvCraftWidth  = 3
	workbenchInvCraftHeight = 3
)

// Inventory with extended function to perform crafting. It assumes that slot 0
// is the output, and that the remaining slots are inputs.
type CraftingInventory struct {
	Inventory
	width, height int
	recipes       *recipe.RecipeSet
}

func (inv *CraftingInventory) Init(width, height int, onUnsubscribed func(), recipes *recipe.RecipeSet) {
	inv.Inventory.Init(1+width*height, onUnsubscribed)
	inv.width = width
	inv.height = height
	inv.recipes = recipes
}

// Click handles window clicks from a user with special handling for crafting.
func (inv *CraftingInventory) Click(slotId SlotId, cursor *slot.Slot, rightClick bool, shiftClick bool) (accepted bool) {
	if slotId == 0 {
		// Player may only *take* the *whole* stack from the output slot.
		accepted = inv.Inventory.TakeOnlyClick(
			slotId, cursor, rightClick, shiftClick)
	} else {
		// Player may interact with the input slots like any other slot.
		accepted = inv.Inventory.StandardClick(
			slotId, cursor, rightClick, shiftClick)
	}

	if !accepted {
		return
	}

	inv.lock.Lock()
	defer inv.lock.Unlock()

	if slotId == 0 {
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

type WorkbenchInventory struct {
	CraftingInventory
}

func NewWorkbenchInventory(onUnsubscribed func(), recipes *recipe.RecipeSet) (inv *WorkbenchInventory) {
	inv = new(WorkbenchInventory)
	inv.CraftingInventory.Init(
		workbenchInvCraftWidth,
		workbenchInvCraftHeight,
		onUnsubscribed,
		recipes,
	)
	return
}

// TakeAllItems empties the inventory, and returns all items that were inside
// it inside a slice of Slots.
func (inv *WorkbenchInventory) TakeAllItems() (items []slot.Slot) {
	inv.lock.Lock()
	defer inv.lock.Unlock()

	items = make([]slot.Slot, 0, len(inv.slots)-1)

	// The output slot gets emptied, the only items that are to be ejected are
	// those in the input slots.
	output := &inv.slots[0]
	output.Init()
	inv.slotUpdate(output, 0)

	for i := 1; i < len(inv.slots); i++ {
		curSlot := &inv.slots[i]
		if curSlot.Count > 0 {
			var taken slot.Slot
			taken.Init()
			taken.Swap(curSlot)
			items = append(items, taken)
			inv.slotUpdate(curSlot, SlotId(i))
		}
	}

	return
}
