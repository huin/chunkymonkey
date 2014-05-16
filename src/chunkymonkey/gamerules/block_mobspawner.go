package gamerules

import (
	. "chunkymonkey/types"
	"errors"
	"nbt"
)

func makeMobSpawnerAspect() (aspect IBlockAspect) {
	return &MobSpawnerAspect{}
}

type mobSpawnerTileEntity struct {
	tileEntity
	entityMobType string
	delay         Ticks
}

func NewMobSpawnerTileEntity() ITileEntity {
	return &mobSpawnerTileEntity{}
}

func (mobSpawner *mobSpawnerTileEntity) UnmarshalNbt(tag *nbt.Compound) (err error) {
	if err = mobSpawner.tileEntity.UnmarshalNbt(tag); err != nil {
		return
	}

	if entityIdTag, ok := tag.Lookup("EntityId").(*nbt.String); !ok {
		return errors.New("missing or incorrect type for MobSpawner EntityId")
	} else {
		mobSpawner.entityMobType = entityIdTag.Value
	}

	if delayTag, ok := tag.Lookup("Delay").(*nbt.Short); !ok {
		return errors.New("missing or incorrect type for MobSpawner Delay")
	} else {
		mobSpawner.delay = Ticks(delayTag.Value)
	}

	return nil
}

func (mobSpawner *mobSpawnerTileEntity) MarshalNbt(tag *nbt.Compound) (err error) {
	if err = mobSpawner.tileEntity.MarshalNbt(tag); err != nil {
		return
	}

	tag.Set("id", &nbt.String{"MobSpawner"})
	tag.Set("EntityId", &nbt.String{mobSpawner.entityMobType})
	tag.Set("Delay", &nbt.Short{int16(mobSpawner.delay)})

	return nil
}

type MobSpawnerAspect struct {
	StandardAspect
}

func (aspect MobSpawnerAspect) Name() string {
	return "MobSpawner"
}

// TODO behaviours for mob spawners.
