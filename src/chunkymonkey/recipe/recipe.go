// The recipe package covers the logic and data for crafting items from other
// items in the inventory or workbench.
package recipe

import (
	"chunkymonkey/itemtype"
	"chunkymonkey/slot"
	. "chunkymonkey/types"
)

const (
	maxRecipeWidth  = 3
	maxRecipeHeight = 3
)

type RecipeSlot struct {
	Type *itemtype.ItemType
	Data ItemData
}

type Recipe struct {
	Comment     string
	Width       byte
	Height      byte
	Input       []RecipeSlot
	Output      RecipeSlot
	OutputCount ItemCount
}

func (r *Recipe) match(width, height byte, slots []*slot.Slot) (isMatch bool) {
	isMatch = false
	if width != r.Width || height != r.Height {
		return
	}
	for i := range r.Input {
		inSlot := slots[i]
		rSlot := &r.Input[i]
		if inSlot.ItemType != rSlot.Type {
			return
		}

	}
	isMatch = true
	return
}

type RecipeSet struct {
	Recipes []Recipe
	// TODO Recipe index data for fast lookup by recipe dimensions, or maybe by
	// some sort of recipe input hash?
}

// Match looks for a matching recipe for the input slots, and returns a Slot
// with the result of any matching recipe. output.ItemType==nil and
// output.Count==0 if nothing matched).
//
// The order of slots is left to right, then top to bottom.
//
// Precondition: len(slots) == width * height
func (r *RecipeSet) Match(width, height int, slots []slot.Slot) (output slot.Slot) {

	// Precondition check.
	if width*height != len(slots) || width > maxRecipeWidth || height > maxRecipeHeight {
		return
	}

	minX, minY := int(width), int(height)
	maxX, maxY := 0, 0
	curIndex := 0
	// Find the position and size of the smallest rectangle that contains all
	// non-empty slots.
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			hasItem := slots[curIndex].ItemType != nil
			if hasItem {
				if x < minX {
					minX = x
				}
				if y < minY {
					minY = y
				}
				if x > maxX {
					maxX = x
				}
				if y > maxY {
					maxY = y
				}
			}
			curIndex++
		}
	}

	widthUsed := 1 + maxY - minY
	heightUsed := 1 + maxX - minX
	// Empty grid.
	if widthUsed <= 0 || heightUsed <= 0 {
		return
	}

	// Make used rectangle into a linear array of []*Slot.
	rectSlots := make([]*slot.Slot, widthUsed*heightUsed)
	outIndex := 0
	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			inIndex := y*width + x
			rectSlots[outIndex] = &slots[inIndex]
			outIndex++
		}
	}

	// Find the matching recipe, if any.
	for i := range r.Recipes {
		recipe := &r.Recipes[i]
		if recipe.match(byte(widthUsed), byte(heightUsed), rectSlots) {
			output.ItemType = recipe.Output.Type
			output.Count = recipe.OutputCount
			output.Data = recipe.Output.Data
		}
	}

	return
}
