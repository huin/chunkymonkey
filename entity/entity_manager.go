package entity

import (
	"sync"

	. "chunkymonkey/types"
)

type EntityManager struct {
	nextEntityId EntityId
	entities     map[EntityId]bool
	lock         sync.Mutex
}

func (mgr *EntityManager) Init() {
	mgr.lock.Lock()
	defer mgr.lock.Unlock()

	mgr.nextEntityId = 0
	mgr.entities = make(map[EntityId]bool)
}

func (mgr *EntityManager) createEntityId() EntityId {
	// Search for next free ID
	entityId := mgr.nextEntityId
	_, exists := mgr.entities[entityId]
	for exists {
		entityId++
		if entityId == mgr.nextEntityId {
			// TODO Better handling of this? It shouldn't happen, realistically - but
			// neither should it explode.
			panic("EntityId space exhausted")
		}
		_, exists = mgr.entities[entityId]
	}
	mgr.nextEntityId = entityId + 1

	return entityId
}

// AddEntity adds an entity to the manager, and assigns it a world-unique
// EntityId.
// NewEntity creates a world-unique entityId in the manager and returns it.
func (mgr *EntityManager) NewEntity() EntityId {
	mgr.lock.Lock()
	defer mgr.lock.Unlock()

	entityId := mgr.createEntityId()
	mgr.entities[entityId] = true
	return entityId
}

// RemoveEntity removes an entity from the manager.
func (mgr *EntityManager) RemoveEntityById(entityId EntityId) {
	mgr.lock.Lock()
	defer mgr.lock.Unlock()

	delete(mgr.entities, entityId)
}
