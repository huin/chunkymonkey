package chunkstore

import (
	"compress/gzip"
	"errors"
	"fmt"
	"os"
	"path"

	. "chunkymonkey/types"
	"chunkymonkey/util"
	"nbt"
)

type chunkStoreAlpha struct {
	worldPath string
}

// Creates an IChunkStore that reads the Minecraft Alpha world format.
func newChunkStoreAlpha(worldPath string, dimension DimensionId) (s *chunkStoreAlpha, err error) {
	// Don't know the dimension directory structure for alpha, but it's likely
	// not worth writing support for.

	s = &chunkStoreAlpha{
		worldPath: worldPath,
	}
	return s, nil
}

func (s *chunkStoreAlpha) chunkPath(chunkLoc ChunkXz) string {
	return path.Join(
		s.worldPath,
		base36Encode(int32(chunkLoc.X&63)),
		base36Encode(int32(chunkLoc.Z&63)),
		"c."+base36Encode(int32(chunkLoc.X))+"."+base36Encode(int32(chunkLoc.Z))+".dat")
}

func (s *chunkStoreAlpha) ReadChunk(chunkLoc ChunkXz) (reader IChunkReader, err error) {
	file, err := os.Open(s.chunkPath(chunkLoc))
	if err != nil {
		if os.IsNotExist(err) {
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
	reader, err = newNbtChunkReader(gzipReader)
	if err != nil {
		return
	}

	loadedLoc := reader.ChunkLoc()
	if loadedLoc.X != chunkLoc.X || loadedLoc.Z != chunkLoc.Z {
		err = errors.New(fmt.Sprintf(
			"Attempted to load chunk for %+v, but got chunk identified as %+v",
			chunkLoc,
			loadedLoc,
		))
	}

	return
}

func (s *chunkStoreAlpha) SupportsWrite() bool {
	return true
}

func (s *chunkStoreAlpha) Writer() IChunkWriter {
	return newNbtChunkWriter()
}

func (s *chunkStoreAlpha) WriteChunk(writer IChunkWriter) (err error) {
	nbtWriter, ok := writer.(*nbtChunkWriter)
	if !ok {
		return fmt.Errorf("%T is incorrect IChunkWriter implementation for %T", writer, s)
	}

	destName := s.chunkPath(writer.ChunkLoc())
	dirName, _ := path.Split(destName)

	if err = os.MkdirAll(dirName, 0777); err != nil {
		return
	}

	file, err := util.OpenFileUniqueName(destName, os.O_WRONLY, 0666)
	if err != nil {
		return
	}
	defer file.Close()

	gzipWriter := gzip.NewWriter(file)
	defer gzipWriter.Close()

	if err = nbt.Write(gzipWriter, nbtWriter.RootTag()); err != nil {
		return
	}

	// Atomically move the newly written file into place.
	return os.Rename(file.Name(), destName)
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
