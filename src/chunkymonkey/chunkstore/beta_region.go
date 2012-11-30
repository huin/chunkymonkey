package chunkstore

import (
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"path"

	. "chunkymonkey/types"
	"nbt"
)

// TODO Handle timestamps of chunks.
// TODO Keep track of used/unused sectors for more efficient packing.

// Handle on a chunk file - used to read chunk data from the file.
type regionFile struct {
	offsets   regionFileHeader
	endSector uint32 // The index of the sector just beyond the last used sector.
	file      *os.File
}

func newRegionFile(filePath string) (rf *regionFile, err error) {
	file, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		if os.IsNotExist(err) {
			err = NoSuchChunkError(false)
		}
		return
	}

	fi, err := file.Stat()
	if err != nil {
		return
	}

	rf = &regionFile{
		file: file,
	}

	if fi.Size() == 0 {
		// Newly created region file. Create new header index if so.
		if err = rf.offsets.Write(rf.file); err != nil {
			return
		}

		rf.endSector = 2
	} else {
		// Existing region file, read header index.
		if err = rf.offsets.Read(rf.file); err != nil {
			return
		}

		// Find the index of the sector at the end of the file.
		for i := range rf.offsets {
			sectorCount, sectorIndex := rf.offsets[i].Get()
			lastUsedSector := sectorIndex + sectorCount
			if lastUsedSector >= rf.endSector {
				rf.endSector = lastUsedSector + 1
			}
		}

		if rf.endSector < 2 {
			rf.endSector = 2
		}
	}

	return
}

func (rf *regionFile) Close() {
	rf.file.Close()
}

func (rf *regionFile) ReadChunkData(chunkLoc ChunkXz) (r *nbtChunkReader, err error) {
	offset := rf.offsets.Offset(chunkLoc)

	if !offset.IsPresent() {
		// Chunk doesn't exist in file
		err = NoSuchChunkError(false)
		return
	}

	sectorCount, sectorIndex := offset.Get()

	if sectorIndex == 0 || sectorCount == 0 {
		err = errors.New("Header gave bad chunk offset.")
		return
	}

	rf.file.Seek(int64(sectorIndex)*regionFileSectorSize, 0)

	maxChunkDataSize := (sectorCount * regionFileSectorSize) - chunkDataHeaderSize

	var header chunkDataHeader
	binary.Read(rf.file, binary.BigEndian, &header)
	if header.DataSize > maxChunkDataSize {
		err = fmt.Errorf(
			"Chunk is too big (%d bytes) for the sectors it is within (%d*%d - %d=%d) header.",
			header.DataSize, sectorCount, regionFileSectorSize, chunkDataHeaderSize, maxChunkDataSize)
		return
	}

	dataReader, err := header.DataReader(rf.file)
	if err != nil {
		return
	}
	defer dataReader.Close()

	r, err = newNbtChunkReader(dataReader)

	return
}

func (rf *regionFile) WriteChunkData(w *nbtChunkWriter) (err error) {
	chunkData, err := serializeChunkData(w)
	if err != nil {
		return
	}

	requiredSize := uint32(len(chunkData))

	offset := rf.offsets.Offset(w.ChunkLoc())
	sectorCount, sectorIndex := offset.Get()

	if offset.IsPresent() && requiredSize <= sectorCount*regionFileSectorSize {
		// Chunk already exists in the region file and the data will fit in its
		// present location.
		if _, err = rf.file.WriteAt(chunkData, int64(sectorIndex)*regionFileSectorSize); err != nil {
			return
		}
	} else {
		// Chunk doesn't yet exist in the region file or won't fit in its present
		// location. Write it at the end of the file.
		sectorIndex = rf.endSector

		if _, err = rf.file.WriteAt(chunkData, int64(sectorIndex)*regionFileSectorSize); err != nil {
			return
		}

		sectorCount = requiredSize / regionFileSectorSize
		if requiredSize%regionFileSectorSize != 0 {
			sectorCount++
		}
		rf.endSector += sectorCount

		offset.Set(sectorCount, sectorIndex)
		rf.offsets.SetOffset(w.ChunkLoc(), offset, rf.file)
	}

	return
}

