package main

import (
	torrent "bittorrent/torrent"
	"fmt"
	"net"
	"os"
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
	// for _, peerIP := range peers.Peers {
	peerIP := peers.Peers[0]
	conn, err = net.Dial("tcp", peerIP)

	if err != nil {
		fmt.Printf("Failed to connect to peer %s: %v\n", peerIP, err)
		conn.Close()
		os.Exit(1)
	}

	fmt.Printf("Connected to peer %s\n", peerIP)

	_, err = torrent.HandShakePeer(conn, torrentFile.InfoHash, peerId)
	if err != nil {
		fmt.Printf("Handshake failed with peer %s: %v\n", peerIP, err)
		conn.Close()
		os.Exit(1)

	}

	fmt.Printf("Handshake successful with peer %s\n", peerIP)
	fmt.Println("Pieces length is ", torrentFile.Info.PieceLength)

	client := &torrent.Client{
		PeerIP: peerIP,
		Conn:   conn,
		Choked: true,
	}

	pieceProgress := torrent.PieceProgress{
		Client:    client,
		Hash:      torrentFile.PiecesHash[1],
		Index:     0,
		Received:  0,
		Total:     torrentFile.Info.PieceLength,
		Requested: 0,
		BlockData: make([]byte, torrentFile.Info.PieceLength),
	}

	_, err = torrent.DownloadBlockFromPeer(&pieceProgress)
	if err != nil {
		panic(err)
	}

	if !torrent.ValidatePiece(pieceProgress.BlockData, pieceProgress.Hash) {
		panic(fmt.Errorf("piece validation failed for piece %d", pieceProgress.Index))
	}

	conn.Close()

}
