package generation

import (
	"errors"

	"chunkymonkey/chunkstore"
	"chunkymonkey/gamerules"
	. "chunkymonkey/types"
	"math/rand"
	"nbt"
	"perlin"
	"time"
)

const SeaLevel = 63

// ChunkData implements chunkstore.IChunkReader.
type ChunkData struct {
	loc        ChunkXz
	blocks     []byte
	blockData  []byte
	blockLight []byte
	skyLight   []byte
	heightMap  []byte
}

func newChunkData(loc ChunkXz) *ChunkData {
	return &ChunkData{
		loc:        loc,
		blocks:     make([]byte, ChunkSizeH*ChunkSizeH*ChunkSizeY),
		blockData:  make([]byte, (ChunkSizeH*ChunkSizeH*ChunkSizeY)>>1),
		skyLight:   make([]byte, (ChunkSizeH*ChunkSizeH*ChunkSizeY)>>1),
		blockLight: make([]byte, (ChunkSizeH*ChunkSizeH*ChunkSizeY)>>1),
		heightMap:  make([]byte, ChunkSizeH*ChunkSizeH),
	}
}

func (data *ChunkData) ChunkLoc() ChunkXz {
	return data.loc
}

func (data *ChunkData) Blocks() []byte {
	return data.blocks
}

func (data *ChunkData) BlockData() []byte {
	return data.blockData
}

func (data *ChunkData) BlockLight() []byte {
	return data.blockLight
}

func (data *ChunkData) SkyLight() []byte {
	return data.skyLight
}

func (data *ChunkData) HeightMap() []byte {
	return data.heightMap
}

func (data *ChunkData) Entities() []gamerules.INonPlayerEntity {
	return nil
}

func (data *ChunkData) TileEntities() []gamerules.ITileEntity {
	return nil
}

func (data *ChunkData) RootTag() nbt.ITag {
	return nil
}

// TestGenerator implements chunkstore.IChunkStore.
type TestGenerator struct {
	heightSource ISource
	randSource   rand.Source
	randGen      *rand.Rand
}

func NewTestGenerator(seed int64) *TestGenerator {
	perlin := perlin.NewPerlinNoise(seed)

	randSource := rand.NewSource(time.Now().UnixNano())
	randGen := rand.New(randSource)

	return &TestGenerator{
		randSource: randSource,
		randGen:    randGen,
		heightSource: &Sum{
			Inputs: []ISource{
				&Turbulence{
					Dx:     &Scale{50, 1, &Offset{20.1, 0, perlin}},
					Dy:     &Scale{50, 1, &Offset{10.1, 0, perlin}},
					Factor: 50,
					Source: &Scale{
						Wavelength: 200,
						Amplitude:  50,
						Source:     perlin,
					},
				},
				&Turbulence{
					Dx:     &Scale{40, 1, &Offset{20.1, 0, perlin}},
					Dy:     &Scale{40, 1, &Offset{10.1, 0, perlin}},
					Factor: 10,
					Source: &Mult{
						A: &Scale{
							Wavelength: 40,
							Amplitude:  20,
							Source:     perlin,
						},
						// Local steepness.
						B: &Scale{
							Wavelength: 200,
							Amplitude:  1,
							Source:     &Add{perlin, 0.6},
						},
					},
				},
				&Scale{
					Wavelength: 5,
					Amplitude:  2,
					Source:     perlin,
				},
			},
		},
	}
}

func (s *TestGenerator) SupportsWrite() bool {
	return false
}

func (s *TestGenerator) Writer() chunkstore.IChunkWriter {
	return nil
}

func (s *TestGenerator) WriteChunk(writer chunkstore.IChunkWriter) error {
	return errors.New("writes not supported by TestGenerator")
}

func (gen *TestGenerator) ReadChunk(chunkLoc ChunkXz) (reader chunkstore.IChunkReader, err error) {
	baseBlockXyz := chunkLoc.ChunkCornerBlockXY()

	baseX, baseZ := baseBlockXyz.X, baseBlockXyz.Z

	data := newChunkData(chunkLoc)

	baseIndex := BlockIndex(0)
	heightMapIndex := 0
	for x := 0; x < ChunkSizeH; x++ {
		for z := 0; z < ChunkSizeH; z++ {
			xf, zf := float64(x)+float64(baseX), float64(z)+float64(baseZ)
			height := int(SeaLevel + gen.heightSource.At2d(xf, zf))

			if height < 0 {
				height = 0
			} else if height >= ChunkSizeY {
				height = ChunkSizeY - 1
			}

			skyLightHeight := gen.setBlockStack(
				height,
				data.blocks[baseIndex:baseIndex+ChunkSizeY])

			data.heightMap[heightMapIndex] = byte(skyLightHeight)

			heightMapIndex++
			baseIndex += ChunkSizeY
		}
	}

	// The chunk has been generated, now add some trees if appropriate
	gen.addSaplings(data)
	gen.setSkylight(data)

	return data, nil
}

