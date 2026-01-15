package main

import (
	torrent "bittorrent/torrent"
	"fmt"
	"net"
	"os"
)

type worker struct {
	index            int
	pieceHash        [20]byte
	downloadAttempts int // Number of download attempts that have been made
}

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

	// Create a worker queue for all the pieces to be downloaded
	workerQueue := []worker{}
	for i := 0; i < len(torrentFile.PiecesHash); i++ {
		piece := worker{
			index:            i,
			pieceHash:        torrentFile.PiecesHash[i],
			downloadAttempts: 0,
		}
		workerQueue = append(workerQueue, piece)
	}

	peerConnections := []torrent.Client{}
	for _, peerIP := range peers.Peers {
		conn, err := net.Dial("tcp", peerIP)
		if err != nil {
			fmt.Printf("Failed to connect to peer %s: %v\n", peerIP, err)
			continue
		}

		_, err = torrent.HandShakePeer(conn, torrentFile.InfoHash, peerId)
		if err != nil {
			fmt.Printf("Handshake failed with peer %s: %v\n", conn.RemoteAddr().String(), err)
			conn.Close()
			continue
		}

		fmt.Printf("Handshake successful with peer %s\n", conn.RemoteAddr().String())

		client := torrent.Client{
			PeerIP: peerIP,
			Conn:   conn,
			Choked: true,
		}

		peerConnections = append(peerConnections, client)
	}

	peerIndex := 0
	fileData := make([]byte, torrentFile.Info.Length)

	for len(workerQueue) != 0 {
		piece := workerQueue[0]
		workerQueue = workerQueue[1:]

		if piece.downloadAttempts >= 5 {
			fmt.Printf("Piece %d has failed to download after multiple attempts. Exitting the program\n", piece.index)
			os.Exit(1)
		}

		// Choosing peer in round-robin fashion
		client := peerConnections[peerIndex%len(peerConnections)]
		peerIndex++

		// Set up piece download state
		pieceProgress := torrent.ConstructPieceProgress(&client, *torrentFile)

		// Attempt to download the piece
		_, err = torrent.TryDownloadPiece(pieceProgress)
		if err != nil {
			fmt.Printf("Failed to download piece %d from peer %s: %v\n", piece.index, client.PeerIP, err)
			workerQueue = append(workerQueue, piece)
			workerQueue[len(workerQueue)-1].downloadAttempts++
			continue
		}

		if !torrent.ValidatePiece(pieceProgress.BlockData, pieceProgress.Hash) {
			fmt.Printf("Piece %d validation failed from peer %s\n", piece.index, client.PeerIP)
			workerQueue = append(workerQueue, piece)
			workerQueue[len(workerQueue)-1].downloadAttempts++
			continue
		}

		fmt.Printf("Successfully downloaded and validated piece %d from peer %s\n", piece.index, client.PeerIP)
		copy(fileData[piece.index*torrentFile.Info.PieceLength:], pieceProgress.BlockData)
	}

	// Clean up connections
	for _, client := range peerConnections {
		client.Conn.Close()
	}

	// Save the downloaded file
	outputFile, err := os.Create(torrentFile.Info.Name)
	if err != nil {
		panic(err)
	}
	defer outputFile.Close()

	_, err = outputFile.Write(fileData)
	if err != nil {
		panic(err)
	}
}
