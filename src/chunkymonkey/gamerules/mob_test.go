package gamerules

import (
	"bytes"
	"os"
	"testing"

	"chunkymonkey/types"
	te "testencoding"
)

type testCase struct {
	name   string
	result func(writer *bytes.Buffer) (err error)
	want   te.IBytesMatcher
}

func TestMobSpawn(t *testing.T) {
	tests := []testCase{
		{
			"pig",
			func(writer *bytes.Buffer) error {
				m := NewPig().(*Pig)
				m.PointObject.Init(&types.AbsXyz{11, 70, -172}, &types.AbsVelocity{0, 0, 0})
				m.Mob.EntityId = 0x1234
				m.SetBurning(true)
				m.SetBurning(false)
				m.SetLook(types.LookDegrees{10, 20})
				return m.SendSpawn(writer)
			},
			te.InOrder(
				// packetIdEntitySpawn
				te.LiteralString("\x18"+ // Packet ID
					"\x00\x00\x12\x34"+ // EntityId
					"Z"+ // EntityMobType
					"\x00\x00\x01`\x00\x00\b\xc0\xff\xff\xea\x80"+ // X, Y, Z
					"\a\x0e", // Yaw, Pitch
				),
				te.AnyOrder(
					te.LiteralString("\x00\x00"), // burning=false
					te.LiteralString("\x10\x00"), // 16=0 (?)
				),
				te.LiteralString("\x7f"), // 127 = end of metadata
				// packetIdEntityVelocity
				te.LiteralString("\x1c\x00\x00\x12\x34\x00\x00\x00\x00\x00\x00"),
			),
		},
		{
			"creeper",
			func(writer *bytes.Buffer) error {
				// Bogus position, changing below.
				m := NewCreeper().(*Creeper)
				m.PointObject.Init(&types.AbsXyz{11, 70, -172}, &types.AbsVelocity{})
				m.Mob.EntityId = 0x5678
				m.CreeperSetBlueAura()
				m.SetBurning(true)
				m.SetLook(types.LookDegrees{0, 199})
				return m.SendSpawn(writer)
			},
			te.InOrder(
				// packetIdEntitySpawn
				te.LiteralString("\x18"+ // Packet ID
					"\x00\x00\x56\x78"+ // EntityId
					"2"+ // EntityMobType
					"\x00\x00\x01\x60\x00\x00\x08\xc0\xff\xff\xea\x80"+ // X, Y, Z
					"\x00\x8d", // Yaw, Pitch
				),
				te.AnyOrder(
					te.LiteralString("\x00\x01"), // burning=true
					te.LiteralString("\x10\xff"), // 16=255 (?)
					te.LiteralString("\x11\x01"), // blue aura=true
				),
				te.LiteralString("\x7f"), // 127 = end of metadata
				// packetIdEntityVelocity
				te.LiteralString("\x1c\x00\x00\x56\x78\x00\x00\x00\x00\x00\x00"),
			),
		},
	}
	for _, x := range tests {
		buf := new(bytes.Buffer)
		want, err := x.want, x.result(buf)
		if err != nil {
			t.Errorf("Error when writing in test %s: %v", x.name, err)
			continue
		}
		result := buf.Bytes()
		if err = te.Matches(want, result); err != nil {
			t.Errorf("Resulting raw data mismatch for %s spawn: %v\nGot bytes: %x", x.name, err, result)
		}
	}
}
