// Code in this file covers the logic and data for crafting items from other
// items in the inventory or workbench.

package gamerules

import (
	"fmt"
)

const (
	fnv1_32_offset = 2166136261
	fnv1_32_prime  = 16777619
)

const (
	maxRecipeWidth  = 3
	maxRecipeHeight = 3
)

type Recipe struct {
	Comment string
	Width   byte
	Height  byte
	Input   []Slot
	Output  Slot
}

func (r *Recipe) match(width, height byte, slots []Slot, indices []int) (isMatch bool) {
	isMatch = false
	if width != r.Width || height != r.Height {
		return
	}
	for i := range r.Input {
		inSlot := &slots[indices[i]]
		rSlot := &r.Input[i]
		if inSlot.ItemTypeId != rSlot.ItemTypeId || inSlot.Data != rSlot.Data {
			return
		}

	}
	isMatch = true
	return
}

func (r *Recipe) hash() (hash uint32) {
	indices := make([]int, len(r.Input))
	for i := range r.Input {
		indices[i] = i
	}
	return inputHash(r.Input, indices)
}

func (r *Recipe) check() error {
	for i := range r.Input {
		slot := &r.Input[i]
		if !slot.IsValidType() {
			return fmt.Errorf("Recipe %q input slot %d has unknown item type %d", r.Comment, i, slot.ItemTypeId)
		}
	}
	if !r.Output.IsValidType() {
		return fmt.Errorf("Recipe %q output slot has unknown item type %d", r.Comment, r.Output.ItemTypeId)
	}
	return nil
}

func inputHash(slots []Slot, indices []int) (hash uint32) {
	// Hash based on FNV-1a.
	hash = fnv1_32_offset
	for _, index := range indices {
		slot := &slots[index]

		// Hash the lower 16 bits of the item type ID.
		itemTypeId := slot.ItemTypeId
		hash ^= uint32(itemTypeId & 0xff)
		hash *= fnv1_32_prime
		hash ^= uint32((itemTypeId >> 8) & 0xff)
		hash *= fnv1_32_prime

		// Hash the data value
		hash ^= uint32(slot.Data & 0xff)
		hash *= fnv1_32_prime
	}

	return
}

type RecipeSet struct {
	recipes []Recipe

	// Recipe by inputs hash.
	recipeHash map[uint32][]*Recipe
}

func (r *RecipeSet) init() error {
	r.recipeHash = make(map[uint32][]*Recipe)
	for i := range r.recipes {
		recipe := &r.recipes[i]
		hash := recipe.hash()
		bucket := r.recipeHash[hash]
		bucket = append(bucket, recipe)
		r.recipeHash[hash] = bucket
	}

	return r.check()
}

// check checks all the recipes to ensure that they seem consistent, i.e item
// type IDs exist, etc.
func (r *RecipeSet) check() error {
	for i := range r.recipes {
		if err := r.recipes[i].check(); err != nil {
			return err
		}
	}
	return nil
}

// RecipeSetMatcher looks up recipes within a RecipeSet, an instance must be
// used from a single goroutine.
type RecipeSetMatcher struct {
	recipes *RecipeSet

	// slotBuf is used in searching for a match. Having it in the struct saves
	// reallocation per call to Match().
	indicesArray [maxRecipeWidth * maxRecipeHeight]int
}

func (r *RecipeSetMatcher) Init(recipes *RecipeSet) {
	r.recipes = recipes
}

// Match looks for a matching recipe for the input slots, and returns a Slot
// with the result of any matching recipe. output.ItemType==nil and
// output.Count==0 if nothing matched).
//
// The order of slots is left to right, then top to bottom.
//
// Precondition: len(slots) == width * height
func (r *RecipeSetMatcher) Match(width, height int, slots []Slot) (output Slot) {

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
			if slots[curIndex].Count > 0 {
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

	// Make used rectangle into a linear array of []int indices.
	indices := r.indicesArray[:widthUsed*heightUsed]
	outIndex := 0
	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			inIndex := y*width + x
			indices[outIndex] = inIndex
			outIndex++
		}
	}

	hash := inputHash(slots, indices)

	bucket, ok := r.recipes.recipeHash[hash]

	if !ok {
		return
	}

	// Find the matching recipe, if any.
	for i := range bucket {
		recipe := bucket[i]
		if recipe.match(byte(widthUsed), byte(heightUsed), slots, indices) {
			// Found matching recipe.
			output = recipe.Output
		}
	}

	return
}
