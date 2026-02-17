package main

import (
	"fmt"
	"os"

	"github.com/anthony/BT/download"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Please provide the path to the torrent file as a command-line argument. E.g. go run main.go /path/to/file.torrent")
		os.Exit(1)
	}

	torrentFilePath := os.Args[1]
	if _, err := os.Stat(torrentFilePath); os.IsNotExist(err) {
		fmt.Println(err)
		os.Exit(1)
	}

	download.DownloadFile(torrentFilePath)
}
