// Responsible for reading the overall world persistent state.
// Eventually this should also be responsible for writing it as well.
package worldstore

import (
	"compress/gzip"
	"fmt"
	"os"
	"path"

	"chunkymonkey/chunkstore"
	"chunkymonkey/generation"
	. "chunkymonkey/types"
	"nbt"
)

type WorldStore struct {
	WorldPath string

	LevelData     *nbt.NamedTag
	ChunkStore    chunkstore.IChunkStore
	SpawnPosition AbsXyz
}

func LoadWorldStore(worldPath string) (world *WorldStore, err os.Error) {
	levelData, err := loadLevelData(worldPath)
	if err != nil {
		return
	}

	// In both single-player and SMP maps, the 'spawn position' is stored in
	// the level data.
	var spawnPosition AbsXyz

	x, xok := levelData.Lookup("/Data/SpawnX").(*nbt.Int)
	y, yok := levelData.Lookup("/Data/SpawnY").(*nbt.Int)
	z, zok := levelData.Lookup("/Data/SpawnZ").(*nbt.Int)

	if xok && yok && zok {
		spawnPosition = AbsXyz{
			AbsCoord(x.Value),
			AbsCoord(y.Value),
			AbsCoord(z.Value),
		}
	} else {
		err = os.NewError("Invalid map level data: does not contain Spawn{X,Y,Z}")
		return
	}

	var chunkStores []chunkstore.IChunkStore
	persistantChunkStore, err := chunkstore.ChunkStoreForLevel(worldPath, levelData)
	if err != nil {
		return
	}
	chunkStores = append(chunkStores, chunkstore.NewChunkService(persistantChunkStore))

	seed, ok := levelData.Lookup("/Data/RandomSeed").(*nbt.Long)
	if ok {
		chunkStores = append(chunkStores, chunkstore.NewChunkService(generation.NewTestGenerator(seed.Value)))
	}

	for _, store := range chunkStores {
		go store.Serve()
	}

	world = &WorldStore{
		WorldPath:     worldPath,
		LevelData:     levelData,
		ChunkStore:    chunkstore.NewChunkService(chunkstore.NewMultiStore(chunkStores)),
		SpawnPosition: spawnPosition,
	}

	go world.ChunkStore.Serve()

	return
}

func loadLevelData(worldPath string) (levelData *nbt.NamedTag, err os.Error) {
	file, err := os.Open(path.Join(worldPath, "level.dat"))
	if err != nil {
		return
	}
	defer file.Close()

	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		return
	}
	defer gzipReader.Close()

	levelData, err = nbt.Read(gzipReader)

	return
}

func (world *WorldStore) PlayerData(user string) (playerData *nbt.NamedTag, err os.Error) {
	// TODO: This code opens a file, so needs to handle 'bad' usernames
	file, err := os.Open(path.Join(world.WorldPath, "players", user+".dat"))
	if err != nil {
		return
	}
	defer file.Close()

	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		return
	}
	defer gzipReader.Close()

	playerData, err = nbt.Read(gzipReader)

	return
}

func absXyzFromNbt(tag nbt.ITag, path string) (pos AbsXyz, err os.Error) {
	posList, posOk := tag.Lookup(path).(*nbt.List)
	if !posOk {
		err = BadType(path)
		return
	}
	x, xOk := posList.Value[0].(*nbt.Double)
	y, yOk := posList.Value[1].(*nbt.Double)
	z, zOk := posList.Value[2].(*nbt.Double)
	if !xOk || !yOk || !zOk {
		err = BadType(path)
		return
	}

	pos = AbsXyz{
		AbsCoord(x.Value),
		AbsCoord(y.Value),
		AbsCoord(z.Value),
	}
	return
}

type BadType string

func (err BadType) String() string {
	return fmt.Sprintf("Bad type in level.dat for %s", string(err))
}
