package main

import (
	download "bittorrent/download"
	"log"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatalf("Please provide the path to the torrent file as a command-line argument. E.g., go run main.go /path/to/file.torrent")
	}

	torrentFilePath := os.Args[1]
	if _, err := os.Stat(torrentFilePath); os.IsNotExist(err) {
		log.Fatalf("The specified torrent file does not exist: %s", torrentFilePath)
	}

	download.DownloadFile(torrentFilePath)

}
