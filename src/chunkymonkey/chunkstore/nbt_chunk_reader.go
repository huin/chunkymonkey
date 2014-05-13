package chunkstore

import (
	"io"
	"log"
	"os"

	"chunkymonkey/gamerules"
	. "chunkymonkey/types"
	"nbt"
)

// Returned to chunks to pull their data from.
type nbtChunkReader struct {
	chunkTag nbt.ITag
}

// Load a chunk from its NBT representation
func newNbtChunkReader(reader io.Reader) (r *nbtChunkReader, err error) {
	chunkTag, err := nbt.Read(reader)
	if err != nil {
		return
	}

	r = &nbtChunkReader{
		chunkTag: chunkTag,
	}

	return
}

func (r *nbtChunkReader) ChunkLoc() ChunkXz {
	return ChunkXz{
		X: ChunkCoord(r.chunkTag.Lookup("Level/xPos").(*nbt.Int).Value),
		Z: ChunkCoord(r.chunkTag.Lookup("Level/zPos").(*nbt.Int).Value),
	}
}

func (r *nbtChunkReader) Blocks() []byte {
	return r.chunkTag.Lookup("Level/Blocks").(*nbt.ByteArray).Value
}

func (r *nbtChunkReader) BlockData() []byte {
	return r.chunkTag.Lookup("Level/Data").(*nbt.ByteArray).Value
}

func (r *nbtChunkReader) BlockLight() []byte {
	return r.chunkTag.Lookup("Level/BlockLight").(*nbt.ByteArray).Value
}

func (r *nbtChunkReader) SkyLight() []byte {
	return r.chunkTag.Lookup("Level/SkyLight").(*nbt.ByteArray).Value
}

func (r *nbtChunkReader) HeightMap() []byte {
	return r.chunkTag.Lookup("Level/HeightMap").(*nbt.ByteArray).Value
}

func (r *nbtChunkReader) Entities() (entities []gamerules.INonPlayerEntity) {
	entityListTag, ok := r.chunkTag.Lookup("Level/Entities").(*nbt.List)
	if !ok {
		return
	}

	entities = make([]gamerules.INonPlayerEntity, 0, len(entityListTag.Value))
	for _, tag := range entityListTag.Value {
		compound, ok := tag.(*nbt.Compound)
		if !ok {
			log.Printf("Found non-compound in entities list: %T", tag)
			continue
		}

		entityObjectId, ok := compound.Lookup("id").(*nbt.String)
		if !ok {
			log.Printf("Missing or bad entity type ID in NBT: %s", entityObjectId)
			continue
		}

		entity := gamerules.NewEntityByTypeName(entityObjectId.Value)
		if entity == nil {
			log.Printf("Found unhandled entity type: %s", entityObjectId.Value)
			continue
		}

		err := entity.UnmarshalNbt(compound)
		if err != nil {
			log.Printf("Error unmarshalling entity NBT: %s", err)
			continue
		}

		entities = append(entities, entity)
	}

	return
}

func (r *nbtChunkReader) TileEntities() (tileEntities []gamerules.ITileEntity) {
	entityListTag, ok := r.chunkTag.Lookup("Level/TileEntities").(*nbt.List)
	if !ok {
		return
	}

	tileEntities = make([]gamerules.ITileEntity, 0, len(entityListTag.Value))
	for _, tag := range entityListTag.Value {
		compound, ok := tag.(*nbt.Compound)
		if !ok {
			log.Printf("Found non-compound in tile entities list: %T", tag)
			continue
		}

		entityObjectId, ok := compound.Lookup("id").(*nbt.String)
		if !ok {
			log.Printf("Missing or bad tile entity type ID in NBT: %s", entityObjectId)
			continue
		}

		entity := gamerules.NewTileEntityByTypeName(entityObjectId.Value)
		if entity == nil {
			log.Printf("Found unhandled tile entity type: %s", entityObjectId.Value)
			continue
		}

		if err := entity.UnmarshalNbt(compound); err != nil {
			log.Printf("%T.UnmarshalNbt failed: %s", err)
			continue
		}

		tileEntities = append(tileEntities, entity)
	}

	return
}

func (r *nbtChunkReader) RootTag() nbt.ITag {
	return r.chunkTag
}
