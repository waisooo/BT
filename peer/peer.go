package peer

import (
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/anthony/BT/message"
	"github.com/anthony/BT/piece"
	"github.com/anthony/BT/torrent"
)

type Peers struct {
	Peers []*message.Client
}

// DownloadFromPeers takes in a torrent file and peer id and attempts to download all the pieces of the file
// from available peers. Upon downloading all the pieces, it reconstructs the original file and writes it to disk.
func (p *Peers) DownloadFromPeers(tf *torrent.TorrentFile, peerId [20]byte) {
	// Initialise worker queue and file data channel
	workerQueue := make(chan piece.PieceWork, len(tf.PiecesHash))
	results := make(chan piece.PieceResult, len(tf.PiecesHash))

	fmt.Println("Starting download")
	fmt.Println("There are", len(p.Peers), "peers available for download")

	// Start download workers for each peer
	for _, client := range p.Peers {
		go func() {
			for pw := range workerQueue {
				data, err := piece.TryDownloadPiece(client, pw)
				if err != nil {
					workerQueue <- pw

					if err == io.EOF {
						break
					}
				}

				results <- piece.PieceResult{
					Index: pw.Index,
					Data:  data,
				}
			}

			// Close the connection when done
			client.Conn.Close()
		}()
	}

	// Send all the piece to the worker queue
	for i := 0; i < len(tf.PiecesHash); i++ {
		length := tf.Info.PieceLength
		if i == len(tf.PiecesHash)-1 {
			length = tf.Info.Length - (tf.Info.PieceLength * (len(tf.PiecesHash) - 1))
		}

		piece := piece.PieceWork{
			Index:     i,
			PieceHash: tf.PiecesHash[i],
			PieceSize: length,
		}
		workerQueue <- piece
	}

	// Combine pieces into final file data as they are downloaded
	finalData := make([]byte, tf.Info.Length)
	for i := 0; i < len(tf.PiecesHash); i++ {
		result := <-results

		copy(finalData[result.Index*tf.Info.PieceLength:], result.Data)
		fmt.Printf("%0.2f%% complete\n", float64(i)/float64(len(tf.PiecesHash))*100)
	}

	close(workerQueue)

	// Determine whether the file to be downloaded is a single file or multiple files and write to disk
	if len(tf.Info.Files) == 0 {
		err := os.WriteFile(tf.Info.Name, finalData, 0644)
		if err != nil {
			fmt.Printf("Error: Failed to write file to disk, %s\n", err)
			os.Exit(1)
		}
	} else {
		index := 0
		for _, path := range tf.Info.Files {
			dir := filepath.Dir(path.Path)
			err := os.MkdirAll(dir, 0755)
			if err != nil {
				fmt.Printf("Error: Failed to create directory, %s\n", err)
				os.Exit(1)
			}

			err = os.WriteFile(path.Path, finalData[index:path.Length], 0644)
			if err != nil {
				fmt.Printf("Error: Failed to write file to disk, %s\n", err)
				os.Exit(1)
			}

			index += path.Length
		}
	}

}

// RemoveDuplicatePeers takes in a list of peers and returns a new list with duplicate peers removed.
func RemoveDuplicatePeers(peers []net.TCPAddr) []net.TCPAddr {
	peerSet := make(map[string]bool)
	uniquePeers := []net.TCPAddr{}

	for _, peer := range peers {
		if !peerSet[peer.String()] {
			uniquePeers = append(uniquePeers, peer)
			peerSet[peer.String()] = true
		}
	}

	return uniquePeers
}

// HandShakePeer performs the BitTorrent handshake with a peer as specified by the BitTorrent protocol.
// It sends a handshake message to the peer and waits for a response.
//
// If the handshake is successful, it returns nil. Otherwise, it returns an error.
func HandShakePeer(conn net.Conn, infoHash [20]byte, peerId [20]byte) error {
	// Create handshake message
	message := make([]byte, 68)

	message[0] = 19                                    // Length of protocol
	copy(message[1:20], []byte("BitTorrent protocol")) // Protocol identifier
	copy(message[28:48], infoHash[:])                  // Info hash
	copy(message[48:68], peerId[:])                    // Peer ID

	// Send initial handshake
	_, err := conn.Write(message)
	if err != nil {
		return err
	}

	// Read handshake response from peer
	// It should be in the same format as the sent message
	response := make([]byte, 68)
	_, err = conn.Read(response)
	if err != nil {
		return err
	}

	return nil
}

// NewPeerClient attempts to establish a connection with the peer, perform the BitTorrent handshake, and
// recieve the initial bitfield message from the peer if it is sent.
//
// If the handshake is successful, it returns a new Client representing the peer. Otherwise, it returns an error.
func NewPeerClient(addr net.TCPAddr, hash [20]byte, peerId [20]byte, bitfieldLength int) (*message.Client, error) {
	conn, err := net.DialTimeout("tcp", addr.String(), 5*time.Second)
	if err != nil {
		return nil, fmt.Errorf("dialing: %w", err)
	}

	conn.SetDeadline(time.Now().Add(5 * time.Second))
	defer conn.SetDeadline(time.Time{})

	err = HandShakePeer(conn, hash, peerId)
	if err != nil {
		return nil, err
	}

	// Check if first message is bitfield
	msg, err := message.RecieveMessage(conn)
	if err != nil {
		return nil, err
	}

	bf := make([]byte, bitfieldLength)
	if msg.Id == message.Bitfield {
		bf = msg.Payload
	}

	return &message.Client{
		Ip:       addr.IP.String(),
		Conn:     conn,
		Bitfield: bf,
		Choked:   true,
	}, nil

}
