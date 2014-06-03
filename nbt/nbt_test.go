package nbt

import (
	"bytes"
	"fmt"
	"reflect"
	"testing"

	te "github.com/huin/chunkymonkey/testencoding"
)

type Test struct {
	Serialized te.IBytesMatcher
	Value      ITag
}

func (test *Test) String() string {
	return fmt.Sprintf("Test{Serialized=%v, Value=%#v}", test.Serialized, test.Value)
}

func (test *Test) testRead(t *testing.T) {
	t.Logf("Test read %v", test)

	bytesBuf := new(bytes.Buffer)
	test.Serialized.Write(bytesBuf)

	var result ITag
	var err error
	if result, err = test.Value.Type().NewTag(); err != nil {
		t.Errorf("  Fail: failed to create tag to read into: %v", err)
		return
	}

	if err = result.Read(bytesBuf); err != nil {
		t.Errorf("  Fail: failed to read a %T: %T%v", result, err, err)
		return
	}

	if !reflect.DeepEqual(test.Value, result) {
		t.Errorf("  Fail: got result = %T%v", result, result)
	}
}

func (test *Test) testWrite(t *testing.T) {
	t.Logf("Test write %v", test)

	resultBuf := new(bytes.Buffer)
	err := test.Value.Write(resultBuf)
	if err != nil {
		t.Errorf("  Fail: failed to write %#v: %T%v", test.Value, err, err)
		return
	}

	result := resultBuf.Bytes()
	n, err := test.Serialized.Match(result)

	if err != nil || n != len(result) {
		t.Errorf("  Fail: got result = %x:\n    %v", result, err)
	}
}

func TestSerialization(t *testing.T) {
	tests := []Test{
		{te.LiteralString("\x01"), &Byte{1}},
		{te.LiteralString("\x10\x20"), &Short{0x1020}},
		{te.LiteralString("\x10\x20\x30\x40"), &Int{0x10203040}},
		{te.LiteralString("\x10\x20\x30\x40\x50\x60\x70\x80"), &Long{0x1020304050607080}},
		{te.LiteralString("\x3f\x80\x00\x00"), &Float{1.0}},
		{te.LiteralString("\x3f\xf0\x00\x00\x00\x00\x00\x00"), &Double{1.0}},
		{te.LiteralString("\x00\x00\x00\x04\x00\x01\x02\x03"), &ByteArray{[]byte{0, 1, 2, 3}}},
		{te.LiteralString("\x00\x03foo"), &String{"foo"}},
		{te.LiteralString("\x01\x00\x00\x00\x02\x01\x02"), &List{TagByte, []ITag{&Byte{1}, &Byte{2}}}},
		{te.LiteralString("\x03\x00\x00\x00\x02\x00\x00\x00\x01\x00\x00\x00\x02"), &List{TagInt, []ITag{&Int{1}, &Int{2}}}},

		{
			// Single item Compound.
			te.InOrder(
				te.LiteralString("\x01\x00\x03foo\x01"), // NamedTag "foo" Byte{1}
				te.LiteralString("\x00"),                // End
			),
			&Compound{
				map[string]ITag{
					"foo": &Byte{1},
				},
			},
		},
		{
			// Multiple item Compound.
			te.InOrder(
				te.AnyOrder(
					// NamedTag "Byte" Byte{1}
					te.LiteralString("\x01\x00\x04Byte\x01"),
					// NamedTag "Short" Short{2}
					te.LiteralString("\x02\x00\x05Short\x00\x02"),
					// NamedTag "Int" Int{3}
					te.LiteralString("\x03\x00\x03Int\x00\x00\x00\x03"),
					// NamedTag "Long" Long{4}
					te.LiteralString("\x04\x00\x04Long\x00\x00\x00\x00\x00\x00\x00\x04"),
					// NamedTag "Float" Float{5}
					te.LiteralString("\x05\x00\x05Float\x40\xa0\x00\x00"),
					// NamedTag "Double" Double{6}
					te.LiteralString("\x06\x00\x06Double\x40\x18\x00\x00\x00\x00\x00\x00"),
					// NamedTag "String" String{"foo"}
					te.LiteralString("\x08\x00\x06String\x00\x03foo"),
					// NamedTag "List" List{Byte{1}, Byte{2}}
					te.LiteralString("\x09\x00\x04List\x01\x00\x00\x00\x02\x01\x02"),
				),
				te.LiteralString("\x00"), // End
			),
			&Compound{
				map[string]ITag{
					"Byte":   &Byte{1},
					"Short":  &Short{2},
					"Int":    &Int{3},
					"Long":   &Long{4},
					"Float":  &Float{5},
					"Double": &Double{6},
					"String": &String{"foo"},
					"List":   &List{TagByte, []ITag{&Byte{1}, &Byte{2}}},
				},
			},
		},
	}

	for i := range tests {
		tests[i].testRead(t)
		tests[i].testWrite(t)
	}
}

func Test_ReadAndWrite(t *testing.T) {
	compound := &Compound{
		map[string]ITag{
			"Data": &Compound{
				map[string]ITag{
					"Byte": &Byte{5},
				},
			},
		},
	}

	serialized := []byte("" +
		"\x0a\x00\x00" + // Empty name containing Compound
		"\x0a\x00\x04Data" + // "Data" Compound
		"\x01\x00\x04Byte\x05" + // Byte{5}
		"\x00\x00") // End of both compounds.
	reader := bytes.NewBuffer(serialized)

	result, err := Read(reader)

	if err != nil {
		t.Fatalf("Got Read error: %v", err)
	}

	if !reflect.DeepEqual(compound, result) {
		t.Errorf("Got unexpected result: %#v", result)
	}

	// Test writing back out
	writer := new(bytes.Buffer)
	if err = Write(writer, compound); err != nil {
		t.Fatalf("Got Write error: %v", err)
		return
	}

	matcher := &te.BytesLiteral{serialized}
	if err = te.Matches(matcher, writer.Bytes()); err != nil {
		t.Errorf("Got unexpected output from Write: %v", err)
	}
}

func Test_Lookup(t *testing.T) {
	root := &Compound{
		map[string]ITag{
			"Data": &Compound{
				map[string]ITag{
					"Byte":   &Byte{1},
					"Short":  &Short{2},
					"Int":    &Int{3},
					"Long":   &Long{4},
					"Float":  &Float{5},
					"Double": &Double{6},
					"String": &String{"foo"},
					"List":   &List{TagByte, []ITag{&Byte{1}, &Byte{2}}},
				},
			},
		},
	}

	var ok bool
	var tag ITag

	// Absolute lookup from root.
	tag = root.Lookup("Data")
	if _, ok = tag.(*Compound); !ok {
		t.Fatalf("Failed to look up /Data Compound, got: %#v", tag)
	}

	tag = root.Lookup("Data/Byte")
	if _, ok = tag.(*Byte); !ok {
		t.Fatalf("Failed to look up /Data/Byte, got: %#v", tag)
	}

	tag = root.Lookup("Data/List")
	if _, ok = tag.(*List); !ok {
		t.Fatalf("Failed to look up /Data/List, got: %#v", tag)
	}

	// Relative lookup from compound.
	compound := root.Lookup("Data")
	tag = compound.Lookup("Byte")
	if _, ok = tag.(*Byte); !ok {
		t.Fatalf("Failed to look up Byte, got: %#v", tag)
	}
}
