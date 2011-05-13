// Responsible for reading the overall world persistent state.
// Eventually this should also be responsible for writing it as well.
package worldstore

import (
	"compress/gzip"
	"fmt"
	"os"
	"path"

	"chunkymonkey/chunkstore"
	. "chunkymonkey/types"
	"chunkymonkey/nbt"
)

type WorldStore struct {
	WorldPath string

	LevelData     *nbt.NamedTag
	ChunkStore    chunkstore.ChunkStore
	StartPosition AbsXyz
}

func LoadWorldStore(worldPath string) (world *WorldStore, err os.Error) {
	levelData, err := loadLevelData(worldPath)
	if err != nil {
		return
	}

	chunkStore, err := chunkstore.ChunkStoreForLevel(worldPath, levelData)
	if err != nil {
		return
	}
	startPosition, err := absXyzFromNbt(levelData, "/Data/Player/Pos")
	if err != nil {
		return
	}

	world = &WorldStore{
		WorldPath:     worldPath,
		LevelData:     levelData,
		ChunkStore:    chunkStore,
		StartPosition: startPosition,
	}

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

func absXyzFromNbt(tag nbt.Tag, path string) (pos AbsXyz, err os.Error) {
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
	return fmt.Sprintf("Bad type in level.dat for %s", err)
}
