package gamerules

import (
	"chunkymonkey/permission"
)

// GameRules is a container type for block, item and recipe definitions.
var (
	Blocks           BlockTypeList
	Items            ItemTypeMap
	Recipes          *RecipeSet
	FurnaceReactions FurnaceData
	// TODO: Commands should maybe be accessible via IGame.
	CommandFramework ICommandFramework
	Permissions      permission.IPermissions
)

func LoadGameRules(blocksDefFile, itemsDefFile, recipesDefFile, furnaceDefFile, userDefFile, groupDefFile string) (err error) {
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

	Permissions, err = permission.LoadJsonPermissionFromFiles(userDefFile, groupDefFile)
	if err != nil {
		return
	}

	// Ensure that the block aspects are configured correctly, now that
	// everything is loaded.
	for i := range Blocks {
		if err = Blocks[i].Aspect.Check(); err != nil {
			return
		}
	}

	return
}
