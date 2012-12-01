package proto

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"reflect"
	"unicode/utf8"
)

// Possible error values for reading and writing packets.
var (
	ErrorPacketNotPtr      = errors.New("packet not passed as a pointer")
	ErrorUnknownPacketType = errors.New("unknown packet type")
	ErrorPacketNil         = errors.New("packet was passed by a nil pointer")
	ErrorLengthNegative    = errors.New("length was negative")
	ErrorStrTooLong        = errors.New("string was too long")
	ErrorBadPacketData     = errors.New("packet data well-formed but contains out of range values")
	ErrorBadChunkDataSize  = errors.New("map chunk data length mismatches with size")
	ErrorMismatchingValues = errors.New("packet data contains mismatching values")
	ErrorInternal          = errors.New("implementation problem with packetization")
)

type ErrorUnexpectedPacketId byte

func (err ErrorUnexpectedPacketId) Error() string {
	return fmt.Sprintf("unexpected packet ID 0x%02x", byte(err))
}

type ErrorUnknownPacketId byte

func (err ErrorUnknownPacketId) Error() string {
	return fmt.Sprintf("unknown packet ID 0x%02x", byte(err))
}

var (
	// Space to read unwanted data into. As the contents of this aren't used, it
	// doesn't require syncronization.
	dump [4096]byte
)

// IMinecraftMarshaler is the interface by which packet fields (or potentially
// even whole packets) can customize their serialization. It will only work for
// struct and slice-based types currently, as a hacky method of optimizing
// which packet fields are checked for this property.
type IMarshaler interface {
	MinecraftUnmarshal(reader io.Reader, ps *PacketSerializer) error
	MinecraftMarshal(writer io.Writer, ps *PacketSerializer) error
}

// PacketSerializer reads and writes packets. It is not safe to use one
// simultaneously between multiple goroutines.
//
// It does not take responsibility for reading/writing the packet ID byte
// header.
//
// It is designed to read and write struct types, and can only handle a few
// types - it is not a generalized serialization mechanism and isn't intended
// to be one. It exercises the freedom of having only limited types of packet
// structure partly for simplicity, and partly to allow for optimizations.
type PacketSerializer struct {
	// Scratch space to be able to encode up to 32 bytes without allocating.
	scratch [32]byte
}

func (ps *PacketSerializer) ReadPacketExpect(reader io.Reader, fromClient bool, pktIds ...byte) (packet IPacket, err error) {
	// Read packet ID.
	if _, err = io.ReadFull(reader, ps.scratch[0:1]); err != nil {
		return
	}

	pktId := ps.scratch[0]

	for _, expPktId := range pktIds {
		if expPktId == pktId {
			return ps.readPacketCommon(reader, fromClient, pktId)
		}
	}

	return nil, ErrorUnexpectedPacketId(pktId)
}

func (ps *PacketSerializer) ReadPacket(reader io.Reader, fromClient bool) (packet IPacket, err error) {
	// Read packet ID.
	if _, err = io.ReadFull(reader, ps.scratch[0:1]); err != nil {
		return
	}

	return ps.readPacketCommon(reader, fromClient, ps.scratch[0])
}

func (ps *PacketSerializer) readPacketCommon(reader io.Reader, fromClient bool, id byte) (packet IPacket, err error) {
	pktInfo := &pktIdInfo[ps.scratch[0]]
	if !pktInfo.validPacket {
		return nil, ErrorUnknownPacketType
	}

	var expected bool
	if fromClient {
		expected = pktInfo.clientToServer
	} else {
		expected = pktInfo.serverToClient
	}
	if !expected {
		return nil, ErrorUnexpectedPacketId(ps.scratch[0])
	}

	value := reflect.New(pktInfo.pktType)
	if err = ps.readData(reader, reflect.Indirect(value)); err != nil {
		return
	}

	return value.Interface().(IPacket), nil
}

