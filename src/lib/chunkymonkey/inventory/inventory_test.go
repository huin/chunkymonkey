package inventory

import (
	"testing"
)

func TestInventory_Init(t *testing.T) {
	var inv Inventory
	inv.Init(10, nil)

	for i, slot := range inv.slots {
		if slot.ItemType != nil || slot.Count != 0 || slot.Data != 0 {
			t.Errorf("Slot %d not initialized: %+v", i, slot)
		}
	}
}
