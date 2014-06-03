package gamerules

import (
	"fmt"
	"runtime"
	"testing"

	. "chunkymonkey/types"
)

const (
	plankId     = ItemTypeId(5)
	ironOreId   = ItemTypeId(15)
	ironIngotId = ItemTypeId(265)

	plankFuelTime = Ticks(300)
)

var (
	emptySlot = Slot{}
)

func callerDesc(skip int) string {
	if _, file, line, ok := runtime.Caller(skip + 1); ok {
		return fmt.Sprintf("%s:%d", file, line)
	}
	return "<unknown code>"
}

func checkLit(t *testing.T, furnace *FurnaceInventory, expectLit bool) {
	if expectLit != furnace.IsLit() {
		var msg string
		if expectLit {
			msg = "lit"
		} else {
			msg = "unlit"
		}
		t.Errorf("%s: Expected furnace to be %s", callerDesc(1), msg)
	}
}

func checkTx(t *testing.T, expected, txState TxState) {
	if expected != txState {
		t.Errorf("%s: Expected tx state=%v, got %v", callerDesc(1), expected, txState)
	}
}

func checkSlot(t *testing.T, expected, result Slot) {
	if !expected.Equals(&result) {
		t.Errorf("%s: Expected slot=%v, got %v", callerDesc(1), expected, result)
	}
}

// Utility type to keep track of the number of ticks that a furnace has been
// run for.
type furnaceRunner struct {
	t        *testing.T
	furnace  *FurnaceInventory
	curTicks Ticks
}

func (fr *furnaceRunner) runFor(numTicks Ticks) {
	for i := Ticks(0); i < numTicks; i++ {
		fr.furnace.Tick()
	}
	fr.curTicks += numTicks
}

func (fr *furnaceRunner) runUntil(absNumTicks Ticks) {
	numTicks := absNumTicks - fr.curTicks
	if numTicks < 0 {
		fr.t.Fatalf(
			"Was asked to run until %d ticks, but current ticks is %d",
			absNumTicks, fr.curTicks)
	}
	fr.runFor(numTicks)
}

// Creates a furnace, and loads with one plank and one iron ore, performing
// common tests along the way.
func loadedFurnace(t *testing.T, numFuel, numOre ItemCount) (furnace *FurnaceInventory, runner *furnaceRunner) {
	var txState TxState
	var click Click
	furnace = NewFurnaceInventory()

	checkLit(t, furnace, false)

	fuelInput := Slot{plankId, numFuel, 0}

	// Put a plank into the fuel slot.
	click = Click{
		furnaceSlotFuel,
		fuelInput,
		false, false,
		0,
		emptySlot,
	}
	txState = furnace.Click(&click)
	checkTx(t, TxStateAccepted, txState)
	checkSlot(t, emptySlot, click.Cursor)

	checkLit(t, furnace, false)

	// Put iron ore into the reagent slot.
	click = Click{
		furnaceSlotReagent,
		Slot{ironOreId, numOre, 0},
		false, false,
		0,
		emptySlot,
	}
	txState = furnace.Click(&click)
	checkTx(t, TxStateAccepted, txState)
	checkSlot(t, emptySlot, click.Cursor)

	// Should now be lit
	checkLit(t, furnace, true)

	// One unit of fuel from the fuel slot should have been consumed.
	expectedFuel := Slot{plankId, numFuel - 1, 0}
	expectedFuel.Normalize()
	checkSlot(t, expectedFuel, furnace.slots[furnaceSlotFuel])

	// The output should be empty.
	checkSlot(t, emptySlot, furnace.slots[furnaceSlotOutput])

	runner = &furnaceRunner{t, furnace, 0}

	return
}

func Test_FurnaceProducesIronIngot(t *testing.T) {
	furnace, runner := loadedFurnace(t, 1, 1)

	// reactionDuration-1 ticks later the output should still be empty.
	runner.runFor(reactionDuration - 1)
	checkSlot(t, emptySlot, furnace.slots[furnaceSlotOutput])

	// One tick later there should be an iron ingot present, and the furnace
	// should still be lit.
	runner.runFor(1)
	checkSlot(t, Slot{ironIngotId, 1, 0}, furnace.slots[furnaceSlotOutput])
}

func Test_FurnaceFinishBurning(t *testing.T) {
	furnace, runner := loadedFurnace(t, 1, 1)

	// 299 ticks later there should be an iron ingot present, and the furnace
	// should still be lit.
	runner.runFor(plankFuelTime - 1)
	checkSlot(t, Slot{ironIngotId, 1, 0}, furnace.slots[furnaceSlotOutput])
	checkLit(t, furnace, true)

	// One more tick later and the furnace should be unlit.
	runner.runFor(1)
	checkLit(t, furnace, false)
}

func Test_FurnaceBurnsMultipleFuel(t *testing.T) {
	furnace, runner := loadedFurnace(t, 2, 2)

	// 299 ticks later there should be an iron ingot present, and the furnace
	// should still be lit with one unit of fuel left to consume.
	runner.runUntil(plankFuelTime - 1)
	checkSlot(t, Slot{ironIngotId, 1, 0}, furnace.slots[furnaceSlotOutput])
	checkLit(t, furnace, true)
	checkSlot(t, Slot{plankId, 1, 0}, furnace.slots[furnaceSlotFuel])

	// One more tick later and the furnace should still be lit, as there is a
	// second unit of fuel to consume, leaving the fuel input slot empty.
	runner.runFor(1)
	checkLit(t, furnace, true)
	checkSlot(t, emptySlot, furnace.slots[furnaceSlotFuel])

	// Check that a single ingot is present just before the second one should be
	// produced.
	runner.runUntil(2*reactionDuration - 1)
	checkSlot(t, Slot{ironIngotId, 1, 0}, furnace.slots[furnaceSlotOutput])

	// The reaction should produce a second iron ingot after a total of 2 *
	// reactionDuration ticks.
	runner.runUntil(2 * reactionDuration)
	checkSlot(t, Slot{ironIngotId, 2, 0}, furnace.slots[furnaceSlotOutput])
	checkLit(t, furnace, true)

	// The furnace should be lit after a total of 599 ticks, just before the
	// second and last unit of fuel runs out.
	runner.runUntil(plankFuelTime*2 - 1)
	checkLit(t, furnace, true)

	// ... and then the furnace should be unlit after 600 ticks.
	runner.runUntil(plankFuelTime * 2)
	checkLit(t, furnace, false)
}
