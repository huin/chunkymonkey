package main

import (
	_ "expvar"
	"flag"
	"http"
	"log"
	"net"
	"os"

	"chunkymonkey"
	"chunkymonkey/gamerules"
)

var addr = flag.String(
	"addr", ":25565",
	"Serves on the given address:port.")

var httpAddr = flag.String(
	"http_addr", ":25566",
	"Serves HTTP diagnostics on the given address:port.")

var blockDefs = flag.String(
	"blocks", "blocks.json",
	"The JSON file containing block type definitions.")

var itemDefs = flag.String(
	"items", "items.json",
	"The JSON file containing item type definitions.")

var recipeDefs = flag.String(
	"recipes", "recipes.json",
	"The JSON file containing recipe definitions.")

var underMaintenaceMsg = flag.String(
	"underMaintenanceMsg", "",
	"If set, all logins will be denied and this message will be given as reason.")


func usage() {
	os.Stderr.WriteString("usage: " + os.Args[0] + " [flags] <world>\n")
	flag.PrintDefaults()
}

func startHttpServer(addr string) (err os.Error) {
	httpPort, err := net.Listen("tcp", addr)
	if err != nil {
		return
	}
	go http.Serve(httpPort, nil)
	return
}

func main() {
	var err os.Error

	flag.Usage = usage
	flag.Parse()

	if flag.NArg() != 1 {
		flag.Usage()
		os.Exit(1)
	}

	gameRules, err := gamerules.LoadGameRules(*blockDefs, *itemDefs, *recipeDefs)
	if err != nil {
		log.Print("Error loading game rules: ", err)
		os.Exit(1)
	}

	worldPath := flag.Arg(0)
	fi, err := os.Stat(worldPath)
	if err != nil {
		log.Printf("Error loading world from directory %v: %v", worldPath, err)
		os.Exit(1)
	}
	if !fi.IsDirectory() {
		log.Printf("Error loading world %v: Not a directory", worldPath)
		os.Exit(1)
	}

	game, err := chunkymonkey.NewGame(worldPath, gameRules)
	if err != nil {
		log.Panic(err)
	}
	game.UnderMaintenanceMsg = *underMaintenaceMsg
	err = startHttpServer(*httpAddr)
	if err != nil {
		log.Panic(err)
	}

	game.Serve(*addr)
}
