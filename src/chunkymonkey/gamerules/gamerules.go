package gamerules

import (
	"os"

	"chunkymonkey/command"
	"chunkymonkey/permission"
)

// GameRules is a container type for block, item and recipe definitions.
var (
	Blocks           BlockTypeList
	Items            ItemTypeMap
	Recipes          *RecipeSet
	FurnaceReactions FurnaceData
	CommandFramework *command.CommandFramework
	Permissions      permission.IPermissions
)

func LoadGameRules(blocksDefFile, itemsDefFile, recipesDefFile, furnaceDefFile string) (err os.Error) {
	Blocks, err = LoadBlocksFromFile(blocksDefFile)
	if err != nil {
		return
	}

	Items, err = LoadItemTypesFromFile(itemsDefFile)
	if err != nil {
		return
	}

	Blocks.CreateBlockItemTypes(Items)

	Recipes, err = LoadRecipesFromFile(recipesDefFile, Items)
	if err != nil {
		return
	}

	FurnaceReactions, err = LoadFurnaceDataFromFile(furnaceDefFile)
	if err != nil {
		return
	}

	// TODO: Load the prefix from a config file
	CommandFramework = command.NewCommandFramework("/")

	Permissions, err = permission.LoadJsonPermission("./")
	if err != nil {
		return
	}

	return
}
