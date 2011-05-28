package mob

import (
	"bytes"
	"testing"

	"chunkymonkey/types"
)

type testCase struct {
	name   string
	result func() string
	want   string
}


func TestMobSpawn(t *testing.T) {
	tests := []testCase{
		{
			"pig",
			func() string {
				m := NewPig(&types.AbsXyz{11, 70, -172}, &types.AbsVelocity{0, 0, 0})
				m.Mob.EntityId = 8888
				m.SetBurning(true)
				m.SetBurning(false)
				m.SetLook(types.LookDegrees{10, 20})
				buf := bytes.NewBuffer(nil)
				if err := m.SendSpawn(buf); err != nil {
					return ""
				}
				return buf.String()
			},
			"\x18\x00\x00\"\xb8Z\x00\x00\x01`\x00\x00\b\xc0\xff\xff\xea\x80\a\x0e\x00\x00\x10\x00\x7f\x1c\x00\x00\"\xb8\x00\x00\x00\x00\x00\x00",
		},
		{
			"creeper",
			func() string {
				// Bogus position, changing below.
				m := NewCreeper(&types.AbsXyz{0, 0, 0}, &types.AbsVelocity{})
				m.Mob.EntityId = 7777
				m.CreeperSetBlueAura()
				m.SetBurning(true)
				m.SetPosition(types.AbsXyz{11, 70, -172})
				m.SetLook(types.LookDegrees{0, 199})
				buf := bytes.NewBuffer(nil)
				if err := m.SendSpawn(buf); err != nil {
					return ""
				}
				return buf.String()
			},
			"\x18\x00\x00\x1ea2\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x8d\x00\x01\x10\xff\x11\x01\x7f\x1c\x00\x00\x1ea\x00\x00\x00\x00\x00\x00",
		},
	}
	for _, x := range tests {
		want, result := x.want, x.result()
		if want != result {
			t.Errorf("Resulting raw data mismatch for %v spawn, wanted:\n\t%x\n\tbut got: \n\t%x", x.name, want, result)
		}
	}
}
