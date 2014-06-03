package gamerules

var EntityCreateByName = map[string]func() INonPlayerEntity{
	// Pick-up items.
	"Item": NewBlankItem,

	// Mobs.
	"Hen":      NewHen,
	"Chicken":  NewHen,
	"Cow":      NewCow,
	"Creeper":  NewCreeper,
	"Pig":      NewPig,
	"Sheep":    NewSheep,
	"Skeleton": NewSkeleton,
	"Squid":    NewSquid,
	"Spider":   NewSpider,
	"Wolf":     NewWolf,
	"Zombie":   NewZombie,

	// "Objects".
	"Boat":           NewBoat,
	"Minecart":       NewMinecart,
	"StorageCart":    NewStorageCart,
	"PoweredCart":    NewPoweredCart,
	"ActivatedTnt":   NewActivatedTnt,
	"Arrow":          NewArrow,
	"ThrownSnowball": NewThrownSnowball,
	"ThrownEgg":      NewThrownEgg,
	"FallingSand":    NewFallingSand,
	"FallingGravel":  NewFallingGravel,
	"FishingFloat":   NewFishingFloat,
}

// NewEntityByTypeName creates the appropriate entity type based on the input
// string, e.g "Item" or "Zombie". Returns nil if typeName is unknown.
func NewEntityByTypeName(typeName string) INonPlayerEntity {
	if fn, ok := EntityCreateByName[typeName]; ok {
		return fn()
	}

	return nil
}

var TileEntityCreateByName = map[string]func() ITileEntity{
	"Chest":        NewChestTileEntity,
	"Furnace":      NewFurnaceTileEntity,
	"Trap":         NewDispenserTileEntity,
	"Sign":         NewSignTileEntity,
	"MobSpawner":   NewMobSpawnerTileEntity,
	"Music":        NewMusicTileEntity,
	"RecordPlayer": NewRecordPlayerTileEntity,
}

func NewTileEntityByTypeName(typeName string) ITileEntity {
	if fn, ok := TileEntityCreateByName[typeName]; ok {
		return fn()
	}

	return nil
}