func (ps *PacketSerializer) readData(reader io.Reader, value reflect.Value) (err error) {
	kind := value.Kind()

	switch kind {
	case reflect.Struct:
		valuePtr := value.Addr()
		if valueMarshaller, ok := valuePtr.Interface().(IMarshaler); ok {
			// Get the value to read itself.
			return valueMarshaller.MinecraftUnmarshal(reader, ps)
		}

		numField := value.NumField()
		for i := 0; i < numField; i++ {
			field := value.Field(i)
			if err = ps.readData(reader, field); err != nil {
				return
			}
		}

	case reflect.Slice:
		valuePtr := value.Addr()
		if valueMarshaller, ok := valuePtr.Interface().(IMarshaler); ok {
			// Get the value to read itself.
			return valueMarshaller.MinecraftUnmarshal(reader, ps)
		} else {
			return ErrorInternal
		}

	case reflect.Bool:
		if v, err := ps.readBool(reader); err != nil {
			return err
		} else {
			value.SetBool(v)
		}

		// Integer types:

	case reflect.Int8:
		if v, err := ps.readUint8(reader); err != nil {
			return err
		} else {
			value.SetInt(int64(int8(v)))
		}
	case reflect.Int16:
		if v, err := ps.readUint16(reader); err != nil {
			return err
		} else {
			value.SetInt(int64(int16(v)))
		}
	case reflect.Int32:
		if v, err := ps.readUint32(reader); err != nil {
			return err
		} else {
			value.SetInt(int64(int32(v)))
		}
	case reflect.Int64:
		if v, err := ps.readUint64(reader); err != nil {
			return err
		} else {
			value.SetInt(int64(v))
		}
	case reflect.Uint8:
		if v, err := ps.readUint8(reader); err != nil {
			return err
		} else {
			value.SetUint(uint64(v))
		}
	case reflect.Uint16:
		if v, err := ps.readUint16(reader); err != nil {
			return err
		} else {
			value.SetUint(uint64(v))
		}
	case reflect.Uint32:
		if v, err := ps.readUint32(reader); err != nil {
			return err
		} else {
			value.SetUint(uint64(v))
		}
	case reflect.Uint64:
		if v, err := ps.readUint64(reader); err != nil {
			return err
		} else {
			value.SetUint(v)
		}

		// Floating point types:

	case reflect.Float32:
		if v, err := ps.readFloat32(reader); err != nil {
			return err
		} else {
			value.SetFloat(float64(v))
		}
	case reflect.Float64:
		if v, err := ps.readFloat64(reader); err != nil {
			return err
		} else {
			value.SetFloat(v)
		}

	case reflect.String:
		// TODO Maybe the tag field could/should suggest a max length.
		str, err := ps.readString16(reader)
		if err != nil {
			return err
		}
		value.SetString(str)

	default:
		typ := value.Type()
		log.Printf("Unimplemented type in packet: %v", typ)
		return ErrorInternal
	}
	return
}

func (ps *PacketSerializer) WritePacket(writer io.Writer, packet IPacket) (err error) {
	value := reflect.Indirect(reflect.ValueOf(packet))
	pktType := value.Type()

	// Write packet ID.
	var ok bool
	if ps.scratch[0], ok = pktTypeId[pktType]; !ok {
		return ErrorUnknownPacketType
	}
	if _, err = writer.Write(ps.scratch[0:1]); err != nil {
		return
	}

	return ps.writeData(writer, value)
}

func (ps *PacketSerializer) WritePacketsBuffer(buf *bytes.Buffer, packets ...IPacket) {
	for _, pkt := range packets {
		if err := ps.WritePacket(buf, pkt); err != nil {
			// bytes.Buffer should never return an error, and any IPacket that isn't
			// serializable is a programming error.
			panic(err)
		}
	}
}

