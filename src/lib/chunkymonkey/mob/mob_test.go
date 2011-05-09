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
				m := NewPig()
				m.Mob.EntityId = 8888
				m.SetBurning(true)
				m.SetBurning(false)
				m.SetPosition(types.AbsXyz{11, 70, -172})
				m.SetLook(types.LookDegrees{10, 20})
				buf := bytes.NewBuffer(nil)
				if err := m.SendSpawn(buf); err != nil {
					return ""
				}
				return buf.String()
			},
			"\x18\x00\x00\"\xb8Z\x00\x00\x01`\x00\x00\b\xc0\xff\xff\xea\x80\a\x0e\x00\x00\x10\x00\x7f",
		},
		{
			"creeper",
			func() string {
				m := NewCreeper()
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
			"\x18\x00\x00\x1ea2\x00\x00\x01`\x00\x00\b\xc0\xff\xff\xea\x80\x00\x8d\x00\x01\x11\x01\x10\xff\x7f",
		},
	}
	for _, x := range tests {
		want, result := x.want, x.result()
		if want != result {
			t.Errorf("Resulting raw data mismatch for %v spawn, wanted:\n\t%q\n\tbut got: \n\t%q", x.name, want, result)
		}
	}
}
