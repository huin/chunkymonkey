package main

import (
	"log"
	"os"

	"chunkymonkey/gamerules"
)

func dumpBlocks(filename string) (err error) {
	outputFile, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0640)
	if err != nil {
		return
	}
	defer outputFile.Close()

	blocks, _ := loadBlocks()
	err = gamerules.SaveBlockDefs(outputFile, blocks)

	return
}

func loadBlocks() (blocks gamerules.BlockTypeList, err error) {
	file, err := os.Open("blocks.json")
	if err != nil {
		return
	}
	defer file.Close()

	return gamerules.LoadBlockDefs(file)
}

func main() {
	err := dumpBlocks("blocks2.json")
	if err != nil {
		log.Printf("Error: %v", err)
	}
}
