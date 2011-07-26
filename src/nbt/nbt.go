// nbt provides the ability to read/write NBT data structures from Readers and
// Writers.
//
// NBT is the data serialization format used in many places in the official
// Notchian Minecraft server, typically to represent structured world, chunk
// and player information.
//
// An NBT data structure can be created with code such as the following:
//
//   root := &Compound{
//     map[string]ITag{
//       "Data": &Compound{
//         map[string]ITag{
//           "Byte":   &Byte{1},
//           "Short":  &Short{2},
//           "Int":    &Int{3},
//           "Long":   &Long{4},
//           "Float":  &Float{5},
//           "Double": &Double{6},
//           "String": &String{"foo"},
//           "List":   &List{TagByte, []ITag{&Byte{1}, &Byte{2}}},
//         },
//       },
//     },
//   }
//
// It is required that the root structure be a Compound for compatibility with
// existing NBT structures observed in the official server.
//
// NBT structures can be read from an io.Reader with the Read function.
package nbt

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"strings"
)

// ITag is the interface for all tags that can be represented in an NBT tree.
type ITag interface {
	Type() TagType
	Read(io.Reader) os.Error
	Write(io.Writer) os.Error
	Lookup(path string) ITag
}

// TagType is the header byte value that identifies the type of tag(s).
type TagType byte

const (
	// Tag types. All these values can be used to create a new tag, except
	// TagEnd.
	TagEnd       = TagType(0)
	TagByte      = TagType(1)
	TagShort     = TagType(2)
	TagInt       = TagType(3)
	TagLong      = TagType(4)
	TagFloat     = TagType(5)
	TagDouble    = TagType(6)
	TagByteArray = TagType(7)
	TagString    = TagType(8)
	TagList      = TagType(9)
	TagCompound  = TagType(10)
)

// NewTag creates a new tag of the given TagType. TagEnd is not a valid value
// here.
func (tt TagType) NewTag() (tag ITag, err os.Error) {
	switch tt {
	case TagByte:
		tag = new(Byte)
	case TagShort:
		tag = new(Short)
	case TagInt:
		tag = new(Int)
	case TagLong:
		tag = new(Long)
	case TagFloat:
		tag = new(Float)
	case TagDouble:
		tag = new(Double)
	case TagByteArray:
		tag = new(ByteArray)
	case TagString:
		tag = new(String)
	case TagList:
		tag = new(List)
	case TagCompound:
		tag = new(Compound)
	default:
		err = fmt.Errorf("invalid NBT tag type %#x", tt)
	}
	return
}

func (tt *TagType) read(reader io.Reader) os.Error {
	return binary.Read(reader, binary.BigEndian, tt)
}

func (tt TagType) write(writer io.Writer) os.Error {
	return binary.Write(writer, binary.BigEndian, tt)
}

type Byte struct {
	Value int8
}

func (*Byte) Type() TagType {
	return TagByte
}

func (*Byte) Lookup(path string) ITag {
	return nil
}

func (b *Byte) Read(reader io.Reader) (err os.Error) {
	return binary.Read(reader, binary.BigEndian, &b.Value)
}

func (b *Byte) Write(writer io.Writer) (err os.Error) {
	return binary.Write(writer, binary.BigEndian, &b.Value)
}

type Short struct {
	Value int16
}

func (*Short) Type() TagType {
	return TagShort
}

func (s *Short) Read(reader io.Reader) (err os.Error) {
	return binary.Read(reader, binary.BigEndian, &s.Value)
}

func (s *Short) Write(writer io.Writer) (err os.Error) {
	return binary.Write(writer, binary.BigEndian, &s.Value)
}

func (*Short) Lookup(path string) ITag {
	return nil
}

type Int struct {
	Value int32
}

func (*Int) Type() TagType {
	return TagInt
}

func (i *Int) Read(reader io.Reader) (err os.Error) {
	return binary.Read(reader, binary.BigEndian, &i.Value)
}

func (i *Int) Write(writer io.Writer) (err os.Error) {
	return binary.Write(writer, binary.BigEndian, &i.Value)
}

func (*Int) Lookup(path string) ITag {
	return nil
}

type Long struct {
	Value int64
}

func (*Long) Type() TagType {
	return TagLong
}

func (l *Long) Read(reader io.Reader) (err os.Error) {
	return binary.Read(reader, binary.BigEndian, &l.Value)
}

func (l *Long) Write(writer io.Writer) (err os.Error) {
	return binary.Write(writer, binary.BigEndian, &l.Value)
}

func (*Long) Lookup(path string) ITag {
	return nil
}

type Float struct {
	Value float32
}

func (*Float) Type() TagType {
	return TagFloat
}

func (f *Float) Read(reader io.Reader) (err os.Error) {
	return binary.Read(reader, binary.BigEndian, &f.Value)
}

func (f *Float) Write(writer io.Writer) (err os.Error) {
	return binary.Write(writer, binary.BigEndian, &f.Value)
}

func (*Float) Lookup(path string) ITag {
	return nil
}

type Double struct {
	Value float64
}

func (*Double) Type() TagType {
	return TagDouble
}

func (d *Double) Read(reader io.Reader) (err os.Error) {
	return binary.Read(reader, binary.BigEndian, &d.Value)
}

