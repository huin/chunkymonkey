package gamerules

import (
	"errors"
	"nbt"
)

func makeRecordPlayerAspect() (aspect IBlockAspect) {
	return &RecordPlayerAspect{}
}

type recordPlayerTileEntity struct {
	tileEntity
	record int32
}

func NewRecordPlayerTileEntity() ITileEntity {
	return &recordPlayerTileEntity{}
}

func (recordPlayer *recordPlayerTileEntity) UnmarshalNbt(tag *nbt.Compound) (err error) {
	if err = recordPlayer.tileEntity.UnmarshalNbt(tag); err != nil {
		return
	}

	if recordTag, ok := tag.Lookup("Record").(*nbt.Int); !ok {
		return errors.New("missing or incorrect type for RecordPlayer Record")
	} else {
		recordPlayer.record = recordTag.Value
	}

	return nil
}

func (recordPlayer *recordPlayerTileEntity) MarshalNbt(tag *nbt.Compound) (err error) {
	if err = recordPlayer.tileEntity.MarshalNbt(tag); err != nil {
		return
	}

	tag.Set("id", &nbt.String{"RecordPlayer"})
	tag.Set("Record", &nbt.Int{recordPlayer.record})

	return nil
}

type RecordPlayerAspect struct {
	StandardAspect
}

func (aspect RecordPlayerAspect) Name() string {
	return "RecordPlayer"
}

// TODO behaviours for record player.
