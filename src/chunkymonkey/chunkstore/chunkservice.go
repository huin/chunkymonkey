package chunkstore

import (
	"os"

	. "chunkymonkey/types"
)

type request struct {
	chunkLoc     ChunkXz
	responseChan chan<- ChunkResult
}

type IChunkStoreForeground interface {
	LoadChunk(chunkLoc ChunkXz) (reader IChunkReader, err os.Error)
}

// ChunkService adapts an IChunkStoreForeground (which can only be accessed
// from one goroutine) to an IChunkStore.
type ChunkService struct {
	store    IChunkStoreForeground
	requests chan request
}

func NewChunkService(store IChunkStoreForeground) (s *ChunkService) {
	return &ChunkService{
		store:    store,
		requests: make(chan request),
	}
}

func (s *ChunkService) Serve() {
	for {
		request := <-s.requests
		reader, err := s.store.LoadChunk(request.chunkLoc)
		request.responseChan <- ChunkResult{reader, err}
	}
}

func (s *ChunkService) LoadChunk(chunkLoc ChunkXz) <-chan ChunkResult {
	responseChan := make(chan ChunkResult)

	s.requests <- request{
		chunkLoc:     chunkLoc,
		responseChan: responseChan,
	}

	return responseChan
}
