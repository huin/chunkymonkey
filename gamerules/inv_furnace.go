package gamerules

import (
	"errors"
	"github.com/huin/chunkymonkey/nbt"
	. "github.com/huin/chunkymonkey/types"
)

const (
	furnaceSlotReagent = SlotId(0)
	furnaceSlotFuel    = SlotId(1)
	furnaceSlotOutput  = SlotId(2)
	furnaceNumSlots    = 3

	reactionDuration = Ticks(185)
	maxFuelPrg       = 255
)

type FurnaceInventory struct {
	Inventory
	burnTimeMax Ticks
	burnTime    Ticks
	cookTime    Ticks

	lastCurFuel           PrgBarValue
	lastReactionRemaining PrgBarValue
	ticksSinceUpdate      int
}

// NewFurnaceInventory creates a furnace inventory.
func NewFurnaceInventory() (inv *FurnaceInventory) {
	inv = &FurnaceInventory{
		burnTimeMax: 0,
		burnTime:    0,
		cookTime:    reactionDuration,
	}
	inv.Inventory.Init(furnaceNumSlots)
	return
}

func (inv *FurnaceInventory) UnmarshalNbt(tag *nbt.Compound) (err error) {
	if err = inv.Inventory.UnmarshalNbt(tag); err != nil {
		return
	}

	if burnTimeTag, ok := tag.Lookup("BurnTime").(*nbt.Short); !ok {
		return errors.New("Bad or missing BurnTime tag in Furnace NBT")
	} else {
		inv.burnTime = Ticks(burnTimeTag.Value)

		// We don't know what the burnTimeMax was, as it is not stored, so taking
		// BurnTime for this value as well.
		inv.burnTimeMax = inv.burnTime
	}

	if cookTimeTag, ok := tag.Lookup("CookTime").(*nbt.Short); !ok {
		return errors.New("Bad or missing CookTime tag in Furnace NBT")
	} else {
		inv.cookTime = Ticks(cookTimeTag.Value)
	}

	return nil
}

func (inv *FurnaceInventory) MarshalNbt(tag *nbt.Compound) (err error) {
	tag.Set("id", &nbt.String{"Furnace"})
	tag.Set("BurnTime", &nbt.Short{int16(inv.burnTime)})
	tag.Set("CookTime", &nbt.Short{int16(inv.cookTime)})
	return inv.Inventory.MarshalNbt(tag)
}

func (inv *FurnaceInventory) Click(click *Click) (txState TxState) {

	switch click.SlotId {
	case furnaceSlotReagent:
		slotBefore := inv.slots[furnaceSlotReagent]

		txState = inv.Inventory.Click(click)

		slotAfter := &inv.slots[furnaceSlotReagent]

		// If the reagent type changes, the reaction restarts.
		if slotBefore.ItemTypeId != slotAfter.ItemTypeId || slotBefore.Data != slotAfter.Data {
			inv.cookTime = reactionDuration
		}
	case furnaceSlotFuel:
		cursorItemId := click.Cursor.ItemTypeId
		_, cursorIsFuel := FurnaceReactions.Fuels[cursorItemId]
		if cursorIsFuel || click.Cursor.IsEmpty() {
			txState = inv.Inventory.Click(click)
		}
	case furnaceSlotOutput:
		// Player may only *take* the *whole* stack from the output slot.
		txState = inv.Inventory.TakeOnlyClick(click)
	}

	inv.stateCheck()

	inv.sendProgressUpdates()

	return
}

func (inv *FurnaceInventory) stateCheck() {
	reagentSlot := &inv.slots[furnaceSlotReagent]
	fuelSlot := &inv.slots[furnaceSlotFuel]
	outputSlot := &inv.slots[furnaceSlotOutput]

	reaction, haveReagent := FurnaceReactions.Reactions[reagentSlot.ItemTypeId]
	fuelTicks, haveFuel := FurnaceReactions.Fuels[fuelSlot.ItemTypeId]

	// Work out if the output slot is ready for items to be produced from the
	// reaction.
	var outputReady bool
	if outputSlot.ItemTypeId != 0 {
		itemType := outputSlot.ItemType()
		maxStack := MaxStackDefault
		if itemType != nil {
			maxStack = itemType.MaxStack
		}
		// Output has items in.
		if !haveReagent {
			outputReady = false
		} else if outputSlot.Count >= maxStack {
			// Output is full.
			outputReady = false
		} else if outputSlot.ItemTypeId != reaction.Output || outputSlot.Data != reaction.OutputData {
			// Output has a different type from the reaction.
			outputReady = false
		} else {
			// Output contains compatible items and is not full.
			outputReady = true
		}
	} else {
		// Output is empty.
		outputReady = true
	}

	if inv.burnTime <= 0 {
		if haveReagent && haveFuel && outputReady {
			// Everything is in place, light the furnace by consuming one unit of
			// fuel.
			fuelSlot.Decrement()
			inv.burnTimeMax = fuelTicks
			inv.burnTime = fuelTicks
			inv.slotUpdate(fuelSlot, furnaceSlotFuel)
		} else {
			inv.cookTime = reactionDuration
			inv.burnTimeMax = 0
			inv.burnTime = 0
		}
	}

	if inv.burnTime > 0 {
		// Furnace is lit.
		if !outputReady {
			inv.cookTime = reactionDuration
		} else if haveReagent && inv.cookTime == 0 {
			// One reaction complete.
			itemCreated := Slot{
				ItemTypeId: reaction.Output,
				Count:      1,
				Data:       reaction.OutputData,
			}
			inv.cookTime = reactionDuration

			outputSlot.AddOne(&itemCreated)
			inv.slotUpdate(outputSlot, furnaceSlotOutput)
			reagentSlot.Decrement()
			inv.slotUpdate(reagentSlot, furnaceSlotReagent)
		}
	}
}

// sendProgressUpdates sends an update to the subscriber. Not every time,
// however - to cut down on unnecessary communication.
func (inv *FurnaceInventory) sendProgressUpdates() {
	if inv.subscriber == nil {
		return
	}

	inv.ticksSinceUpdate++
	if inv.ticksSinceUpdate > 5 || !inv.IsLit() {
		inv.ticksSinceUpdate = 0

		curFuelPrg := PrgBarValue(0)
		if inv.burnTimeMax != 0 {
			curFuelPrg = PrgBarValue((maxFuelPrg * inv.burnTime) / inv.burnTimeMax)
		}
		if inv.lastCurFuel != curFuelPrg {
			inv.lastCurFuel = curFuelPrg
			inv.subscriber.ProgressUpdate(PrgBarIdFurnaceFire, curFuelPrg)
		}

		curReactionRemaining := PrgBarValue(reactionDuration - inv.cookTime)
		if inv.lastReactionRemaining != curReactionRemaining {
			inv.lastReactionRemaining = curReactionRemaining
			inv.subscriber.ProgressUpdate(PrgBarIdFurnaceProgress, curReactionRemaining)
		}
	}
}

func (inv *FurnaceInventory) IsLit() bool {
	return inv.burnTime > 0
}

// Tick runs the furnace for a single tick.
func (inv *FurnaceInventory) Tick() {
	if inv.burnTime > 0 {
		inv.burnTime--

		if inv.cookTime > 0 {
			inv.cookTime--
		}

		inv.stateCheck()

		inv.sendProgressUpdates()
	}
}
