package main

import (
	"compress/gzip"
	"fmt"
	"os"
	"strconv"
	"strings"

	. "chunkymonkey/types"
	"chunkymonkey/worldstore"
	"nbt"
)

func displayNbt(indentCount int, tag nbt.ITag) {
	if tag == nil {
		fmt.Print("<nil>\n")
		return
	}
	indent := strings.Repeat("  ", indentCount)
	switch t := tag.(type) {
	case *nbt.Compound:
		fmt.Print("Compound:\n")
		for name, subTag := range t.Tags {
			fmt.Printf("%s%#v: ", indent, name)
			displayNbt(indentCount+1, subTag)
		}
	case *nbt.List:
		fmt.Printf("List[%d]:\n", len(t.Value))
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
	default:
		fmt.Printf("<Unknown type>\n")
	}
}

func cmdLevel(args []string) (err error) {
	if len(args) != 1 {
		os.Stderr.WriteString("usage: " + os.Args[0] + " level <world path>\n")
		return
	}

	worldPath := args[0]

	worldStore, err := worldstore.LoadWorldStore(worldPath)
	if err != nil {
		return
	}

	displayNbt(1, worldStore.LevelData)

	return
}

func cmdChunk(args []string) (err error) {
	if len(args) < 3 || len(args) > 4 {
		os.Stderr.WriteString("usage: " + os.Args[0] + " chunk <world path> <chunk x> <chunk z> [dimension]\n")
		return
	}

	worldPath := args[0]
	x, err := strconv.Atoi(args[1])
	if err != nil {
		return
	}
	z, err := strconv.Atoi(args[2])
	if err != nil {
		return
	}

	dimension := DimensionNormal
	if len(args) >= 4 {
		var dimInt int
		dimInt, err = strconv.Atoi(args[3])
		if err != nil {
			return
		}
		dimension = DimensionId(dimInt)
	}

	worldStore, err := worldstore.LoadWorldStore(worldPath)
	if err != nil {
		return
	}

	chunkLoc := ChunkXz{ChunkCoord(x), ChunkCoord(z)}

	store, err := worldStore.ChunkStoreForDimension(dimension)
	if err != nil {
		return
	}

	chunkResult := <-store.ReadChunk(chunkLoc)
	if chunkResult.Err != nil {
		return chunkResult.Err
	}

	fmt.Printf("Chunk %#v data:\n", chunkLoc)
	displayNbt(1, chunkResult.Reader.RootTag())

	return
}

func cmdNbt(args []string) (err error) {
	if len(args) != 1 {
		os.Stderr.WriteString("usage: " + os.Args[0] + " nbt <NBT file path>\n")
		return
	}

	file, err := os.Open(args[0])
	if err != nil {
		return
	}
	defer file.Close()

	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		return
	}
	defer gzipReader.Close()

	namedTag, err := nbt.Read(gzipReader)
	if err != nil {
		return
	}

	displayNbt(1, namedTag)

	return
}

func main() {
	usage := func() {
		os.Stderr.WriteString("usage: " + os.Args[0] + " <command> [options]\n")
		os.Stderr.WriteString("Commands:\n")
		os.Stderr.WriteString("  level - Display level information.\n")
		os.Stderr.WriteString("  chunk - Display chunk information.\n")
		os.Stderr.WriteString("  nbt - Display NBT information from file.\n")
	}

	if len(os.Args) < 2 {
		usage()
		return
	}

	var err error

	switch command := os.Args[1]; command {
	case "level":
		err = cmdLevel(os.Args[2:])

	case "chunk":
		err = cmdChunk(os.Args[2:])

	case "nbt":
		err = cmdNbt(os.Args[2:])

	default:
		usage()
		err = fmt.Errorf("No such command: %q\n", command)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