func (d *Double) Write(writer io.Writer) (err os.Error) {
	return binary.Write(writer, binary.BigEndian, &d.Value)
}

func (*Double) Lookup(path string) ITag {
	return nil
}

type ByteArray struct {
	Value []byte
}

func (*ByteArray) Type() TagType {
	return TagByteArray
}

func (b *ByteArray) Read(reader io.Reader) (err os.Error) {
	var length Int

	err = length.Read(reader)
	if err != nil {
		return
	}

	bs := make([]byte, length.Value)
	_, err = io.ReadFull(reader, bs)
	if err != nil {
		return
	}

	b.Value = bs
	return
}

func (b *ByteArray) Write(writer io.Writer) (err os.Error) {
	length := Int{int32(len(b.Value))}

	if err = length.Write(writer); err != nil {
		return
	}

	_, err = writer.Write(b.Value)
	return
}

func (*ByteArray) Lookup(path string) ITag {
	return nil
}

type String struct {
	Value string
}

func (*String) Type() TagType {
	return TagString
}

func (s *String) Read(reader io.Reader) (err os.Error) {
	var length Short

	err = length.Read(reader)
	if err != nil {
		return
	}

	bs := make([]byte, length.Value)
	_, err = io.ReadFull(reader, bs)
	if err != nil {
		return
	}

	s.Value = string(bs)
	return
}

func (s *String) Write(writer io.Writer) (err os.Error) {
	length := Short{int16(len(s.Value))}

	if err = length.Write(writer); err != nil {
		return
	}

	_, err = writer.Write([]byte(s.Value))
	return
}

func (*String) Lookup(path string) ITag {
	return nil
}

type List struct {
	TagType TagType
	Value   []ITag
}

func (*List) Type() TagType {
	return TagList
}

func (l *List) Read(reader io.Reader) (err os.Error) {
	if err = l.TagType.read(reader); err != nil {
		return
	}

	var length Int
	err = length.Read(reader)
	if err != nil {
		return
	}

	list := make([]ITag, length.Value)
	for i, _ := range list {
		var tag ITag
		if tag, err = l.TagType.NewTag(); err != nil {
			return
		}
		err = tag.Read(reader)
		if err != nil {
			return
		}

		list[i] = tag
	}

	l.Value = list
	return
}

func (l *List) Write(writer io.Writer) (err os.Error) {
	tagType := Byte{int8(l.TagType)}
	if err = tagType.Write(writer); err != nil {
		return
	}

	length := Int{int32(len(l.Value))}
	if err = length.Write(writer); err != nil {
		return
	}

	for _, tag := range l.Value {
		if err = tag.Write(writer); err != nil {
			return
		}
	}

	return
}

func (*List) Lookup(path string) ITag {
	return nil
}

type Compound struct {
	Tags map[string]ITag
}

func (*Compound) Type() TagType {
	return TagCompound
}

func readTagAndName(reader io.Reader) (tag ITag, name string, err os.Error) {
	var tagType TagType
	if tagType.read(reader); err != nil {
		return
	}

	if tagType == TagEnd {
		return
	}

	var nameTag String
	if err = nameTag.Read(reader); err != nil {
		return
	}

	name = nameTag.Value

	if tag, err = tagType.NewTag(); err != nil {
		return
	}
	err = tag.Read(reader)

	return
}

func (c *Compound) Read(reader io.Reader) (err os.Error) {
	tags := make(map[string]ITag)
	var tag ITag
	var tagName string

	for {
		if tag, tagName, err = readTagAndName(reader); err != nil {
			return
		}

		if tag == nil {
			break
		}

		tags[tagName] = tag
	}

	c.Tags = tags
	return
}

func writeTagAndName(writer io.Writer, tag ITag, name string) (err os.Error) {
	if err = tag.Type().write(writer); err != nil {
		return
	}

	nameTag := String{name}
	if err = nameTag.Write(writer); err != nil {
		return
	}

	err = tag.Write(writer)

	return
}

func (c *Compound) Write(writer io.Writer) (err os.Error) {
	for name, tag := range c.Tags {
		if err = writeTagAndName(writer, tag, name); err != nil {
			return
		}
	}

	err = TagEnd.write(writer)

	return
}

func (c *Compound) Lookup(path string) (tag ITag) {
	components := strings.Split(path, "/", 2)
	tag, ok := c.Tags[components[0]]
	if !ok {
		return nil
	}

	if len(components) >= 2 {
		return tag.Lookup(components[1])
	}

	return tag
}

// Read reads an NBT structure from the given reader. It expects it to contain
// a Compound at the root.
func Read(reader io.Reader) (tag ITag, err os.Error) {
	var name string
	if tag, name, err = readTagAndName(reader); err != nil {
		return
	}

	if name != "" {
		return nil, os.NewError("Root name should be empty")
	}

	if tag.Type() != TagCompound {
		return nil, os.NewError("Expected named compound tag")
	}

	return
}

// Write writes an NBT structure to the given writer. It should be passed a
// *Compound, but doesn't require it.
func Write(writer io.Writer, tag ITag) (err os.Error) {
	return writeTagAndName(writer, tag, "")
}
