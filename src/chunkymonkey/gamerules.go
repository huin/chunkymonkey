package gamerules

import (
	"os"

	"chunkymonkey/block"
	"chunkymonkey/itemtype"
	"chunkymonkey/recipe"
)

// GameRules is a container type for block, item and recipe definitions.
type GameRules struct {
	BlockTypes block.BlockTypeList
	ItemTypes  itemtype.ItemTypeMap
	Recipes    *recipe.RecipeSet
}

func LoadGameRules(blocksDefFile, itemsDefFile, recipesDefFile string) (rules *GameRules, err os.Error) {
	blockTypes, err := block.LoadBlocksFromFile(blocksDefFile)
	if err != nil {
		return
	}

	itemTypes, err := itemtype.LoadItemTypesFromFile(itemsDefFile)
	if err != nil {
		return
	}

	blockTypes.CreateBlockItemTypes(itemTypes)

	recipes, err := recipe.LoadRecipesFromFile(recipesDefFile, itemTypes)
	if err != nil {
		return
	}

	rules = &GameRules{
		BlockTypes: blockTypes,
		ItemTypes:  itemTypes,
		Recipes:    recipes,
	}

	return
}
