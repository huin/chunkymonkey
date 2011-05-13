package main

import (
	"flag"
	"io"
	"log"
	"net"
	"os"
	parser "util/intercept/intercept_parse"
)

type RelayReport struct {
	written int64
	err     os.Error
}

// Sends a report on reportChan when it completes
func spliceParser(parser func(reader io.Reader),
dst io.Writer, src io.Reader) (reportChan chan RelayReport) {

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

func serveConn(clientConn net.Conn, remoteaddr string) {
	defer clientConn.Close()

	clientAddr := clientConn.RemoteAddr().String()

	log.Printf("(%s) Client connected ", clientAddr)

	log.Printf("(%s) Creating relay to server %s", clientAddr, remoteaddr)
	serverConn, err := net.Dial("tcp", remoteaddr)

	if err != nil {
		log.Printf("(%s) Error connecting to %s: %v", clientAddr, remoteaddr, err)
		return
	}

	defer serverConn.Close()

	log.Printf("(%s) Connected to server %s", clientAddr, remoteaddr)

	clientParser := new(parser.MessageParser)
	serverParser := new(parser.MessageParser)

	// Set up for parsing messages from server to client
	serverToClientReportChan := spliceParser(
		func(reader io.Reader) { serverParser.ScParse(reader) },
		clientConn, serverConn)

	// Set up for parsing messages from client to server
	clientToServerReportChan := spliceParser(
		func(reader io.Reader) { clientParser.CsParse(reader) },
		serverConn, clientConn)

	// Wait for the both relay/splices to stop, then we let the connections
	// close via deferred calls
	report := <-serverToClientReportChan
	log.Printf("(%s) Server->client relay after %d bytes with error: %v", clientAddr, report.written, report.err)
	report = <-clientToServerReportChan
	log.Printf("(%s) Client->server relay after %d bytes with error: %v", clientAddr, report.written, report.err)

	log.Printf("(%s) Client disconnected", clientAddr)
}

func serve(localaddr, remoteaddr string) (err os.Error) {
	listener, err := net.Listen("tcp", localaddr)
	if err != nil {
		log.Fatal("Listen: ", err.String())
		return
	}

	defer listener.Close()
	log.Print("Listening on ", localaddr)

	for {
		clientConn, acceptErr := listener.Accept()
		if acceptErr != nil {
			log.Printf("Accept error: %s", acceptErr.String())
		} else {
			go serveConn(clientConn, remoteaddr)
		}
	}
	return
}

func usage() {
	os.Stderr.WriteString("usage: " + os.Args[0] + " localaddr:port remoteaddr:port\n")
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
