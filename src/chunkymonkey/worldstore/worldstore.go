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

func LoadWorldStore(worldPath string) (world *WorldStore, err error) {
	levelData, err := loadLevelData(worldPath)
	if err != nil {
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
		return nil, err
	}

	persistantChunkService := chunkstore.NewChunkService(persistantChunkStore)
	chunkStores = append(chunkStores, persistantChunkService)

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
		ChunkStore:    chunkstore.NewChunkService(chunkstore.NewMultiStore(chunkStores, persistantChunkService)),
		SpawnPosition: spawnPosition,
	}

	go world.ChunkStore.Serve()

	return
}

func loadLevelData(worldPath string) (levelData nbt.ITag, err error) {
	filename := path.Join(worldPath, "level.dat")
	file, err := os.Open(filename)
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

// NOTE: ChunkStoreForDimension shouldn't really be used in the server just
// yet.
func (world *WorldStore) ChunkStoreForDimension(dimension DimensionId) (store chunkstore.IChunkStore, err error) {
	fgStore, err := chunkstore.ChunkStoreForLevel(world.WorldPath, world.LevelData, dimension)
	if err != nil {
		return
	}
	store = chunkstore.NewChunkService(fgStore)
	go store.Serve()
	return
}

func (world *WorldStore) PlayerData(user string) (playerData *nbt.Compound, err error) {
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

func (world *WorldStore) WritePlayerData(user string, data *nbt.Compound) (err error) {
	playerDir := path.Join(world.WorldPath, "players")
	if err = os.MkdirAll(playerDir, 0777); err != nil {
		return
	}

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

// Creates a new world at 'worldPath'
func CreateWorld(worldPath string) (err error) {
	source := rand.NewSource(time.Nanoseconds())
	seed := source.Int63()

	data := &nbt.Compound{
		map[string]nbt.ITag{
			"Data": &nbt.Compound{
				map[string]nbt.ITag{
					"Time":        &nbt.Long{0},
					"rainTime":    &nbt.Int{0},
					"thunderTime": &nbt.Int{0},
					"version":     &nbt.Int{19132}, // TODO: What should this be?
					"thundering":  &nbt.Byte{0},
					"raining":     &nbt.Byte{0},
					"LevelName":   &nbt.String{"world"}, // TODO: Should be specifyable
					"SpawnX":      &nbt.Int{0},          // TODO: Figure this out from chunk generator?
					"SpawnY":      &nbt.Int{75},         // TODO: Figure this out from chunk generator?
					"SpawnZ":      &nbt.Int{0},          // TODO: Figure this out from chunk generator?
					"LastPlayed":  &nbt.Long{0},
					"SizeOnDisk":  &nbt.Long{0}, // Needs to be accurate?
					"RandomSeed":  &nbt.Long{seed},
				},
			},
		},
	}

	if err = os.MkdirAll(worldPath, 0777); err != nil {
		return
	}

	filename := path.Join(worldPath, "level.dat")
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		return err
	}

	gzipWriter, err := gzip.NewWriter(file)
	if err != nil {
		return err
	}

	err = nbt.Write(gzipWriter, data)
	gzipWriter.Close()
	file.Close()

	return nil
}

func absXyzFromNbt(tag nbt.ITag, path string) (pos AbsXyz, err error) {
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
