package chunkstore

import (
	"compress/gzip"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path"

	. "chunkymonkey/types"
)

const (
	regionFileEdge       = 32
	regionFileEdgeShift  = 5
	regionFileSectorSize = 4096
)

type chunkStoreBeta struct {
	worldPath   string
	regionFiles map[uint64]*regionFileReader
}

// Creates a ChunkStore that reads the Minecraft Beta world format.
func NewChunkStoreBeta(worldPath string) ChunkStore {
	return &chunkStoreBeta{
		worldPath:   worldPath,
		regionFiles: make(map[uint64]*regionFileReader),
	}
}

func (s *chunkStoreBeta) LoadChunk(chunkLoc *ChunkXz) (reader ChunkReader, err os.Error) {
	regionLoc := regionLocForChunkXz(chunkLoc)

	var cfr *regionFileReader
	cfr, ok := s.regionFiles[regionLoc.regionKey()]
	if !ok {
		// TODO limit number of regionFileReader objs to a maximum number of
		// most-frequently-used regions. Close regionFileReader objects when no
		// longer needed.
		filePath := regionLoc.regionFilePath(s.worldPath)
		cfr, err = newRegionFileReader(filePath)
		if err != nil {
			return
		}
		s.regionFiles[regionLoc.regionKey()] = cfr
	}

	chunkReader, err := cfr.ReadChunkData(chunkLoc)
	if chunkReader != nil {
		reader = chunkReader
	}

	return
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

// Represents a chunk file header containing chunk data offsets.
type regionFileHeader [regionFileEdge * regionFileEdge]chunkOffset

// Returns the chunk offset data for the given chunk. It assumes that chunkLoc
// is within the chunk file - discarding upper bits of the X and Z coords.
func (h regionFileHeader) GetOffset(chunkLoc *ChunkXz) chunkOffset {
	x := chunkLoc.X & (regionFileEdge - 1)
	z := chunkLoc.Z & (regionFileEdge - 1)
	return h[x+(z<<regionFileEdgeShift)]
}

// Represents the header of a single chunk of data within a chunkfile.
type chunkDataHeader struct {
	DataSize uint32
	Version  byte
}

// Returns an io.Reader to correctly decompress data from the chunk data.
// The reader passed in must be just after the chunkDataHeader in the source
// data stream. The caller is responsible for closing the returned ReadCloser.
func (cdh *chunkDataHeader) GetDataReader(raw io.Reader) (output io.ReadCloser, err os.Error) {
	limitReader := io.LimitReader(raw, int64(cdh.DataSize))
	switch cdh.Version {
	case 1:
		output, err = gzip.NewReader(limitReader)
	case 2:
		output, err = zlib.NewReader(limitReader)
	default:
		err = os.NewError("Chunk data header contained unknown version number.")
	}
	return
}

// Handle on a chunk file - used to read chunk data from the file.
type regionFileReader struct {
	offsets regionFileHeader
	file    *os.File
}

func newRegionFileReader(filePath string) (cfr *regionFileReader, err os.Error) {
	file, err := os.Open(filePath)
	if err != nil {
		if sysErr, ok := err.(*os.SyscallError); ok && sysErr.Errno == os.ENOENT {
			err = NoSuchChunkError(false)
		}
		return
	}

	cfr = &regionFileReader{
		file: file,
	}

	err = binary.Read(file, binary.BigEndian, &cfr.offsets)
	if err != nil {
		cfr = nil
		return
	}

	return
}

func (cfr *regionFileReader) Close() {
	cfr.file.Close()
}

func (cfr *regionFileReader) ReadChunkData(chunkLoc *ChunkXz) (r *chunkReader, err os.Error) {
	offset := cfr.offsets.GetOffset(chunkLoc)

	if !offset.IsPresent() {
		// Chunk doesn't exist in file
		err = NoSuchChunkError(false)
		return
	}

	sectorCount, sectorIndex := offset.Get()

	if sectorIndex == 0 || sectorCount == 0 {
		err = os.NewError("Header gave bad chunk offset.")
		return
	}

	cfr.file.Seek(int64(sectorIndex)*regionFileSectorSize, 0)

	// 5 is the size of chunkDataHeader in bytes.
	maxChunkDataSize := (sectorCount * regionFileSectorSize) - 5

	var header chunkDataHeader
	binary.Read(cfr.file, binary.BigEndian, &header)
	if header.DataSize > maxChunkDataSize {
		err = os.NewError("Chunk is too big for the sectors it is within.")
		return
	}

	dataReader, err := header.GetDataReader(cfr.file)
	if err != nil {
		return
	}
	defer dataReader.Close()

	r, err = newChunkReader(dataReader)

	return
}

type regionCoord int32

type regionLoc struct {
	X, Z regionCoord
}

func regionLocForChunkXz(chunkLoc *ChunkXz) regionLoc {
	return regionLoc{
		regionCoord(chunkLoc.X >> regionFileEdgeShift),
		regionCoord(chunkLoc.Z >> regionFileEdgeShift),
	}
}

func (loc *regionLoc) regionKey() uint64 {
	return uint64(loc.X)<<32 | uint64(uint32(loc.Z))
}

func (loc *regionLoc) regionFilePath(worldPath string) string {
	return path.Join(
		worldPath,
		"region",
		fmt.Sprintf("r.%d.%d.mcr", loc.X, loc.Z),
	)
}
