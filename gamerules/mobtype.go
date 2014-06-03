package gamerules

import (
	. "github.com/huin/chunkymonkey/types"
)

type MobType struct {
	Id   EntityMobType
	Name string
}

type MobTypeMap map[EntityMobType]*MobType

// Used for protocol parsing.
var Mobs = MobTypeMap{
	MobTypeIdCreeper:      &CreeperType,
	MobTypeIdSkeleton:     &SkeletonType,
	MobTypeIdSpider:       &SpiderType,
	MobTypeIdGiantZombie:  &GiantZombieType,
	MobTypeIdZombie:       &ZombieType,
	MobTypeIdSlime:        &SlimeType,
	MobTypeIdGhast:        &GhastType,
	MobTypeIdZombiePigman: &ZombiePigmanType,
	MobTypeIdPig:          &PigType,
	MobTypeIdSheep:        &SheepType,
	MobTypeIdCow:          &CowType,
	MobTypeIdHen:          &HenType,
	MobTypeIdSquid:        &SquidType,
	MobTypeIdWolf:         &WolfType,
}

var CreeperType = MobType{MobTypeIdCreeper, "creeper"}
var SkeletonType = MobType{MobTypeIdSkeleton, "skeleton"}
var SpiderType = MobType{MobTypeIdSpider, "spider"}
var GiantZombieType = MobType{MobTypeIdGiantZombie, "giantzombie"}
var ZombieType = MobType{MobTypeIdZombie, "zombie"}
var SlimeType = MobType{MobTypeIdSlime, "slime"}
var GhastType = MobType{MobTypeIdGhast, "ghast"}
var ZombiePigmanType = MobType{MobTypeIdZombiePigman, "zombiepigman"}
var PigType = MobType{MobTypeIdPig, "pig"}
var SheepType = MobType{MobTypeIdSheep, "sheep"}
var CowType = MobType{MobTypeIdCow, "cow"}
var HenType = MobType{MobTypeIdHen, "hen"}
var SquidType = MobType{MobTypeIdSquid, "squid"}
var WolfType = MobType{MobTypeIdWolf, "wolf"}
