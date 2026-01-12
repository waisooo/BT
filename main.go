package main

import (
	torrent "bittorrent/torrent"
	"crypto/rand"
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
	fmt.Printf("Total Pieces: %d\n", len(torrentFile.PiecesHash))
	fmt.Printf("Total Length %d\n", torrentFile.Info.Length)
	for i, pieceHash := range torrentFile.PiecesHash {
		fmt.Printf("Piece %d Hash: %x\n", i, pieceHash)
	}

	var peerId [20]byte

	_, err = rand.Read(peerId[:])
	if err != nil {
		panic(err)
	}

	peers := torrent.RequestPeers(torrentFile, peerId, 6881)
	fmt.Println(peers)

	// fmt.Println([]byte{peers["peers"]})
}
