package inventory

import (
	"bytes"
	"io"
	"os"

	"chunkymonkey/proto"
	"chunkymonkey/slot"
	. "chunkymonkey/types"
)

// IWindow is the interface on to types that represent a view on to multiple
// inventories.
type IWindow interface {
	GetWindowId() WindowId
	Click(slotId SlotId, cursor *slot.Slot, rightClick bool, shiftClick bool) (accepted bool)
	WriteWindowOpen(writer io.Writer) (err os.Error)
	WriteWindowItems(writer io.Writer) (err os.Error)
	Finalize(sendClosePacket bool)
}

// IWindowViewer is the required interface of types that wish to receive packet
// updates from changes to inventories viewed inside a window. Typically
// *player.Player implements this.
type IWindowViewer interface {
	TransmitPacket(packet []byte)
}

// inventoryView provides a single mapping between a window view onto an
// inventory at a particular slot range inside the window.
type inventoryView struct {
	window    *Window
	inventory *Inventory
	startSlot SlotId
	endSlot   SlotId
}

func (iv *inventoryView) Init(window *Window, inventory *Inventory, startSlot SlotId, endSlot SlotId) {
	iv.window = window
	iv.inventory = inventory
	iv.startSlot = startSlot
	iv.endSlot = endSlot
	iv.inventory.AddSubscriber(iv)
}

func (iv *inventoryView) Finalize() {
	iv.inventory.RemoveSubscriber(iv)
}

// Implementing IInventorySubscriber - relays inventory changes to the viewer
// of the window.
func (iv *inventoryView) SlotUpdate(slot *slot.Slot, slotId SlotId) {
	buf := &bytes.Buffer{}
	slot.SendUpdate(buf, iv.window.windowId, iv.startSlot+slotId)
	iv.window.viewer.TransmitPacket(buf.Bytes())
}

// Window represents the common base behaviour of an inventory window. It acts
// as a view onto multiple Inventories.
type Window struct {
	windowId  WindowId
	invTypeId InvTypeId
	viewer    IWindowViewer
	views     []inventoryView
	title     string
	numSlots  int
}

// Init initializes a window as a view onto the given inventories.
func (w *Window) Init(windowId WindowId, invTypeId InvTypeId, viewer IWindowViewer, title string, inventories ...*Inventory) {
	w.windowId = windowId
	w.invTypeId = invTypeId
	w.viewer = viewer
	w.title = title

	w.views = make([]inventoryView, len(inventories))
	startSlot := 0
	for index, inv := range inventories {
		endSlot := startSlot + len(inv.slots)
		w.views[index].Init(w, inv, SlotId(startSlot), SlotId(endSlot))
		startSlot = endSlot
	}
	w.numSlots = startSlot

	return
}

func (w *Window) GetWindowId() WindowId {
	return w.windowId
}

// Finalize cleans up, subscriber information so that the window can be
// properly garbage collected. This should be called when the window is thrown
// away.
func (w *Window) Finalize(sendClosePacket bool) {
	for index := range w.views {
		w.views[index].Finalize()
	}
	if sendClosePacket {
		buf := &bytes.Buffer{}
		proto.WriteWindowClose(buf, w.windowId)
	}
}

// WriteWindowOpen writes a packet describing the window to the writer.
func (w *Window) WriteWindowOpen(writer io.Writer) (err os.Error) {
	err = proto.WriteWindowOpen(writer, w.windowId, w.invTypeId, w.title, byte(w.numSlots))
	return
}

// WriteWindowItems writes a packet describing the window contents to the
// writer. It assumes that any required locks on the inventories are held.
func (w *Window) WriteWindowItems(writer io.Writer) (err os.Error) {
	items := make([]proto.WindowSlot, w.numSlots)

	for i := range w.views {
		view := &w.views[i]
		view.inventory.writeProtoSlots(items[view.startSlot:view.endSlot])
	}

	err = proto.WriteWindowItems(writer, w.windowId, items)
	return
}