// serializeChunkData produces the compressed chunk NBT data.
func serializeChunkData(w *nbtChunkWriter) (chunkData []byte, err error) {
	// Reserve room for the chunk data header at the start.
	buffer := bytes.NewBuffer(make([]byte, chunkDataHeaderSize, chunkDataGuessSize))

	zlibWriter := zlib.NewWriter(buffer)
	if err = nbt.Write(zlibWriter, w.RootTag()); err != nil {
		zlibWriter.Close()
		return nil, err
	}
	if err = zlibWriter.Close(); err != nil {
		return nil, err
	}
	chunkData = buffer.Bytes()

	// Write chunk data header
	header := chunkDataHeader{
		DataSize: uint32(len(chunkData)) - chunkDataHeaderSize,
		Version:  chunkCompressionZlib,
	}
	buffer = bytes.NewBuffer(chunkData[:0])
	if err = binary.Write(buffer, binary.BigEndian, header); err != nil {
		return nil, err
	}

	return chunkData, nil
}

// A chunk file header entry.
type chunkOffset uint32

// Returns true if the offset value states that the chunk is present in the
// file.
func (o chunkOffset) IsPresent() bool {
	return o != 0
}

func (o chunkOffset) Get() (sectorCount, sectorIndex uint32) {
	sectorCount = uint32(o & 0xff)
	sectorIndex = uint32(o >> 8)
	return
}

func (o *chunkOffset) Set(sectorCount, sectorIndex uint32) {
	*o = chunkOffset(sectorIndex<<8 | sectorCount&0xff)
}

// Represents a chunk file header containing chunk data offsets.
type regionFileHeader [regionFileEdge * regionFileEdge]chunkOffset

func (h *regionFileHeader) Read(file *os.File) (err error) {
	if _, err = file.Seek(0, os.SEEK_SET); err != nil {
		return
	}
	return binary.Read(file, binary.BigEndian, h[:])
}

func (h *regionFileHeader) Write(file *os.File) (err error) {
	if _, err = file.Seek(0, os.SEEK_SET); err != nil {
		return
	}
	return binary.Write(file, binary.BigEndian, h[:])
}

// Returns the chunk offset data for the given chunk. It assumes that chunkLoc
// is within the chunk file - discarding upper bits of the X and Z coords.
func (h *regionFileHeader) Offset(chunkLoc ChunkXz) chunkOffset {
	return h[indexForChunkLoc(chunkLoc)]
}

func (h *regionFileHeader) SetOffset(chunkLoc ChunkXz, offset chunkOffset, file *os.File) error {
	index := indexForChunkLoc(chunkLoc)
	h[index] = offset

	// Write that part of the index.
	var offsetBytes [4]byte
	binary.BigEndian.PutUint32(offsetBytes[:], uint32(offset))
	_, err := file.WriteAt(offsetBytes[:], int64(index)*4)

	return err
}

func indexForChunkLoc(chunkLoc ChunkXz) int {
	x := chunkLoc.X & (regionFileEdge - 1)
	z := chunkLoc.Z & (regionFileEdge - 1)
	return int(x + (z << regionFileEdgeShift))
}

// Represents the header of a single chunk of data within a chunkfile.
type chunkDataHeader struct {
	DataSize uint32
	Version  byte
}

// Returns an io.Reader to correctly decompress data from the chunk data.
// The reader passed in must be just after the chunkDataHeader in the source
// data stream. The caller is responsible for closing the returned ReadCloser.
func (cdh *chunkDataHeader) DataReader(raw io.Reader) (output io.ReadCloser, err error) {
	limitReader := io.LimitReader(raw, int64(cdh.DataSize))
	switch cdh.Version {
	case chunkCompressionGzip:
		output, err = gzip.NewReader(limitReader)
	case chunkCompressionZlib:
		output, err = zlib.NewReader(limitReader)
	default:
		err = errors.New("Chunk data header contained unknown version number.")
	}
	return
}

type regionCoord int32

type regionLoc struct {
	X, Z regionCoord
}

func regionLocForChunkXz(chunkLoc ChunkXz) regionLoc {
	return regionLoc{
		regionCoord(chunkLoc.X >> regionFileEdgeShift),
		regionCoord(chunkLoc.Z >> regionFileEdgeShift),
	}
}

func (loc *regionLoc) regionKey() uint64 {
	return uint64(loc.X)<<32 | uint64(uint32(loc.Z))
}

func (loc *regionLoc) regionFilePath(regionPath string) string {
	return path.Join(
		regionPath,
		fmt.Sprintf("r.%d.%d.mcr", loc.X, loc.Z),
	)
}
