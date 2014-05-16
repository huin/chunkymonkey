package chunkstore

import (
	"fmt"

	"chunkymonkey/gamerules"
	. "chunkymonkey/types"
	"nbt"
)

type ChunkReadResult struct {
	Reader IChunkReader
	Err    error
}

type IChunkStore interface {
	// Serve() serves requests in the foreground.
	Serve()

	ReadChunk(chunkLoc ChunkXz) (result <-chan ChunkReadResult)
	SupportsWrite() bool
	Writer() IChunkWriter

	// Submits the set chunk data for writing. The chunk writer must not be
	// altered any further after calling this.
	WriteChunk(writer IChunkWriter)
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

	// Return a slice of the entities (items, mobs) within the chunk.
	Entities() []gamerules.INonPlayerEntity

	// Return a slice of the tile entities (chests, furnaces, etc.) within the
	// chunk.
	TileEntities() []gamerules.ITileEntity

	// For low-level NBT access. Not for regular use. It's possible that this
	// might return nil if the underlying system doesn't use NBT.
	RootTag() nbt.ITag
}

// IChunkWriter is the interface for objects that accept chunk data and write
// it. These are created by IChunkWriteableStore for use by a chunk to store a
// snapshot of its current state into. The Set* functions make copies of the
// data passed in, so that the original data structures passed in can be
// modified upon return.
type IChunkWriter interface {
	// ChunkLoc returns the chunk location.
	ChunkLoc() ChunkXz

	// SetChunkLoc sets the chunk location.
	SetChunkLoc(loc ChunkXz)

	// SetBlocks sets the block IDs in the chunk.
	SetBlocks(blocks []byte)

	// SetBlockData sets the block data in the chunk.
	SetBlockData(blockData []byte)

	// SetBlockLight sets the block light data in the chunk.
	SetBlockLight(blockLight []byte)

	// SetSkyLight sets the sky light data in the chunk.
	SetSkyLight(skyLight []byte)

	// SetHeightMap sets the height map data in the chunk.
	SetHeightMap(heightMap []byte)

	// SetEntities sets a list of the entities (items, mobs) within the chunk.
	SetEntities(entities map[EntityId]gamerules.INonPlayerEntity)

	// SetTileEntities sets a list of the tile entities (chests, furnaces, etc.)
	// within the chunk.
	SetTileEntities(tileEntities map[BlockIndex]gamerules.ITileEntity)
}

// Given the NamedTag for a level.dat, returns an appropriate
// IChunkStoreForeground.
func ChunkStoreForLevel(worldPath string, levelData nbt.ITag, dimension DimensionId) (store IChunkStoreForeground, err error) {
	versionTag, ok := levelData.Lookup("Data/version").(*nbt.Int)

	if !ok {
		store, err = newChunkStoreAlpha(worldPath, dimension)
	} else {
		switch version := versionTag.Value; version {
		case 19132:
			store, err = newChunkStoreBeta(worldPath, dimension)
		default:
			err = UnknownLevelVersion(version)
		}
	}

	return
}

type UnknownLevelVersion int32

func (err UnknownLevelVersion) Error() string {
	return fmt.Sprintf("Unknown level version %d", err)
}

type NoSuchChunkError bool

func (err NoSuchChunkError) Error() string {
	return "Chunk does not exist."
}
