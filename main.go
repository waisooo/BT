package main

import (
	torrent "bittorrent/torrent"
	"fmt"
	"net"
)

func main() {
	// Path to the torrent file
	torrentFilePath := "sample.torrent"

	// Parse the torrent file to extract metadata
	torrentFile, err := torrent.ExtractTorrentInfo(torrentFilePath)
	if err != nil {
		panic(err)
	}

	torrent.CalculatePiecesHash(torrentFile)

	peerId := torrent.GeneratePeerId()

	peers := torrent.RequestPeers(torrentFile, peerId, 6881)

	var conn net.Conn
	for _, peerIP := range peers.Peers {
		conn, err = net.Dial("tcp", peerIP)

		if err != nil {
			fmt.Printf("Failed to connect to peer %s: %v\n", peerIP, err)
			continue
		}

		fmt.Printf("Connected to peer %s\n", peerIP)

		_, err = torrent.HandShakePeer(conn, torrentFile.InfoHash, peerId)
		if err != nil {
			fmt.Printf("Handshake failed with peer %s: %v\n", peerIP, err)
			conn.Close()
			continue
		}

		conn.Close()

	}

}
