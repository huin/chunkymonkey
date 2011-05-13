package entity

import (
	"io"
	"os"
	"chunkymonkey/physics"
	. "chunkymonkey/types"
)

type Entity struct {
	EntityId EntityId
}

type EntityManager struct {
	nextEntityId EntityId
	entities     map[EntityId]*Entity
}

// Allocate and assign a new entity ID
func (mgr *EntityManager) AddEntity(entity *Entity) {
	// EntityManager starts initialized to zero
	if mgr.entities == nil {
		mgr.entities = make(map[EntityId]*Entity)
	}

	// Search for next free ID
	entityId := mgr.nextEntityId
	_, exists := mgr.entities[entityId]
	for exists {
		entityId++
		if entityId == mgr.nextEntityId {
			panic("EntityId space exhausted")
		}
		_, exists = mgr.entities[entityId]
	}

	entity.EntityId = entityId
	mgr.entities[entityId] = entity
	mgr.nextEntityId = entityId + 1
}

func (mgr *EntityManager) RemoveEntity(entity *Entity) {
	mgr.entities[entity.EntityId] = nil, false
}

// ISpawn has the ability to spawn an item or mob. It's used for example by the
// chunks.
// This can't be on interfaces.go because it would create a dependency loop.
type ISpawn interface {
	SendSpawn(io.Writer) os.Error
	GetEntity() *Entity
	GetPosition() *AbsXyz
	SendUpdate(io.Writer) os.Error
	Tick(physics.BlockQueryFn) (leftBlock bool)
}
