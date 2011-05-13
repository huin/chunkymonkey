// Utility to perform basic checks on supplied data files for blocks, items and
// recipes.
package main

import (
	"flag"
	"fmt"
	"os"

	"chunkymonkey/block"
	"chunkymonkey/itemtype"
	"chunkymonkey/recipe"
)

var blockDefs = flag.String(
	"blocks", "blocks.json",
	"The JSON file containing block type definitions.")

var itemDefs = flag.String(
	"items", "items.json",
	"The JSON file containing item type definitions.")

var recipeDefs = flag.String(
	"recipes", "recipes.json",
	"The JSON file containing recipe definitions.")

func main() {
	var err os.Error

	blockTypes, err := block.LoadBlocksFromFile(*blockDefs)
	if err != nil {
		fmt.Fprintf(os.Stdout, "Error loading block definitions: %v\n", err)
		os.Exit(1)
	}

	itemTypes, err := itemtype.LoadItemTypesFromFile(*itemDefs)
	if err != nil {
		fmt.Fprintf(os.Stdout, "Error loading item definitions: %v\n", err)
		os.Exit(1)
	}

	blockTypes.CreateBlockItemTypes(itemTypes)

	_, err = recipe.LoadRecipesFromFile(*recipeDefs, itemTypes)
	if err != nil {
		fmt.Fprintf(os.Stdout, "Error loading recipe definitions: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("PASS datatests\n")
}