func (ps *PacketSerializer) SerializePackets(packets ...IPacket) []byte {
	buf := new(bytes.Buffer)
	ps.WritePacketsBuffer(buf, packets...)
	return buf.Bytes()
}

func (ps *PacketSerializer) writeData(writer io.Writer, value reflect.Value) (err error) {
	kind := value.Kind()

	switch kind {
	case reflect.Struct:
		valuePtr := value.Addr()
		if valueMarshaller, ok := valuePtr.Interface().(IMarshaler); ok {
			// Get the value to write itself.
			return valueMarshaller.MinecraftMarshal(writer, ps)
		}

		numField := value.NumField()
		for i := 0; i < numField; i++ {
			field := value.Field(i)
			if err = ps.writeData(writer, field); err != nil {
				return
			}
		}

	case reflect.Slice:
		valuePtr := value.Addr()
		if valueMarshaller, ok := valuePtr.Interface().(IMarshaler); ok {
			// Get the value to write itself.
			return valueMarshaller.MinecraftMarshal(writer, ps)
		} else {
			return ErrorInternal
		}

	case reflect.Bool:
		err = ps.writeBool(writer, value.Bool())

		// Integer types:

	case reflect.Int8:
		err = ps.writeUint8(writer, uint8(int8(value.Int())))
	case reflect.Int16:
		err = ps.writeUint16(writer, uint16(int16(value.Int())))
	case reflect.Int32:
		err = ps.writeUint32(writer, uint32(int32(value.Int())))
	case reflect.Int64:
		err = ps.writeUint64(writer, uint64(int64(value.Int())))
	case reflect.Uint8:
		err = ps.writeUint8(writer, uint8(value.Uint()))
	case reflect.Uint16:
		err = ps.writeUint16(writer, uint16(value.Uint()))
	case reflect.Uint32:
		err = ps.writeUint32(writer, uint32(value.Uint()))
	case reflect.Uint64:
		err = ps.writeUint64(writer, value.Uint())

		// Floating point types:

	case reflect.Float32:
		err = ps.writeFloat32(writer, float32(value.Float()))
	case reflect.Float64:
		err = ps.writeFloat64(writer, value.Float())

	case reflect.String:
		err = ps.writeString16(writer, value.String())

	default:
		typ := value.Type()
		log.Printf("Unimplemented type in packet: %v", typ)
		return ErrorInternal
	}

	return
}

// read/write bool.
func (ps *PacketSerializer) readBool(reader io.Reader) (v bool, err error) {
	vUint8, err := ps.readUint8(reader)
	return vUint8 != 0, err
}
func (ps *PacketSerializer) writeBool(writer io.Writer, v bool) (err error) {
	if v {
		return ps.writeUint8(writer, 1)
	}
	return ps.writeUint8(writer, 0)
}

// read/write uint8.
func (ps *PacketSerializer) readUint8(reader io.Reader) (v uint8, err error) {
	if _, err = io.ReadFull(reader, ps.scratch[0:1]); err != nil {
		return
	}
	return ps.scratch[0], nil
}
func (ps *PacketSerializer) writeUint8(writer io.Writer, v uint8) (err error) {
	ps.scratch[0] = v
	_, err = writer.Write(ps.scratch[0:1])
	return
}

// read/write uint16.
func (ps *PacketSerializer) readUint16(reader io.Reader) (v uint16, err error) {
	if _, err = io.ReadFull(reader, ps.scratch[0:2]); err != nil {
		return
	}
	return binary.BigEndian.Uint16(ps.scratch[0:2]), nil
}
func (ps *PacketSerializer) writeUint16(writer io.Writer, v uint16) (err error) {
	binary.BigEndian.PutUint16(ps.scratch[0:2], v)
	_, err = writer.Write(ps.scratch[0:2])
	return
}

