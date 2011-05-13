package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"chunkymonkey/worldstore"
	. "chunkymonkey/types"
	"chunkymonkey/nbt"
)

func usage() {
	os.Stderr.WriteString("usage: " + os.Args[0] + " <world path>\n")
	flag.PrintDefaults()
}

func displayNbt(indentCount int, tag nbt.Tag) {
	indent := strings.Repeat("  ", indentCount)
	switch t := tag.(type) {
	case *nbt.Compound:
		fmt.Print("Compound:\n")
		for name, subTag := range t.Tags {
			fmt.Printf("%s%#v: ", indent, name)
			displayNbt(indentCount+1, subTag)
		}
	case *nbt.List:
		fmt.Print("List:\n")
		for index, subTag := range t.Value {
			fmt.Printf("%s[%d]: ", indent, index)
			displayNbt(indentCount+1, subTag)
		}
	case *nbt.Byte:
		fmt.Printf("Byte: %d\n", t.Value)
	case *nbt.Short:
		fmt.Printf("Short: %d\n", t.Value)
	case *nbt.Int:
		fmt.Printf("Int: %d\n", t.Value)
	case *nbt.Long:
		fmt.Printf("Long: %d\n", t.Value)
	case *nbt.Float:
		fmt.Printf("Float: %f\n", t.Value)
	case *nbt.Double:
		fmt.Printf("Double: %f\n", t.Value)
	case *nbt.ByteArray:
		fmt.Printf("ByteArray: %d bytes long\n", len(t.Value))
	case *nbt.String:
		fmt.Printf("String: %#v\n", t.Value)
	case *nbt.NamedTag:
		displayNbt(indentCount, t.Tag)
	default:
		fmt.Printf("<Unknown type>\n")
	}
}

func main() {
	flag.Usage = usage
	flag.Parse()

	if flag.NArg() != 1 {
		flag.Usage()
		os.Exit(1)
	}

	worldPath := flag.Arg(0)

	worldStore, err := worldstore.LoadWorldStore(worldPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading world: %s\n", err)
		return
	}

	fmt.Print("Level data:\n")
	displayNbt(1, worldStore.LevelData)

	chunkReader, err := worldStore.ChunkStore.LoadChunk(&ChunkXz{0, 0})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading chunk: %s\n", err)
		return
	}
	fmt.Print("Chunk {0, 0} data:\n")
	displayNbt(1, chunkReader.GetRootTag())
}
