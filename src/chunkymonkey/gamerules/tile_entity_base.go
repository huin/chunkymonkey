package gamerules

import (
	"chunkymonkey/nbtutil"
	. "chunkymonkey/types"
	"nbt"
)

type tileEntity struct {
	chunk    IChunkBlock
	blockLoc BlockXyz
}

func (tileEntity *tileEntity) UnmarshalNbt(tag *nbt.Compound) (err error) {
	if tileEntity.blockLoc, err = nbtutil.ReadBlockXyzCompound(tag); err != nil {
		return
	}

	return nil
}

func (tileEntity *tileEntity) MarshalNbt(tag *nbt.Compound) (err error) {
	nbtutil.WriteBlockXyzCompound(tag, tileEntity.blockLoc)
	return nil
}

func (tileEntity *tileEntity) SetChunk(chunk IChunkBlock) {
	tileEntity.chunk = chunk
}

func (tileEntity *tileEntity) Block() BlockXyz {
	return tileEntity.blockLoc
}
