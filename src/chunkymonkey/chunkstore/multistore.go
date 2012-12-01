package chunkstore

import (
	"errors"

	. "chunkymonkey/types"
)

// MultiStore provides the ability to load a chunk from one or more potential
// sources of chunk data. The primary purpose of this is to read from a
// persistant store first, then fall back to generating a chunk if the
// persistant store does not have it. MultiStore implements IChunkStore.
type MultiStore struct {
	readStores []IChunkStore
	writeStore IChunkStore
}

func NewMultiStore(readStores []IChunkStore, writeStore IChunkStore) *MultiStore {
	s := &MultiStore{
		readStores: readStores,
		writeStore: writeStore,
	}

	return s
}

func (s *MultiStore) ReadChunk(chunkLoc ChunkXz) (reader IChunkReader, err error) {
	for _, store := range s.readStores {
		result := <-store.ReadChunk(chunkLoc)

		if result.Err == nil {
			return result.Reader, result.Err
		} else {
			if _, ok := result.Err.(NoSuchChunkError); ok {
				// Fall through to next chunk store.
				continue
			}
			return nil, result.Err
		}
	}

	return nil, NoSuchChunkError(false)
}

func (s *MultiStore) SupportsWrite() bool {
	return s.writeStore != nil && s.writeStore.SupportsWrite()
}

func (s *MultiStore) Writer() IChunkWriter {
	if s.writeStore != nil {
		return s.writeStore.Writer()
	}
	return nil
}

func (s *MultiStore) WriteChunk(writer IChunkWriter) error {
	if s.writeStore == nil {
		return errors.New("writes not supported")
	}
	s.writeStore.WriteChunk(writer)
	return nil
}
