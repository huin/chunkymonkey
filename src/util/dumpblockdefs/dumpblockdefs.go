package main

import (
	"log"
	"os"

	"chunkymonkey/block"
)

func dumpBlocks(filename string) (err os.Error) {
	outputFile, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0640)
	if err != nil {
		return
	}
	defer outputFile.Close()

	blocks, _ := loadBlocks()
	err = block.SaveBlockDefs(outputFile, blocks)

	return
}

func loadBlocks() (blocks block.BlockTypeList, err os.Error) {
	file, err := os.Open("blocks.json")
	if err != nil {
		return
	}
	defer file.Close()

	return block.LoadBlockDefs(file)
}

func main() {
	err := dumpBlocks("blocks2.json")
	if err != nil {
		log.Printf("Error: %v", err)
	}
}