func (gen *TestGenerator) setBlockStack(height int, blocks []byte) (skyLightHeight int) {
	var topBlockType byte
	if height < SeaLevel+1 {
		skyLightHeight = SeaLevel + 1

		for y := SeaLevel; y > height; y-- {
			blocks[y] = 9 // stationary water
		}
		blocks[height] = 12 // sand
		topBlockType = 12
	} else {
		skyLightHeight = height + 1
		if height <= SeaLevel+1 {
			blocks[height] = 12 // sand
			topBlockType = 12
		} else {
			blocks[height] = 2 // grass
			topBlockType = 3   // dirt
		}
	}

	for y := height - 1; y > height-3 && y > 0; y-- {
		blocks[y] = topBlockType
	}
	for y := height - 3; y > 0; y-- {
		blocks[y] = 1 // stone
	}

	if skyLightHeight < 0 {
		skyLightHeight = 0
	}

	return
}

func (gen *TestGenerator) setSkyLightStack(skyLightHeight int, blocks []byte, skyLight []byte) {
	for y := ChunkSizeY - 1; y >= skyLightHeight; y-- {
		BlockIndex(y).SetBlockData(skyLight, 15)
	}

	var lightLevel int8 = 15

	for y := skyLightHeight; y >= 0 && lightLevel > 0; y-- {
		blockType, ok := gamerules.Blocks.Get(BlockId(blocks[y]))
		if lightLevel > 0 && ok && blockType.Opacity > 0 {
			lightLevel -= blockType.Opacity
		}

		BlockIndex(y).SetBlockData(skyLight, byte(lightLevel))
	}
}

func (gen *TestGenerator) setSkylight(data *ChunkData) {
	baseIndex := 0
	heightMapIndex := 0

	for x := 0; x < ChunkSizeH; x++ {
		for z := 0; z < ChunkSizeH; z++ {
			lightBase := baseIndex >> 1

			gen.setSkyLightStack(
				int(data.heightMap[heightMapIndex]),
				data.blocks[baseIndex:baseIndex+ChunkSizeY],
				data.skyLight[lightBase:lightBase+ChunkSizeY/2])
			heightMapIndex++
			baseIndex += ChunkSizeY
		}
	}

}

func (gen *TestGenerator) addSaplings(data *ChunkData) {
	baseIndex := 0
	heightMapIndex := 0

	// Move throughout the chunk, but refuse to create trees on chunk
	// boundaries.
	for x := 0; x < ChunkSizeH; x++ {
		for z := 0; z < ChunkSizeH; z++ {
			topBlock := int(data.heightMap[heightMapIndex] - 1)
			blockIndex := baseIndex + int(topBlock)

			if data.blocks[blockIndex] == 2 {
				// We could add a tree, check to see if we want to
				addTree := gen.randGen.Intn(100) > 95
				if addTree && x > 0 && x < ChunkSizeH-1 && z > 0 && z < ChunkSizeH-1 {
					if !adjacentBlockIs(data, x, topBlock, z, 2, 2, 2, 6) {
						// Check if an adjacent block has a sapling already
						data.blocks[blockIndex+1] = 6
					}
				}
			}

			heightMapIndex++
			baseIndex += ChunkSizeY
		}
	}
}

// adjacentBlockIs return whether or not at least one block adjacent to
// (bx,by,bz) is of type 'blockType'. The area which this function checks is
// specified by dx,dy,dz. Blocks outside the given chunk are not checked.
func adjacentBlockIs(data *ChunkData, bx, by, bz, dx, dy, dz int, blockType byte) bool {
	subChunk := new(SubChunkXyz)

	for x := bx - dx; x <= bx+dx; x++ {
		for z := bz - dz; z <= bz+dz; z++ {
			for y := by - dy; y <= by+dy; y++ {
				if x != bx || y != by || z != bz {
					subChunk.X = SubChunkCoord(x)
					subChunk.Y = SubChunkCoord(y)
					subChunk.Z = SubChunkCoord(z)
					index, ok := subChunk.BlockIndex()
					if ok {
						if data.blocks[index] == blockType {
							return true
						}
					}
				}
			}
		}
	}

	return false
}
