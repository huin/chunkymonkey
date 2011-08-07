// Responsible for reading the overall world persistent state.
// Eventually this should also be responsible for writing it as well.
package worldstore

import (
	"compress/gzip"
	"fmt"
	"log"
	"os"
	"path"
	"rand"
	"time"

	"chunkymonkey/chunkstore"
	"chunkymonkey/generation"
	. "chunkymonkey/types"
	"chunkymonkey/util"
	"nbt"
)

type WorldStore struct {
	WorldPath string

	Seed int64
	Time Ticks

	LevelData     nbt.ITag
	ChunkStore    chunkstore.IChunkStore
	SpawnPosition BlockXyz
}

func LoadWorldStore(worldPath string) (world *WorldStore, err os.Error) {
	levelData, err := loadLevelData(worldPath)
	if err != nil {
		return
	}

	if err = makeSubdirs(worldPath); err != nil {
		return
	}

	// In both single-player and SMP maps, the 'spawn position' is stored in
	// the level data.
	x, xok := levelData.Lookup("Data/SpawnX").(*nbt.Int)
	y, yok := levelData.Lookup("Data/SpawnY").(*nbt.Int)
	z, zok := levelData.Lookup("Data/SpawnZ").(*nbt.Int)
	if !xok || !yok || !zok {
		err = os.NewError("Invalid map level data: does not contain Spawn{X,Y,Z}")
		log.Printf("%#v", levelData)
		return
	}
	spawnPosition := BlockXyz{
		BlockCoord(x.Value),
		BlockYCoord(y.Value),
		BlockCoord(z.Value),
	}

	var timeTicks Ticks
	if timeTag, ok := levelData.Lookup("Data/Time").(*nbt.Long); ok {
		timeTicks = Ticks(timeTag.Value)
	}

	var chunkStores []chunkstore.IChunkStore
	persistantChunkStore, err := chunkstore.ChunkStoreForLevel(worldPath, levelData, DimensionNormal)
	if err != nil {
		return
	}
	chunkStores = append(chunkStores, chunkstore.NewChunkService(persistantChunkStore))

	var seed int64
	if seedNbt, ok := levelData.Lookup("Data/RandomSeed").(*nbt.Long); ok {
		seed = seedNbt.Value
	} else {
		seed = rand.NewSource(time.Seconds()).Int63()
	}

	chunkStores = append(chunkStores, chunkstore.NewChunkService(generation.NewTestGenerator(seed)))

	for _, store := range chunkStores {
		go store.Serve()
	}

	world = &WorldStore{
		WorldPath:     worldPath,
		Seed:          seed,
		Time:          timeTicks,
		LevelData:     levelData,
		ChunkStore:    chunkstore.NewChunkService(chunkstore.NewMultiStore(chunkStores)),
		SpawnPosition: spawnPosition,
	}

	go world.ChunkStore.Serve()

	return
}

func loadLevelData(worldPath string) (levelData nbt.ITag, err os.Error) {
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

func makeSubdirs(worldPath string) (err os.Error) {
	// Worlds created by the minecraft client don't have the players directory.
	directory := path.Join(worldPath, "players")
	stat, err := os.Stat(directory)
	if err == nil && stat.IsDirectory() {
		return nil
	}
	if err = os.MkdirAll(directory, 0755); err != nil {
		err = os.NewError("Could not create worldstore directory: " + err.String())
		return err
	}
	return
}


// NOTE: ChunkStoreForDimension shouldn't really be used in the server just
// yet.
func (world *WorldStore) ChunkStoreForDimension(dimension DimensionId) (store chunkstore.IChunkStore, err os.Error) {
	fgStore, err := chunkstore.ChunkStoreForLevel(world.WorldPath, world.LevelData, dimension)
	if err != nil {
		return
	}
	store = chunkstore.NewChunkService(fgStore)
	go store.Serve()
	return
}

func (world *WorldStore) PlayerData(user string) (playerData nbt.ITag, err os.Error) {
	file, err := os.Open(path.Join(world.WorldPath, "players", user+".dat"))
	if err != nil {
		if errno, ok := util.Errno(err); ok && errno == os.ENOENT {
			// Player data simply doesn't exist. Not an error, playerData = nil is
			// the result.
			return nil, nil
		}
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

func (world *WorldStore) WritePlayerData(user string, data *nbt.Compound) (err os.Error) {
	filename := path.Join(world.WorldPath, "players", user+".dat")
	file, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE, 0666)

	if err != nil {
		return err
	}
	defer file.Close()

	gzipWriter, err := gzip.NewWriter(file)
	if err != nil {
		return
	}

	err = nbt.Write(gzipWriter, data)
	gzipWriter.Close()

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
