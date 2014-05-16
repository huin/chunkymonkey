package main

import (
	_ "expvar"
	"flag"
	"log"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"

	"chunkymonkey"
	"chunkymonkey/gamerules"
	"chunkymonkey/worldstore"
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

var furnaceDefs = flag.String(
	"furnace", "furnace.json",
	"The JSON file containing furnace fuel and reaction definitions.")

var serverDesc = flag.String(
	"server_desc", "Chunkymonkey Minecraft server",
	"The server description.")

var maintenanceMsg = flag.String(
	"maintenance_msg", "",
	"If set, all logins will be denied and this message will be given as reason.")

var userDefs = flag.String(
	"users", "users.json",
	"The JSON file container user permissions.")

var groupDefs = flag.String(
	"groups", "groups.json",
	"The JSON file containing group permissions.")

// TODO Implement max player count enforcement. Probably would have to be
// implemented atomically at the game level.
var maxPlayerCount = flag.Int(
	"max_player_count", 16,
	"Maximum number of players to allow concurrently. (Does not work yet)")

func usage() {
	os.Stderr.WriteString("usage: " + os.Args[0] + " [flags] <world>\n")
	flag.PrintDefaults()
}

func startHttpServer(addr string) (err error) {
	httpPort, err := net.Listen("tcp", addr)
	if err != nil {
		return
	}
	go http.Serve(httpPort, nil)
	return
}

func main() {
	var err error

	flag.Usage = usage
	flag.Parse()

	if flag.NArg() != 1 {
		flag.Usage()
		os.Exit(1)
	}

	err = gamerules.LoadGameRules(*blockDefs, *itemDefs, *recipeDefs, *furnaceDefs, *userDefs, *groupDefs)
	if err != nil {
		log.Print("Error loading game rules: ", err)
		os.Exit(1)
	}

	worldPath := flag.Arg(0)
	fi, err := os.Stat(worldPath)
	if err != nil {
		log.Printf("Could not load world from directory %v: %v", worldPath, err)
		log.Printf("Creating a new world in directory %v", worldPath)
		err = worldstore.CreateWorld(worldPath)
	}
	if err != nil {
		log.Printf("Error creating new world: %v", err)
	} else {
		fi, err = os.Stat(worldPath)
	}

	if fi == nil || !fi.IsDir() {
		log.Printf("Error loading world %v: Not a directory", worldPath)
		os.Exit(1)
	}

	listener, err := net.Listen("tcp", *addr)
	if err != nil {
		log.Fatal(err)
	}

	game, err := chunkymonkey.NewGame(worldPath, listener, *serverDesc, *maintenanceMsg, *maxPlayerCount)
	if err != nil {
		log.Fatal(err)
	}

	err = startHttpServer(*httpAddr)
	if err != nil {
		log.Fatal(err)
	}

	game.Serve()
}
