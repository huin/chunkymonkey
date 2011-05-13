package chunkstore

import (
	"io"
	"os"

	. "chunkymonkey/types"
	"chunkymonkey/nbt"
)

// Returned to chunks to pull their data from.
type chunkReader struct {
	chunkTag *nbt.NamedTag
}

// Load a chunk from its NBT representation
func newChunkReader(reader io.Reader) (r *chunkReader, err os.Error) {
	chunkTag, err := nbt.Read(reader)
	if err != nil {
		return
	}

	r = &chunkReader{
		chunkTag: chunkTag,
	}

	return
}

func (r *chunkReader) ChunkLoc() *ChunkXz {
	return &ChunkXz{
		X: ChunkCoord(r.chunkTag.Lookup("/Level/xPos").(*nbt.Int).Value),
		Z: ChunkCoord(r.chunkTag.Lookup("/Level/zPos").(*nbt.Int).Value),
	}
}

func (r *chunkReader) Blocks() []byte {
	return r.chunkTag.Lookup("/Level/Blocks").(*nbt.ByteArray).Value
}

func (r *chunkReader) BlockData() []byte {
	return r.chunkTag.Lookup("/Level/Data").(*nbt.ByteArray).Value
}

func (r *chunkReader) BlockLight() []byte {
	return r.chunkTag.Lookup("/Level/BlockLight").(*nbt.ByteArray).Value
}

func (r *chunkReader) SkyLight() []byte {
	return r.chunkTag.Lookup("/Level/SkyLight").(*nbt.ByteArray).Value
}

func (r *chunkReader) HeightMap() []byte {
	return r.chunkTag.Lookup("/Level/HeightMap").(*nbt.ByteArray).Value
}

func (r *chunkReader) GetRootTag() *nbt.NamedTag {
	return r.chunkTag
}
