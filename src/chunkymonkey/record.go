// Wrapper for net.Conn which supports recording and replaying received data

package record

import (
	"io"
	"os"
	"net"
	"log"
	"flag"
	"time"
	"encoding/binary"
)

// Log record header
type header struct {
	Timestamp int64 // delay since last packet, in nanoseconds
	Length    int32 // length of data bytes
}

type recorder struct {
	conn          net.Conn
	log           io.WriteCloser
	lastTimestamp int64
}

func (recorder *recorder) Read(b []byte) (n int, err os.Error) {
	n, err = recorder.conn.Read(b)
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

func (recorder *recorder) Write(b []byte) (n int, err os.Error) {
	return recorder.conn.Write(b)
}

func (recorder *recorder) Close() os.Error {
	recorder.log.Close()
	return recorder.conn.Close()
}

func (recorder *recorder) LocalAddr() net.Addr {
	return recorder.conn.LocalAddr()
}

func (recorder *recorder) RemoteAddr() net.Addr {
	return recorder.conn.RemoteAddr()
}

func (recorder *recorder) SetTimeout(nsec int64) os.Error {
	return recorder.conn.SetTimeout(nsec)
}

func (recorder *recorder) SetReadTimeout(nsec int64) os.Error {
	return recorder.conn.SetReadTimeout(nsec)
}

func (recorder *recorder) SetWriteTimeout(nsec int64) os.Error {
	return recorder.conn.SetWriteTimeout(nsec)
}

type replayer struct {
	conn          net.Conn
	log           io.ReadCloser
	lastTimestamp int64
}

func (replayer *replayer) Read(b []byte) (n int, err os.Error) {
	var header header

	err = binary.Read(replayer.log, binary.BigEndian, &header)
	if err != nil {
		return 0, err
	}

	if int32(len(b)) < header.Length {
		return 0, os.NewError("replay read length too small")
	}

	// Wait until recorded time has passed
	now := time.Nanoseconds()
	delta := now - replayer.lastTimestamp
	if delta < header.Timestamp {
		time.Sleep(header.Timestamp - delta)
	}
	replayer.lastTimestamp = now

	return replayer.log.Read(b[:header.Length])
}

func (replayer *replayer) Write(b []byte) (n int, err os.Error) {
	return replayer.conn.Write(b)
}

func (replayer *replayer) Close() os.Error {
	replayer.log.Close()
	return replayer.conn.Close()
}

func (replayer *replayer) LocalAddr() net.Addr {
	return replayer.conn.LocalAddr()
}

func (replayer *replayer) RemoteAddr() net.Addr {
	return replayer.conn.RemoteAddr()
}

func (replayer *replayer) SetTimeout(nsec int64) os.Error {
	return replayer.conn.SetTimeout(nsec)
}

func (replayer *replayer) SetReadTimeout(nsec int64) os.Error {
	return replayer.conn.SetReadTimeout(nsec)
}

func (replayer *replayer) SetWriteTimeout(nsec int64) os.Error {
	return replayer.conn.SetWriteTimeout(nsec)
}

var record = flag.String("record", "", "record received packets to file")
var replay = flag.String("replay", "", "replay received packets from file")
var connections = 0

// Interpose a recorder or replayer onto a network connection
func WrapConn(raw net.Conn) (wrapped net.Conn) {
	if *record != "" {
		file, err := os.Create(*record)
		if err != nil {
			log.Fatal("WrapConn: ", err.String())
		}

		return &recorder{raw, file, time.Nanoseconds()}
	}

	// The second client connection will replay the log file
	if connections == 1 && *replay != "" {
		file, err := os.Open(*replay)
		if err != nil {
			log.Fatal("WrapConn: ", err.String())
		}

		return &replayer{raw, file, time.Nanoseconds()}
	}
	connections++

	return raw
}
