// Utility functions for reading/writing values to NBT files.
package nbtutil

import (
	"fmt"
	"os"

	. "chunkymonkey/types"
	"nbt"
)

func ReadFloat2(tag nbt.ITag, path string) (x, y float32, err os.Error) {
	list, ok := tag.Lookup(path).(*nbt.List)
	if !ok || len(list.Value) != 2 {
		err = fmt.Errorf("ReadFloat2 %q: not a list of 2", path)
		return
	}

	xv, xok := list.Value[0].(*nbt.Float)
	yv, yok := list.Value[1].(*nbt.Float)

	if ok = xok && yok; ok {
		x, y = xv.Value, yv.Value
	} else {
		err = fmt.Errorf("ReadFloat2 %q: X or Y was not a Float", path)
	}

	return
}

func ReadDouble3(tag nbt.ITag, path string) (x, y, z float64, err os.Error) {
	list, ok := tag.Lookup(path).(*nbt.List)
	if !ok || len(list.Value) != 3 {
		err = fmt.Errorf("ReadDouble3 %q: not a list of 3", path)
		return
	}

	xv, xok := list.Value[0].(*nbt.Double)
	yv, yok := list.Value[1].(*nbt.Double)
	zv, zok := list.Value[2].(*nbt.Double)

	if ok = xok && yok && zok; ok {
		x, y, z = xv.Value, yv.Value, zv.Value
	} else {
		err = fmt.Errorf("ReadDouble3 %q: X, Y or Z was not a Double", path)
	}

	return
}

func ReadShort(tag nbt.ITag, path string) (v int16, err os.Error) {
	vTag, ok := tag.Lookup(path).(*nbt.Short)
	if !ok {
		err = fmt.Errorf("ReadShort %q: was not a Short", path)
		return
	}

	return vTag.Value, nil
}

func ReadByte(tag nbt.ITag, path string) (v int8, err os.Error) {
	vTag, ok := tag.Lookup(path).(*nbt.Byte)
	if !ok {
		err = fmt.Errorf("ReadByte %q: was not a Byte", path)
		return
	}

	return vTag.Value, nil
}

func ReadInt(tag nbt.ITag, path string) (v int32, err os.Error) {
	vTag, ok := tag.Lookup(path).(*nbt.Int)
	if !ok {
		err = fmt.Errorf("ReadInt %q: was not a Int", path)
		return
	}

	return vTag.Value, nil
}

func ReadFloat(tag nbt.ITag, path string) (v float32, err os.Error) {
	vTag, ok := tag.Lookup(path).(*nbt.Float)
	if !ok {
		err = fmt.Errorf("ReadFloat %q: was not a Float", path)
		return
	}

	return vTag.Value, nil
}

func ReadAbsXyz(tag nbt.ITag, path string) (pos AbsXyz, err os.Error) {
	x, y, z, err := ReadDouble3(tag, path)
	if err != nil {
		return
	}

	return AbsXyz{AbsCoord(x), AbsCoord(y), AbsCoord(z)}, nil
}

func ReadAbsVelocity(tag nbt.ITag, path string) (pos AbsVelocity, err os.Error) {
	// TODO Check if the units of velocity in NBT files are the same that we use
	// internally.
	x, y, z, err := ReadDouble3(tag, path)
	if err != nil {
		return
	}

	return AbsVelocity{AbsVelocityCoord(x), AbsVelocityCoord(y), AbsVelocityCoord(z)}, nil
}

func ReadLookDegrees(tag nbt.ITag, path string) (pos LookDegrees, err os.Error) {
	x, y, err := ReadFloat2(tag, path)
	if err != nil {
		return
	}

	return LookDegrees{AngleDegrees(x), AngleDegrees(y)}, nil
}
