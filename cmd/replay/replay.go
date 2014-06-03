package main

import (
	"flag"
	"io"
	"log"
	"net"
	"os"

	"github.com/huin/chunkymonkey/record"
)

func usage() {
	os.Stderr.WriteString("usage: " + os.Args[0] + " server:port file.record\n")
	flag.PrintDefaults()
}

func readAndDiscard(reader io.Reader) {
	buf := make([]byte, 8192)
	for {
		_, err := reader.Read(buf)
		if err != nil {
			return
		}
	}
}

func main() {
	flag.Usage = usage
	flag.Parse()

	if flag.NArg() != 2 {
		flag.Usage()
		os.Exit(1)
	}

	serverAddr := flag.Arg(0)
	recordFilename := flag.Arg(1)

	recordInput, err := os.Open(recordFilename)
	if err != nil {
		log.Fatalf("Failed to open record file %q: %v", recordFilename, err)
	}
	defer recordInput.Close()

	conn, err := net.Dial("tcp", serverAddr)
	if err != nil {
		log.Fatalf("Failed to connect to server %q: %v", serverAddr, err)
	}
	defer conn.Close()

	replayer := record.NewReaderReplayer(recordInput, conn)
	// Discard everything the server sends us.
	go readAndDiscard(conn)
	replayer.Replay()
}
