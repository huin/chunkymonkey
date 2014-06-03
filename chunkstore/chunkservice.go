package chunkstore

import (
	"log"

	. "chunkymonkey/types"
)

type readRequest struct {
	chunkLoc     ChunkXz
	responseChan chan<- ChunkReadResult
}

type IChunkStoreForeground interface {
	ReadChunk(chunkLoc ChunkXz) (reader IChunkReader, err error)
	SupportsWrite() bool
	Writer() IChunkWriter
	WriteChunk(writer IChunkWriter) error
}

// ChunkService adapts an IChunkStoreForeground (which can only be accessed
// from one goroutine) to an IChunkStore.
type ChunkService struct {
	store  IChunkStoreForeground
	reads  chan readRequest
	writes chan IChunkWriter
}

func NewChunkService(store IChunkStoreForeground) (s *ChunkService) {
	return &ChunkService{
		store:  store,
		reads:  make(chan readRequest),
		writes: make(chan IChunkWriter),
	}
}

func (s *ChunkService) Serve() {
	for {
		select {
		case request := <-s.reads:
			reader, err := s.store.ReadChunk(request.chunkLoc)
			request.responseChan <- ChunkReadResult{reader, err}
		case writer := <-s.writes:
			if err := s.store.WriteChunk(writer); err != nil {
				log.Printf("Could not write chunk at %#v: %v", writer.ChunkLoc(), err)
			}
		}
	}
}

func (s *ChunkService) ReadChunk(chunkLoc ChunkXz) <-chan ChunkReadResult {
	responseChan := make(chan ChunkReadResult)

	s.reads <- readRequest{
		chunkLoc:     chunkLoc,
		responseChan: responseChan,
	}

	return responseChan
}

func (s *ChunkService) SupportsWrite() bool {
	return s.store.SupportsWrite()
}

func (s *ChunkService) Writer() IChunkWriter {
	return s.store.Writer()
}

func (s *ChunkService) WriteChunk(writer IChunkWriter) {
	s.writes <- writer
}
