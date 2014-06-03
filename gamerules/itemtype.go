package gamerules

import (
	. "github.com/huin/chunkymonkey/types"
)

const (
	MaxStackDefault = ItemCount(64)
)

type ToolTypeId byte

type ItemType struct {
	Id       ItemTypeId
	Name     string
	MaxStack ItemCount
	ToolType ToolTypeId
	ToolUses ItemData
}

type ItemTypeMap map[ItemTypeId]*ItemType
