// Utility to perform basic checks on supplied data files for blocks, items and
// recipes.
package main

import (
	"flag"
	"fmt"
	"os"

	"chunkymonkey/gamerules"
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

var furnaceDefs = flag.String(
	"furnace", "furnace.json",
	"The JSON file containing furnace fuel and reaction definitions.")

var userDefs = flag.String(
	"users", "users.json",
	"The JSON file container user permissions.")

var groupDefs = flag.String(
	"groups", "groups.json",
	"The JSON file containing group permissions.")

func main() {
	err := gamerules.LoadGameRules(*blockDefs, *itemDefs, *recipeDefs, *furnaceDefs, *userDefs, *groupDefs)

	if err != nil {
		fmt.Fprintf(os.Stdout, "Error loading definitions: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("PASS")
}
