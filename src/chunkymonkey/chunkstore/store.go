package chunkstore

import (
	"fmt"
	"os"

	. "chunkymonkey/types"
	"chunkymonkey/nbt"
)

type ChunkStore interface {
	LoadChunk(chunkLoc *ChunkXz) (reader ChunkReader, err os.Error)
}

type ChunkReader interface {
	// Returns the chunk location.
	ChunkLoc() *ChunkXz

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

	// For low-level NBT access. Not for regular use. It's possible that this
	// might return nil if the underlying system doesn't use NBT.
	GetRootTag() *nbt.NamedTag
}

// Given the NamedTag for a level.dat, returns an appropriate ChunkStore.
func ChunkStoreForLevel(worldPath string, levelData *nbt.NamedTag) (store ChunkStore, err os.Error) {
	versionTag, ok := levelData.Lookup("/Data/version").(*nbt.Int)

	if !ok {
		store = NewChunkStoreAlpha(worldPath)
	} else {
		switch version := versionTag.Value; version {
		case 19132:
			store = NewChunkStoreBeta(worldPath)
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
