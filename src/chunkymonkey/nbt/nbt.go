package nbt

import (
	"os"
	"io"
	"fmt"
	"strings"
	"encoding/binary"
)

const (
	// Tag types
	TagEnd       = 0
	TagByte      = 1
	TagShort     = 2
	TagInt       = 3
	TagLong      = 4
	TagFloat     = 5
	TagDouble    = 6
	TagByteArray = 7
	TagString    = 8
	TagList      = 9
	TagCompound  = 10
	TagNamed     = 0x80
)

type Tag interface {
	GetType() byte
	Read(io.Reader) os.Error
	Lookup(path string) Tag
}

func NewTagByType(tagType byte) (tag Tag) {
	switch tagType {
	case TagEnd:
		tag = new(End)
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
		panic(fmt.Sprintf("Invalid NBT tag type %#x", tagType))
	}
	return
}

type End struct{}

func (end *End) GetType() byte {
	return TagEnd
}

func (end *End) Read(io.Reader) os.Error {
	return nil
}

func (end *End) Lookup(path string) Tag {
	return nil
}

type NamedTag struct {
	Name string
	Tag  Tag
}

func (n *NamedTag) GetType() byte {
	return TagNamed | n.Tag.GetType()
}

func (n *NamedTag) Read(reader io.Reader) (err os.Error) {
	var tagType byte
	err = binary.Read(reader, binary.BigEndian, &tagType)
	if err != nil {
		return
	}

	var name String
	if tagType != TagEnd {
		err = name.Read(reader)
		if err != nil {
			return
		}
	}

	var value = NewTagByType(tagType)
	err = value.Read(reader)
	if err != nil {
		return
	}

	n.Name = name.Value
	n.Tag = value
	return
}

func (n *NamedTag) Lookup(path string) Tag {
	components := strings.Split(path, "/", 2)
	if components[0] != n.Name {
		return nil
	}

	if len(components) == 1 {
		return n.Tag
	}

	return n.Tag.Lookup(components[1])
}

type Byte struct {
	Value int8
}

func (*Byte) GetType() byte {
	return TagByte
}

func (*Byte) Lookup(path string) Tag {
	return nil
}

func (b *Byte) Read(reader io.Reader) (err os.Error) {
	return binary.Read(reader, binary.BigEndian, &b.Value)
}

type Short struct {
	Value int16
}

func (*Short) GetType() byte {
	return TagShort
}

func (s *Short) Read(reader io.Reader) (err os.Error) {
	return binary.Read(reader, binary.BigEndian, &s.Value)
}

func (*Short) Lookup(path string) Tag {
	return nil
}

type Int struct {
	Value int32
}

func (*Int) GetType() byte {
	return TagInt
}

func (i *Int) Read(reader io.Reader) (err os.Error) {
	return binary.Read(reader, binary.BigEndian, &i.Value)
}

func (*Int) Lookup(path string) Tag {
	return nil
}

type Long struct {
	Value int64
}

func (*Long) GetType() byte {
	return TagLong
}

func (l *Long) Read(reader io.Reader) (err os.Error) {
	return binary.Read(reader, binary.BigEndian, &l.Value)
}

func (*Long) Lookup(path string) Tag {
	return nil
}

type Float struct {
	Value float32
}

func (*Float) GetType() byte {
	return TagFloat
}

func (f *Float) Read(reader io.Reader) (err os.Error) {
	return binary.Read(reader, binary.BigEndian, &f.Value)
}

func (*Float) Lookup(path string) Tag {
	return nil
}

type Double struct {
	Value float64
}

func (*Double) GetType() byte {
	return TagDouble
}

func (d *Double) Read(reader io.Reader) (err os.Error) {
	return binary.Read(reader, binary.BigEndian, &d.Value)
}

func (*Double) Lookup(path string) Tag {
	return nil
}

type ByteArray struct {
	Value []byte
}

func (*ByteArray) GetType() byte {
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

func (*ByteArray) Lookup(path string) Tag {
	return nil
}

type String struct {
	Value string
}

func (*String) GetType() byte {
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

func (*String) Lookup(path string) Tag {
	return nil
}

type List struct {
	Value []Tag
}

func (*List) GetType() byte {
	return TagList
}

func (l *List) Read(reader io.Reader) (err os.Error) {
	var tagType Byte
	err = tagType.Read(reader)
	if err != nil {
		return
	}

	var length Int
	err = length.Read(reader)
	if err != nil {
		return
	}

	list := make([]Tag, length.Value)
	for i, _ := range list {
		tag := NewTagByType(byte(tagType.Value))
		err = tag.Read(reader)
		if err != nil {
			return
		}

		list[i] = tag
	}

	l.Value = list
	return
}

func (*List) Lookup(path string) Tag {
	return nil
}

type Compound struct {
	Tags map[string]*NamedTag
}

func (*Compound) GetType() byte {
	return TagCompound
}

func (c *Compound) Read(reader io.Reader) (err os.Error) {
	tags := make(map[string]*NamedTag)
	for {
		tag := &NamedTag{}
		err = tag.Read(reader)
		if err != nil {
			return
		}

		if tag.GetType() == TagNamed|TagEnd {
			break
		}

		tags[tag.Name] = tag
	}

	c.Tags = tags
	return
}

func (c *Compound) Lookup(path string) (tag Tag) {
	components := strings.Split(path, "/", 2)
	tag, ok := c.Tags[components[0]]
	if !ok {
		return nil
	}

	return tag.Lookup(path)
}

func Read(reader io.Reader) (compound *NamedTag, err os.Error) {
	compound = &NamedTag{}
	err = compound.Read(reader)
	if err != nil {
		return
	}

	if compound.GetType() != TagNamed|TagCompound {
		return nil, os.NewError("Expected named compound tag")
	}
	return
}
