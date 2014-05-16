package gamerules

import (
	"nbt"
)

const (
	dispenserInvWidth  = 3
	dispenserInvHeight = 3
)

type DispenserInventory struct {
	Inventory
}

// NewDispenserInventory creates a 3x3 dispenser inventory.
func NewDispenserInventory() (inv *DispenserInventory) {
	inv = new(DispenserInventory)
	inv.Inventory.Init(dispenserInvWidth * dispenserInvHeight)
	return inv
}

func (inv *DispenserInventory) MarshalNbt(tag *nbt.Compound) (err error) {
	tag.Set("id", &nbt.String{"Trap"})
	return inv.Inventory.MarshalNbt(tag)
}
