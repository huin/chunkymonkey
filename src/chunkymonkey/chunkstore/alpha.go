package chunkstore

import (
	"compress/gzip"
	"fmt"
	"os"
	"path"

	. "chunkymonkey/types"
)

type chunkStoreAlpha struct {
	worldPath string
}

// Creates a ChunkStore that reads the Minecraft Alpha world format.
func NewChunkStoreAlpha(worldPath string) ChunkStore {
	return &chunkStoreAlpha{
		worldPath: worldPath,
	}
}

func (s *chunkStoreAlpha) chunkPath(chunkLoc *ChunkXz) string {
	return path.Join(
		s.worldPath,
		base36Encode(int32(chunkLoc.X&63)),
		base36Encode(int32(chunkLoc.Z&63)),
		"c."+base36Encode(int32(chunkLoc.X))+"."+base36Encode(int32(chunkLoc.Z))+".dat")
}

// Load a chunk from its NBT representation
func (s *chunkStoreAlpha) LoadChunk(chunkLoc *ChunkXz) (reader ChunkReader, err os.Error) {
	if err != nil {
		return
	}

	file, err := os.Open(s.chunkPath(chunkLoc))
	if err != nil {
		if sysErr, ok := err.(*os.SyscallError); ok && sysErr.Errno == os.ENOENT {
			err = NoSuchChunkError(false)
		}
		return
	}
	defer file.Close()

	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		return
	}
	defer gzipReader.Close()
	reader, err = newChunkReader(gzipReader)
	if err != nil {
		return
	}

	loadedLoc := reader.ChunkLoc()
	if loadedLoc.X != chunkLoc.X || loadedLoc.Z != chunkLoc.Z {
		err = os.NewError(fmt.Sprintf(
			"Attempted to load chunk for %+v, but got chunk identified as %+v",
			chunkLoc,
			loadedLoc,
		))
	}

	return
}

// Utility functions:

func base36Encode(n int32) (s string) {
	alphabet := "0123456789abcdefghijklmnopqrstuvwxyz"
	negative := false

	if n < 0 {
		n = -n
		negative = true
	}
	if n == 0 {
		return "0"
	}

	for n != 0 {
		i := n % int32(len(alphabet))
		n /= int32(len(alphabet))
		s = string(alphabet[i:i+1]) + s
	}
	if negative {
		s = "-" + s
	}
	return
}
