package gamerules

import (
	"nbt"
)

const (
	chestInvWidth  = 9
	chestInvHeight = 3
)

type ChestInventory struct {
	Inventory
}

// NewChestInventory creates a 9x3 chest inventory.
func NewChestInventory() (inv *ChestInventory) {
	inv = new(ChestInventory)
	inv.Inventory.Init(chestInvWidth * chestInvHeight)
	return inv
}

func (inv *ChestInventory) MarshalNbt(tag *nbt.Compound) (err error) {
	tag.Set("id", &nbt.String{"Furnace"})
	return inv.Inventory.MarshalNbt(tag)
}
