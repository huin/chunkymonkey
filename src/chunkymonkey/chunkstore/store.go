package chunkstore

import (
	"fmt"
	"os"

	. "chunkymonkey/types"
	"nbt"
)

type ChunkResult struct {
	Reader IChunkReader
	Err    os.Error
}

type IChunkStore interface {
	// Serve() serves LoadChunk() requests in the foreground.
	Serve()

	LoadChunk(chunkLoc ChunkXz) (result <-chan ChunkResult)
}

type IChunkReader interface {
	// Returns the chunk location.
	ChunkLoc() ChunkXz

	// Returns the block IDs in the chunk.
	Blocks() []byte

	// Returns the block data in the chunk.
	BlockData() []byte

	// Returns the block light data in the chunk.
	BlockLight() []byte

	// Returns the sky light data in the chunk.
	SkyLight() []byte

	// Returns the height map data in the chunk.
	HeightMap() []byte

	// Return a list of the entities (items, mobs) within the chunk.
	Entities() []nbt.ITag

	// For low-level NBT access. Not for regular use. It's possible that this
	// might return nil if the underlying system doesn't use NBT.
	RootTag() nbt.ITag
}

// Given the NamedTag for a level.dat, returns an appropriate
// IChunkStoreForeground.
func ChunkStoreForLevel(worldPath string, levelData nbt.ITag, dimension DimensionId) (store IChunkStoreForeground, err os.Error) {
	versionTag, ok := levelData.Lookup("Data/version").(*nbt.Int)

	if !ok {
		store = newChunkStoreAlpha(worldPath, dimension)
	} else {
		switch version := versionTag.Value; version {
		case 19132:
			store = newChunkStoreBeta(worldPath, dimension)
		default:
			err = UnknownLevelVersion(version)
		}
	}

	return
}

type UnknownLevelVersion int32

func (err UnknownLevelVersion) String() string {
	return fmt.Sprintf("Unknown level version %d", err)
}

type NoSuchChunkError bool

func (err NoSuchChunkError) String() string {
	return "Chunk does not exist."
}
