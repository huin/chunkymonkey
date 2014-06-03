package gamerules

import (
	. "chunkymonkey/types"
)

func makeVoidAspect() (aspect IBlockAspect) {
	return &VoidAspect{}
}

// Behaviour of a "void" block which has no behaviour.
type VoidAspect struct{}

func (aspect *VoidAspect) setAttrs(blockAttrs *BlockAttrs) {
}

func (aspect *VoidAspect) Name() string {
	return "Void"
}

func (aspect *VoidAspect) Check() error {
	return nil
}

func (aspect *VoidAspect) Hit(instance *BlockInstance, player IPlayerClient, digStatus DigStatus) (destroyed bool) {
	destroyed = false
	return
}

func (aspect *VoidAspect) Interact(instance *BlockInstance, player IPlayerClient) {
}

func (aspect *VoidAspect) InventoryClick(instance *BlockInstance, player IPlayerClient, click *Click) {
}

func (aspect *VoidAspect) InventoryUnsubscribed(instance *BlockInstance, player IPlayerClient) {
}

func (aspect *VoidAspect) Destroy(instance *BlockInstance) {
}

func (aspect *VoidAspect) Tick(instance *BlockInstance) bool {
	return false
}
