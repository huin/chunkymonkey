package generation

import (
	"os"

	"chunkymonkey/chunkstore"
	. "chunkymonkey/types"
	"nbt"
	"perlin"
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

func (data *ChunkData) Entities() []nbt.ITag {
	return nil
}

func (data *ChunkData) RootTag() nbt.ITag {
	return nil
}

// TestGenerator implements chunkstore.IChunkStore.
type TestGenerator struct {
	heightSource ISource
}

func NewTestGenerator(seed int64) *TestGenerator {
	perlin := perlin.NewPerlinNoise(seed)

	return &TestGenerator{
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

func (gen *TestGenerator) LoadChunk(chunkLoc ChunkXz) (reader chunkstore.IChunkReader, err os.Error) {
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

			lightBase := baseIndex >> 1

			gen.setSkyLightStack(
				skyLightHeight,
				data.blocks[baseIndex:baseIndex+ChunkSizeY],
				data.skyLight[lightBase:lightBase+ChunkSizeY/2])

			data.heightMap[heightMapIndex] = byte(skyLightHeight)

			heightMapIndex++
			baseIndex += ChunkSizeY
		}
	}

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

	lightLevel := 15

	for y := skyLightHeight; y >= 0 && lightLevel > 0; y-- {
		// TODO Use real block data in here.
		if blocks[y] == 9 {
			lightLevel -= 3
		} else if blocks[y] == 0 {
			// air
		} else {
			lightLevel -= 15
		}
		if lightLevel < 0 {
			lightLevel = 0
		}

		BlockIndex(y).SetBlockData(skyLight, byte(lightLevel))
	}
}
