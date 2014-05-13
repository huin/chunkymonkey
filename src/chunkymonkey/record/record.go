// Wrapper for net.Conn which supports recording and replaying received data

package record

import (
	"encoding/binary"
	"io"
	"os"
	"time"
)

// Log record header
type header struct {
	Timestamp int64 // delay since last data read, in nanoseconds
	Length    int32 // length of data bytes
}

type ReaderRecorder struct {
	reader        io.Reader
	log           io.WriteCloser
	lastTimestamp int64
}

func NewReaderRecorder(log io.WriteCloser, reader io.Reader) *ReaderRecorder {
	return &ReaderRecorder{
		reader:        reader,
		log:           log,
		lastTimestamp: time.Nanoseconds(),
	}
}

func (recorder *ReaderRecorder) Read(b []byte) (n int, err os.Error) {
	n, err = recorder.reader.Read(b)
	if err == nil {
		now := time.Nanoseconds()
		binary.Write(recorder.log, binary.BigEndian, &header{
			now - recorder.lastTimestamp,
			int32(n),
		})
		binary.Write(recorder.log, binary.BigEndian, b[:n])

		recorder.lastTimestamp = now
	}
	return
}

func (recorder *ReaderRecorder) Close() os.Error {
	return recorder.log.Close()
}

type ReaderReplayer struct {
	writer        io.Writer
	log           io.Reader
	lastTimestamp int64
}

func NewReaderReplayer(log io.Reader, writer io.Writer) *ReaderReplayer {
	return &ReaderReplayer{
		writer:        writer,
		log:           log,
		lastTimestamp: time.Nanoseconds(),
	}
}

func (replayer *ReaderReplayer) Replay() {
	var header header
	var err os.Error
	buf := make([]byte, 4096)

	for {
		if err = binary.Read(replayer.log, binary.BigEndian, &header); err != nil {
			break
		}
		if header.Length > int32(len(buf)) {
			buf = make([]byte, header.Length)
		}
		if _, err = replayer.log.Read(buf[:header.Length]); err != nil {
			break
		}

		// Wait until recorded time has passed
		now := time.Nanoseconds()
		delta := now - replayer.lastTimestamp
		if delta < header.Timestamp {
			time.Sleep(header.Timestamp - delta)
		}
		replayer.lastTimestamp = now

		_, err = replayer.writer.Write(buf[:header.Length])
		if err != nil {
			break
		}
	}

	return
}
