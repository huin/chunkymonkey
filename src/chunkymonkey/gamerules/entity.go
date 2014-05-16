// Defines interfaces for entities in the world (including pick up items,
// mobs, players, and other non-block objects).

package gamerules

import (
	"io"

	"chunkymonkey/physics"
	. "chunkymonkey/types"
	"nbt"
)

// INbtSerializable is the interface for all objects that can be serialized to
// NBT data structures for persistency.
type INbtSerializable interface {
	// UnmarshalNbt reads the NBT tag to set the state of the object.
	UnmarshalNbt(*nbt.Compound) error

	// MarshalNbt creates an NBT tag representing the entity. This can be nil if
	// the entity cannot be serialized.
	MarshalNbt(*nbt.Compound) error
}

// IEntity represents common elements to all types of non-block entities that
// can be present in a chunk.
type IEntity interface {
	// Returns the entity's ID.
	GetEntityId() EntityId

	// SendSpawn writes the packets required to tell a client about the existance
	// and current state of the entity.
	SendSpawn(io.Writer) error

	// SendUpdate writes the packets required to tell a client about the new
	// state of the entity since the last SendUpdate or SendSpawn.
	SendUpdate(io.Writer) error

	// Returns the entity's current position.
	Position() *AbsXyz
}

// INonPlayerEntity is the interface for entities other than players which are
// controlled server-side.
type INonPlayerEntity interface {
	IEntity
	INbtSerializable

	// Sets the entity's ID.
	SetEntityId(EntityId)

	// Runs the physics for the entity for a single server tick.
	Tick(physics.IBlockQuerier) (leftBlock bool)
}

// ITileEntity is the interface common to entities that are tile-based.
type ITileEntity interface {
	INbtSerializable

	// SetChunk sets the parent chunk of the tile entity. This must be called
	// after the tile entity is deserialized and before any game event methods
	// are called.
	SetChunk(chunk IChunkBlock)

	// Block returns the position of the tile entity.
	Block() BlockXyz
}
