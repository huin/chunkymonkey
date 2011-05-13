package chunk

import (
	"log"

	"chunkymonkey/gamerules"
	. "chunkymonkey/interfaces"
	"chunkymonkey/chunkstore"
	. "chunkymonkey/types"
)

// ChunkManager contains all chunks and can look them up
type ChunkManager struct {
	game       IGame
	chunkStore chunkstore.ChunkStore
	gameRules  *gamerules.GameRules
	chunks     map[uint64]*Chunk
}

func NewChunkManager(chunkStore chunkstore.ChunkStore, game IGame) *ChunkManager {
	return &ChunkManager{
		game:       game,
		chunkStore: chunkStore,
		gameRules:  game.GetGameRules(),
		chunks:     make(map[uint64]*Chunk),
	}
}

// Get a chunk at given coordinates
func (mgr *ChunkManager) Get(loc *ChunkXz) (c IChunk) {
	var ok bool
	var chunk *Chunk
	key := loc.ChunkKey()

	if chunk, ok = mgr.chunks[key]; ok {
		c = chunk
		return
	}

	chunkReader, err := mgr.chunkStore.LoadChunk(loc)
	if err != nil {
		if _, ok := err.(chunkstore.NoSuchChunkError); !ok {
			log.Printf("ChunkManager.Get(%+v): %s", loc, err.String())
			return
		} else {
			// Chunk doesn't exist in store.
			// TODO Generate new chunks.
			return
		}
	}

	chunk = newChunkFromReader(chunkReader, mgr)
	c = chunk

	// Notify neighbouring chunk(s) (if any) that this chunk is now active, and
	// notify this chunk of its active neighbours
	linkNeighbours := func(from ChunkSideDir) {
		dx, dz := from.GetDxz()
		loc := ChunkXz{
			X: loc.X + dx,
			Z: loc.Z + dz,
		}
		neighbour, exists := mgr.chunks[loc.ChunkKey()]
		if exists {
			to := from.GetOpposite()
			chunk.Enqueue(func(_ IChunk) {
				chunk.sideCacheSetNeighbour(from, neighbour)
			})
			neighbour.Enqueue(func(_ IChunk) {
				neighbour.sideCacheSetNeighbour(to, chunk)
			})
		}
	}
	// TODO corresponding unlinking when a chunk is unloaded
	linkNeighbours(ChunkSideEast)
	linkNeighbours(ChunkSideSouth)
	linkNeighbours(ChunkSideWest)
	linkNeighbours(ChunkSideNorth)

	mgr.chunks[key] = chunk
	return
}

func (mgr *ChunkManager) ChunksActive() <-chan IChunk {
	c := make(chan IChunk)
	go func() {
		for _, chunk := range mgr.chunks {
			c <- chunk
		}
		close(c)
	}()
	return c
}

// Return a channel to iterate over all chunks within a chunk's radius
func (mgr *ChunkManager) ChunksInRadius(loc *ChunkXz) <-chan IChunk {
	c := make(chan IChunk)
	go func() {
		curChunkXz := ChunkXz{0, 0}
		for z := loc.Z - ChunkRadius; z <= loc.Z+ChunkRadius; z++ {
			for x := loc.X - ChunkRadius; x <= loc.X+ChunkRadius; x++ {
				curChunkXz.X, curChunkXz.Z = x, z
				c <- mgr.Get(&curChunkXz)
			}
		}
		close(c)
	}()
	return c
}