// read/write uint32.
func (ps *PacketSerializer) readUint32(reader io.Reader) (v uint32, err error) {
	if _, err = io.ReadFull(reader, ps.scratch[0:4]); err != nil {
		return
	}
	return binary.BigEndian.Uint32(ps.scratch[0:4]), nil
}
func (ps *PacketSerializer) writeUint32(writer io.Writer, v uint32) (err error) {
	binary.BigEndian.PutUint32(ps.scratch[0:4], v)
	_, err = writer.Write(ps.scratch[0:4])
	return
}

// read/write uint64.
func (ps *PacketSerializer) readUint64(reader io.Reader) (v uint64, err error) {
	if _, err = io.ReadFull(reader, ps.scratch[0:8]); err != nil {
		return
	}
	return binary.BigEndian.Uint64(ps.scratch[0:8]), nil
}
func (ps *PacketSerializer) writeUint64(writer io.Writer, v uint64) (err error) {
	binary.BigEndian.PutUint64(ps.scratch[0:8], v)
	_, err = writer.Write(ps.scratch[0:8])
	return
}

// read/write float32.
func (ps *PacketSerializer) readFloat32(reader io.Reader) (v float32, err error) {
	var vUint32 uint32
	if vUint32, err = ps.readUint32(reader); err != nil {
		return
	}
	return math.Float32frombits(vUint32), nil
}
func (ps *PacketSerializer) writeFloat32(writer io.Writer, v float32) (err error) {
	return ps.writeUint32(writer, math.Float32bits(v))
}

// read/write float64.
func (ps *PacketSerializer) readFloat64(reader io.Reader) (v float64, err error) {
	var vUint64 uint64
	if vUint64, err = ps.readUint64(reader); err != nil {
		return
	}
	return math.Float64frombits(vUint64), nil
}
func (ps *PacketSerializer) writeFloat64(writer io.Writer, v float64) (err error) {
	return ps.writeUint64(writer, math.Float64bits(v))
}

// read/write string16
func (ps *PacketSerializer) readString16(reader io.Reader) (v string, err error) {
	lengthUint16, err := ps.readUint16(reader)
	if err != nil {
		return
	}
	length := int(int16(lengthUint16))
	if length < 0 {
		return "", ErrorLengthNegative
	}

	// Most likely that the string will be this long, assuming no non-ASCII characters.
	output := make([]byte, 0, length)
	maxCp := len(ps.scratch) >> 1
	var encChar [4]byte

	for cpToRead := length; cpToRead > 0; {
		curCpToRead := cpToRead
		if curCpToRead > maxCp {
			curCpToRead = maxCp
		}
		cpToRead -= curCpToRead
		bytesToRead := curCpToRead << 1

		// Read UCS-2BE data.
		if _, err = io.ReadFull(reader, ps.scratch[0:bytesToRead]); err != nil {
			return
		}

		// Extract codepoints.
		for i := 0; i < bytesToRead; i += 2 {
			codepoint := (rune(ps.scratch[i]) << 8) | rune(ps.scratch[i+1])

			nBytes := utf8.EncodeRune(encChar[:], codepoint)
			for j := 0; j < nBytes; j++ {
				output = append(output, encChar[j])
			}
		}
	}

	return string(output), nil
}
func (ps *PacketSerializer) writeString16(writer io.Writer, v string) (err error) {
	if err = ps.writeUint16(writer, uint16(utf8.RuneCountInString(v))); err != nil {
		return
	}

	outIndex := 0
	for _, cp := range v {
		if cp > maxUcs2Char {
			cp = ucs2ReplChar
		}
		ps.scratch[outIndex] = byte(cp >> 8)
		ps.scratch[outIndex+1] = byte(cp & 0xff)
		outIndex += 2

		if outIndex >= len(ps.scratch) {
			if _, err = writer.Write(ps.scratch[0:outIndex]); err != nil {
				return
			}
			outIndex = 0
		}
	}

	if outIndex > 0 {
		_, err = writer.Write(ps.scratch[0:outIndex])
	}

	return
}
