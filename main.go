package main

import (
	torrent "bittorrent/torrent"
	"fmt"
)

func main() {
	torrentFilePath := "sample.torrent"

	torrentFile, err := torrent.ExtractTorrentInfo(torrentFilePath)
	if err != nil {
		panic(err)
	}

	torrent.CalculatePiecesHash(torrentFile)

	fmt.Printf("Info Hash: %x\n", torrentFile.InfoHash)
	for i, pieceHash := range torrentFile.PiecesHash {
		fmt.Printf("Piece %d Hash: %x\n", i, pieceHash)
	}

}
