package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"

	"chunkymonkey/record"
)

var recordBase = flag.String(
	"record", "", "Record player connections to files with this prefix")

type RelayReport struct {
	written int64
	err     error
}

// Sends a report on reportChan when it completes
func spliceParser(parser func(reader io.Reader), dst io.Writer, src io.Reader) (reportChan chan RelayReport) {

	parserReader, parserWriter := io.Pipe()
	wrappedDst := io.MultiWriter(dst, parserWriter)
	reportChan = make(chan RelayReport)

	go parser(parserReader)
	go func() {
		written, err := io.Copy(wrappedDst, src)
		reportChan <- RelayReport{written, err}
	}()

	return
}

func serveConn(clientConn net.Conn, remoteaddr string, connNumber int) {
	defer clientConn.Close()

	clientAddr := clientConn.RemoteAddr().String()

	logPrefix := fmt.Sprintf("[%d]", connNumber)
	logger := log.New(os.Stderr, logPrefix+" ", log.Ldate|log.Ltime|log.Lmicroseconds)

	logger.Printf("Client connected from %v", clientAddr)

	logger.Printf("Creating relay to server %v", remoteaddr)
	serverConn, err := net.Dial("tcp", remoteaddr)
	if err != nil {
		logger.Printf("Error connecting to server: %v", err)
		return
	}
	defer serverConn.Close()
	logger.Print("Connected to server")

	// clientReader reads data sent from the client.
	clientReader := io.Reader(clientConn)
	if *recordBase != "" {
		recordFilename := fmt.Sprintf("%s-%d.record", *recordBase, connNumber)
		recordOutput, err := os.Create(recordFilename)
		if err != nil {
			logger.Printf(
				"Failed to open file %q to record connection: %v",
				recordFilename, err)
		} else {
			recorder := record.NewReaderRecorder(recordOutput, clientConn)
			defer recorder.Close()
			clientReader = recorder
		}
	}

	clientParser := new(MessageParser)
	serverParser := new(MessageParser)

	// Set up for parsing messages from server to client
	scLogger := log.New(os.Stderr, logPrefix+"(S->C) ", log.Ldate|log.Ltime|log.Lmicroseconds)
	serverToClientReportChan := spliceParser(
		func(reader io.Reader) { serverParser.ScParse(reader, scLogger) },
		clientConn, serverConn)

	// Set up for parsing messages from client to server
	csLogger := log.New(os.Stderr, logPrefix+"(C->S) ", log.Ldate|log.Ltime|log.Lmicroseconds)
	clientToServerReportChan := spliceParser(
		func(reader io.Reader) { clientParser.CsParse(reader, csLogger) },
		serverConn, clientReader)

	// Wait for the both relay/splices to stop, then we let the connections
	// close via deferred calls
	report := <-serverToClientReportChan
	logger.Printf("Server->client relay after %d bytes with error: %v", report.written, report.err)
	report = <-clientToServerReportChan
	logger.Printf("Client->server relay after %d bytes with error: %v", report.written, report.err)

	logger.Print("Client disconnected")
}

func serve(localaddr, remoteaddr string) (err error) {
	listener, err := net.Listen("tcp", localaddr)
	if err != nil {
		log.Fatal("Listen: ", err.Error())
		return
	}

	defer listener.Close()
	log.Print("Listening on ", localaddr)

	connNumber := 1

	for {
		clientConn, acceptErr := listener.Accept()
		if acceptErr != nil {
			log.Printf("Accept error: %s", acceptErr.Error())
			break
		} else {
			go serveConn(clientConn, remoteaddr, connNumber)
			connNumber++
		}
	}
	return
}

func usage() {
	os.Stderr.WriteString("usage: " + os.Args[0] + " [options] localaddr:port remoteaddr:port\n")
	flag.PrintDefaults()
}

func main() {
	flag.Usage = usage
	flag.Parse()

	if flag.NArg() != 2 {
		flag.Usage()
		os.Exit(1)
	}

	localaddr := flag.Arg(0)
	remoteaddr := flag.Arg(1)

	// It's nice to have high time precision when looking at packets
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)

	serve(localaddr, remoteaddr)
}
